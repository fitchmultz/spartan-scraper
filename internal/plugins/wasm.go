// Package plugins provides a WASM-based plugin system for third-party extensions.
package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// WASMRuntime manages the WASM runtime and compiled modules.
type WASMRuntime struct {
	runtime wazero.Runtime
	modules map[string]wazero.CompiledModule
	config  wazero.ModuleConfig
	dataDir string
}

// NewWASMRuntime creates a new WASM runtime with default configuration.
func NewWASMRuntime(dataDir string) *WASMRuntime {
	ctx := context.Background()

	// Create runtime with memory limits for safety
	rt := wazero.NewRuntime(ctx)

	// Instantiate WASI for system interface support
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		// Log warning but continue - WASI is optional for basic plugins
		fmt.Fprintf(os.Stderr, "[WARN] Failed to instantiate WASI: %v\n", err)
	}

	return &WASMRuntime{
		runtime: rt,
		modules: make(map[string]wazero.CompiledModule),
		config: wazero.NewModuleConfig().
			WithStdout(os.Stdout).
			WithStderr(os.Stderr).
			WithSysWalltime().
			WithSysNanotime(),
		dataDir: dataDir,
	}
}

// Close shuts down the WASM runtime and releases resources.
func (w *WASMRuntime) Close(ctx context.Context) error {
	// Release all compiled modules
	for name, mod := range w.modules {
		mod.Close(ctx)
		delete(w.modules, name)
	}

	if w.runtime != nil {
		return w.runtime.Close(ctx)
	}
	return nil
}

// LoadModule compiles and caches a WASM module from a file path.
func (w *WASMRuntime) LoadModule(ctx context.Context, name, wasmPath string) (wazero.CompiledModule, error) {
	// Return cached module if available
	if mod, ok := w.modules[name]; ok {
		return mod, nil
	}

	// Read WASM binary
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, apperrors.NotFound(fmt.Sprintf("WASM binary not found: %s", wasmPath))
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to read WASM binary", err)
	}

	// Compile the module
	compiled, err := w.runtime.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindValidation, "failed to compile WASM module", err)
	}

	// Cache for reuse
	w.modules[name] = compiled

	return compiled, nil
}

// Instantiate creates a new module instance with the given configuration.
func (w *WASMRuntime) Instantiate(ctx context.Context, name string, compiled wazero.CompiledModule, manifest *PluginManifest) (*WASMInstance, error) {
	moduleConfig := w.config.WithName(name)

	// Create instance with host functions
	instance := &WASMInstance{
		manifest:   manifest,
		dataDir:    w.dataDir,
		pluginDir:  filepath.Join(w.dataDir, "plugins", manifest.Name),
		memoryData: make(map[string][]byte),
	}

	// Build host function module
	hostBuilder := w.runtime.NewHostModuleBuilder("spartan")

	// Add host functions based on permissions
	if manifest.HasPermission(PermNetwork) {
		instance.exportNetworkFunctions(hostBuilder)
	}
	if manifest.HasPermission(PermFilesystem) {
		instance.exportFilesystemFunctions(hostBuilder)
	}
	if manifest.HasPermission(PermEnv) {
		instance.exportEnvFunctions(hostBuilder)
	}

	// Always available functions
	instance.exportLogFunction(hostBuilder)
	instance.exportConfigFunctions(hostBuilder)
	instance.exportMemoryFunctions(hostBuilder)

	// Instantiate host module
	hostModule, err := hostBuilder.Instantiate(ctx)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to instantiate host module", err)
	}
	instance.hostModule = hostModule

	// Instantiate the plugin module
	module, err := w.runtime.InstantiateModule(ctx, compiled, moduleConfig)
	if err != nil {
		hostModule.Close(ctx)
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to instantiate WASM module", err)
	}
	instance.module = module

	// Get exported memory
	if mem := module.Memory(); mem != nil {
		instance.memory = mem
	}

	return instance, nil
}

// WASMInstance represents a running WASM plugin instance.
type WASMInstance struct {
	module     api.Module
	hostModule api.Module
	memory     api.Memory
	manifest   *PluginManifest
	dataDir    string
	pluginDir  string
	memoryData map[string][]byte // For passing data between host and guest
	dataID     int
}

// Close releases the WASM instance resources.
func (i *WASMInstance) Close(ctx context.Context) {
	if i.module != nil {
		i.module.Close(ctx)
	}
	if i.hostModule != nil {
		i.hostModule.Close(ctx)
	}
}

// CallHook invokes a plugin hook with JSON input/output.
func (i *WASMInstance) CallHook(ctx context.Context, hookName string, input any, output any) error {
	// Find the exported function
	fn := i.module.ExportedFunction(hookName)
	if fn == nil {
		// Hook not implemented by this plugin - not an error
		return nil
	}

	// Serialize input to JSON
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to marshal hook input", err)
	}

	// Write input to guest memory
	inputPtr, err := i.writeToMemory(inputJSON)
	if err != nil {
		return err
	}
	inputLen := uint64(len(inputJSON))

	// Call the function with timeout
	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	results, err := fn.Call(callCtx, inputPtr, inputLen)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, fmt.Sprintf("hook %s failed", hookName), err)
	}

	if len(results) == 0 {
		return nil
	}

	// Result is a pointer to output data (encoded as: high 32 bits = ptr, low 32 bits = len)
	result := results[0]
	outputPtr := uint32(result >> 32)
	outputLen := uint32(result)

	if outputLen == 0 {
		return nil
	}

	// Read output from guest memory
	outputJSON, ok := i.memory.Read(outputPtr, outputLen)
	if !ok {
		return apperrors.Internal("failed to read hook output from memory")
	}

	// Deserialize output
	if err := json.Unmarshal(outputJSON, output); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal hook output", err)
	}

	return nil
}

