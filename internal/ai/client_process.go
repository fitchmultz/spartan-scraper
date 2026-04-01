// Package ai manages the bridge process used for pi-backed LLM operations.
//
// Purpose:
// - Manage the pi bridge subprocess lifecycle and JSONL request transport.
//
// Responsibilities:
// - Start and stop the bridge, stream stderr, send framed requests, resolve the bridge script path,
// - and enforce startup-time health validation.
//
// Scope:
// - Process transport and path-resolution helpers only; public client operations live elsewhere.
//
// Usage:
// - Called internally by `Client.call` and startup/cleanup paths.
//
// Invariants/Assumptions:
// - The bridge speaks one JSON response line per request.
// - Process state is only mutated while the client mutexes are held.
package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

func (c *Client) ensureStarted(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cmd != nil && c.cmd.Process != nil {
		return nil
	}

	scriptPath, err := c.resolveBridgeScriptPath()
	if err != nil {
		return err
	}

	cmd := exec.Command(c.cfg.NodeBin, scriptPath)
	cmd.Env = append(os.Environ(), c.bridgeEnv()...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open bridge stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open bridge stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("open bridge stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start bridge process: %w", err)
	}

	go streamBridgeStderr(stderr)

	c.cmd = cmd
	c.stdin = stdin
	c.stdout = bufio.NewReader(stdout)

	startupCtx, cancel := withConfiguredTimeout(ctx, time.Duration(c.cfg.StartupTimeoutSecs)*time.Second)
	defer cancel()

	c.reqMu.Lock()
	defer c.reqMu.Unlock()

	resp, err := c.sendRequest(startupCtx, OperationHealth, nil)
	if err != nil {
		_ = c.stopLocked()
		return fmt.Errorf("wait for bridge health: %w", err)
	}
	if !resp.OK {
		_ = c.stopLocked()
		if resp.Error == nil {
			return fmt.Errorf("bridge health check failed")
		}
		if resp.Error.Code != "" {
			return fmt.Errorf("bridge %s: %s", resp.Error.Code, resp.Error.Message)
		}
		return fmt.Errorf("bridge error: %s", resp.Error.Message)
	}

	var health HealthResponse
	if len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, &health); err != nil {
			_ = c.stopLocked()
			return fmt.Errorf("decode bridge health: %w", err)
		}
	}
	logBridgeHealth(health)
	if err := validateBridgeHealth(health); err != nil {
		_ = c.stopLocked()
		return &BridgeHealthError{Health: health, Err: err}
	}
	return nil
}

func (c *Client) bridgeEnv() []string {
	env := []string{
		"PI_MODE=" + c.cfg.Mode,
	}
	if c.cfg.ConfigPath != "" {
		env = append(env, "PI_CONFIG_PATH="+c.cfg.ConfigPath)
	}
	return env
}

func (c *Client) resetProcess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.stopLocked()
}

func (c *Client) resetProcessOnFatal(err *bridgeError) {
	if err == nil {
		return
	}
	if strings.EqualFold(err.Code, "bad_request") {
		return
	}
	c.resetProcess()
}

func (c *Client) stopLocked() error {
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	var waitErr error
	if c.cmd != nil {
		if c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
		}
		waitErr = c.cmd.Wait()
	}
	c.cmd = nil
	c.stdin = nil
	c.stdout = nil
	if waitErr != nil && strings.Contains(waitErr.Error(), "signal: killed") {
		return nil
	}
	return waitErr
}

func streamBridgeStderr(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		slog.Debug("pi bridge", "stderr", scanner.Text())
	}
}

func (c *Client) sendRequest(ctx context.Context, op string, payload interface{}) (responseEnvelope, error) {
	req := requestEnvelope{
		ID:      fmt.Sprintf("req-%d", atomic.AddUint64(&c.nextID, 1)),
		Op:      op,
		Payload: payload,
	}

	line, err := json.Marshal(req)
	if err != nil {
		return responseEnvelope{}, fmt.Errorf("marshal bridge request: %w", err)
	}
	line = append(line, '\n')

	if _, err := c.stdin.Write(line); err != nil {
		return responseEnvelope{}, fmt.Errorf("write bridge request: %w", err)
	}

	respCh := make(chan responseEnvelope, 1)
	errCh := make(chan error, 1)
	go func() {
		raw, err := c.stdout.ReadBytes('\n')
		if err != nil {
			errCh <- fmt.Errorf("read bridge response: %w", err)
			return
		}
		var resp responseEnvelope
		if err := json.Unmarshal(raw, &resp); err != nil {
			errCh <- fmt.Errorf("decode bridge response: %w", err)
			return
		}
		respCh <- resp
	}()

	select {
	case <-ctx.Done():
		return responseEnvelope{}, ctx.Err()
	case err := <-errCh:
		return responseEnvelope{}, err
	case resp := <-respCh:
		return resp, nil
	}
}

func (c *Client) resolveBridgeScriptPath() (string, error) {
	wd, _ := os.Getwd()
	executablePath, _ := os.Executable()
	return resolveBridgeScriptPath(
		c.cfg.BridgeScript,
		bridgeScriptSearchRoots(wd, executablePath, c.cfg.ConfigPath),
	)
}

func bridgeScriptSearchRoots(workingDir string, executablePath string, configPath string) []string {
	roots := make([]string, 0, 4)
	if configPath != "" {
		roots = append(roots, filepath.Dir(configPath))
	}
	if workingDir != "" {
		roots = append(roots, workingDir)
	}
	if executablePath != "" {
		executableDir := filepath.Dir(executablePath)
		roots = append(roots, executableDir, filepath.Join(executableDir, ".."))
	}
	return roots
}

func resolveBridgeScriptPath(scriptPath string, searchRoots []string) (string, error) {
	if filepath.IsAbs(scriptPath) {
		return scriptPath, nil
	}

	seen := make(map[string]struct{}, len(searchRoots))
	for _, root := range searchRoots {
		if root == "" {
			continue
		}
		candidate := filepath.Clean(filepath.Join(root, scriptPath))
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf(
		"resolve bridge script path %q: not found relative to PI_CONFIG_PATH, cwd, or executable; set PI_BRIDGE_SCRIPT to an absolute path",
		scriptPath,
	)
}

func withConfiguredTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	if deadline, ok := ctx.Deadline(); ok {
		if time.Until(deadline) <= timeout {
			return ctx, func() {}
		}
	}
	return context.WithTimeout(ctx, timeout)
}
