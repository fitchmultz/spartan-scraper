package config

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
)

type testHandler struct {
	buf    *bytes.Buffer
	inner  slog.Handler
	output string
}

func (h *testHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *testHandler) Handle(ctx context.Context, r slog.Record) error {
	if err := h.inner.Handle(ctx, r); err != nil {
		return err
	}
	h.output = h.buf.String()
	return nil
}

func (h *testHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &testHandler{
		buf:    h.buf,
		inner:  h.inner.WithAttrs(attrs),
		output: h.output,
	}
}

func (h *testHandler) WithGroup(name string) slog.Handler {
	return &testHandler{
		buf:    h.buf,
		inner:  h.inner.WithGroup(name),
		output: h.output,
	}
}

func isJSONOutput(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

func isTextOutput(s string) bool {
	return !isJSONOutput(s)
}