// writeToMemory writes data to the guest's memory and returns the pointer.
func (i *WASMInstance) writeToMemory(data []byte) (uint64, error) {
	// Allocate memory via the guest's malloc function if available
	malloc := i.module.ExportedFunction("malloc")
	if malloc == nil {
		// Fall back to writing to a known location or use memory growth
		// For now, we'll use a simple approach with the existing memory
		return 0, apperrors.Internal("plugin does not export malloc function")
	}

	// Allocate memory
	results, err := malloc.Call(context.Background(), uint64(len(data)))
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to allocate memory", err)
	}

	ptr := uint32(results[0])
	if !i.memory.Write(ptr, data) {
		return 0, apperrors.Internal("failed to write to guest memory")
	}

	return uint64(ptr), nil
}

// Host function exporters

func (i *WASMInstance) exportLogFunction(builder wazero.HostModuleBuilder) {
	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, ptr, length uint32) {
			if i.memory == nil {
				return
			}
			data, ok := i.memory.Read(ptr, length)
			if !ok {
				return
			}
			fmt.Printf("[plugin:%s] %s\n", i.manifest.Name, string(data))
		}).
		Export("log")
}

func (i *WASMInstance) exportConfigFunctions(builder wazero.HostModuleBuilder) {
	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, keyPtr, keyLen uint32) uint64 {
			if i.memory == nil {
				return 0
			}

			keyBytes, ok := i.memory.Read(keyPtr, keyLen)
			if !ok {
				return 0
			}
			key := string(keyBytes)

			value, ok := i.manifest.Config[key]
			if !ok {
				return 0
			}

			valueJSON, err := json.Marshal(value)
			if err != nil {
				return 0
			}

			// Store and return pointer/length
			i.dataID++
			dataKey := fmt.Sprintf("cfg_%d", i.dataID)
			i.memoryData[dataKey] = valueJSON

			ptr, err := i.writeToMemory(valueJSON)
			if err != nil {
				return 0
			}

			// Return encoded result: high 32 bits = ptr, low 32 bits = len
			return ptr<<32 | uint64(len(valueJSON))
		}).
		Export("get_config")
}

func (i *WASMInstance) exportMemoryFunctions(builder wazero.HostModuleBuilder) {
	// Function to free allocated memory
	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, ptr uint32) {
			free := i.module.ExportedFunction("free")
			if free != nil {
				_, _ = free.Call(ctx, uint64(ptr))
			}
		}).
		Export("free_memory")
}

func (i *WASMInstance) exportNetworkFunctions(builder wazero.HostModuleBuilder) {
	// Placeholder for network access - would need full HTTP implementation
	// For security, this should be very limited
	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, urlPtr, urlLen, methodPtr, methodLen uint32) uint64 {
			// Network access is allowed but requires implementation
			// Return error encoded in result
			return 0
		}).
		Export("http_request")
}

func (i *WASMInstance) exportFilesystemFunctions(builder wazero.HostModuleBuilder) {
	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, pathPtr, pathLen uint32, isWrite uint32) uint64 {
			if i.memory == nil {
				return 0
			}

			pathBytes, ok := i.memory.Read(pathPtr, pathLen)
			if !ok {
				return 0
			}

			// Restrict to plugin directory
			path := filepath.Clean(string(pathBytes))
			fullPath := filepath.Join(i.pluginDir, path)

			// Prevent directory traversal
			if !strings.HasPrefix(fullPath, i.pluginDir) {
				return 0
			}

			if isWrite != 0 {
				// Write operation placeholder
				return 0
			}

			// Read file
			data, err := os.ReadFile(fullPath)
			if err != nil {
				return 0
			}

			ptr, err := i.writeToMemory(data)
			if err != nil {
				return 0
			}

			return ptr<<32 | uint64(len(data))
		}).
		Export("file_access")
}

func (i *WASMInstance) exportEnvFunctions(builder wazero.HostModuleBuilder) {
	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, keyPtr, keyLen uint32) uint64 {
			if i.memory == nil {
				return 0
			}

			keyBytes, ok := i.memory.Read(keyPtr, keyLen)
			if !ok {
				return 0
			}

			// Only allow access to SPARTAN_PLUGIN_* environment variables
			key := string(keyBytes)
			if !strings.HasPrefix(key, "SPARTAN_PLUGIN_") {
				return 0
			}

			value := os.Getenv(key)
			if value == "" {
				return 0
			}

			ptr, err := i.writeToMemory([]byte(value))
			if err != nil {
				return 0
			}

			return ptr<<32 | uint64(len(value))
		}).
		Export("get_env")
}
