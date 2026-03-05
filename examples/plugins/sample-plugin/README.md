# Sample Spartan Scraper Plugin

This is a sample WASM plugin for Spartan Scraper that demonstrates how to extend the pipeline with custom hooks.

## What This Plugin Does

This sample plugin adds a custom HTTP header (`X-Sample-Plugin: enabled`) to all outgoing fetch requests. This is useful for:
- Identifying requests that come from Spartan Scraper
- Working with servers that require custom headers
- Demonstrating the plugin system capabilities

## Plugin Structure

```
sample-plugin/
├── manifest.json    # Plugin metadata and configuration
├── main.go          # Plugin source code (Go/TinyGo)
├── plugin.wasm      # Compiled WASM binary (you need to build this)
└── README.md        # This file
```

## Building the Plugin

### Prerequisites

- [TinyGo](https://tinygo.org/) (recommended for smaller WASM binaries)
- Or Go 1.21+ with `GOOS=js GOARCH=wasm`

### Build with TinyGo (Recommended)

```bash
cd examples/plugins/sample-plugin
tinygo build -o plugin.wasm -target wasm .
```

### Build with Go

```bash
cd examples/plugins/sample-plugin
GOOS=js GOARCH=wasm go build -o plugin.wasm .
```

## Installing the Plugin

### Via CLI

```bash
# Install the plugin
spartan plugin install --path ./examples/plugins/sample-plugin/

# Enable the plugin
spartan plugin enable --name sample-plugin
```

### Via API

```bash
# Install
curl -X POST http://localhost:8080/v1/plugins \
  -H "Content-Type: application/json" \
  -H "X-API-Key: ${SPARTAN_API_KEY}" \
  -d '{"source": "/path/to/sample-plugin"}'

# Enable
curl -X POST http://localhost:8080/v1/plugins/sample-plugin/enable \
  -H "X-API-Key: ${SPARTAN_API_KEY}"
```

## Configuration

The plugin can be configured via the CLI or API:

```bash
# Set custom header name
spartan plugin configure --name sample-plugin --key header_name --value "X-Custom-Header"

# Set custom header value
spartan plugin configure --name sample-plugin --key header_value --value "my-value"
```

## How It Works

1. **Manifest**: The `manifest.json` file defines the plugin metadata, hooks, and permissions.
2. **WASM Binary**: The compiled `plugin.wasm` contains the plugin logic.
3. **Hooks**: The plugin exports a `pre_fetch` function that modifies the request before it's sent.

## Plugin API

### Host Functions Available

Plugins can call these host functions (based on granted permissions):

- `log(msg_ptr, msg_len)` - Log messages to the host
- `get_config(key_ptr, key_len) -> value_ptr_with_len` - Get configuration values
- `http_request(...)` - Make HTTP requests (requires `network` permission)
- `file_access(...)` - Access files (requires `filesystem` permission)
- `get_env(key_ptr, key_len)` - Get environment variables (requires `env` permission)

### Hook Functions

Plugins must export functions matching the hooks they declare in their manifest:

```go
//export pre_fetch
func pre_fetch(inputPtr int32, inputLen int32) int64

//export post_fetch  
func post_fetch(inputPtr int32, inputLen int32) int64

// etc.
```

Input/output is JSON-encoded. The return value encodes the output pointer and length.

## Security

- Plugins run in a sandboxed WASM runtime (wazero)
- Access to host resources requires explicit permissions
- Filesystem access is restricted to the plugin's directory
- Environment variables must start with `SPARTAN_PLUGIN_`

## Development Tips

1. **Start Simple**: Begin with a single hook and minimal functionality
2. **Test Locally**: Use the CLI to test plugins before distributing
3. **Handle Errors**: Always return 0 from hook functions on error
4. **Memory Management**: Use the exported `malloc`/`free` functions
5. **JSON Encoding**: Be careful with JSON serialization/deserialization

## See Also

- [Plugin System Documentation](../../docs/architecture.md)
- [CLI Usage Guide](../../docs/usage.md)
- [Wazero Documentation](https://wazero.io/)
