// Package pipeline provides a plugin system for extending scrape and crawl workflows.
// It handles plugin hooks at pre/post stages of fetch, extract, and output operations,
// plugin registration, and JavaScript plugin execution.
// It does NOT handle workflow execution or plugin implementations.
package pipeline

import (
	"encoding/json"
	"fmt"

	"github.com/blues/jsonata-go"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// JSONataTransformer applies JSONata expressions to structured data.
// JSONata is a more powerful query and transformation language than JMESPath,
// supporting complex transformations, aggregations, and custom functions.
type JSONataTransformer struct {
	BaseTransformer
	expression string
	compiled   *jsonata.Expr
}

// JSONataOption configures a JSONataTransformer.
type JSONataOption func(*JSONataTransformer)

// WithJSONataExpression sets the JSONata expression.
func WithJSONataExpression(expression string) JSONataOption {
	return func(t *JSONataTransformer) {
		t.expression = expression
	}
}

// NewJSONataTransformer creates a transformer with a compiled JSONata expression.
// Returns an error if the expression cannot be compiled.
func NewJSONataTransformer(opts ...JSONataOption) (*JSONataTransformer, error) {
	t := &JSONataTransformer{}
	for _, opt := range opts {
		opt(t)
	}

	if t.expression != "" {
		compiled, err := jsonata.Compile(t.expression)
		if err != nil {
			return nil, apperrors.Wrap(
				apperrors.KindValidation,
				fmt.Sprintf("invalid JSONata expression: %s", err.Error()),
				err,
			)
		}
		t.compiled = compiled
	}

	return t, nil
}

// Name returns "jsonata".
func (t *JSONataTransformer) Name() string {
	return "jsonata"
}

// Priority returns 50 (runs after validator at 100, same priority as JMESPath).
func (t *JSONataTransformer) Priority() int {
	return 50
}

// Enabled checks if expression is non-empty.
func (t *JSONataTransformer) Enabled(_ Target, opts Options) bool {
	if t.expression != "" {
		return true
	}
	if opts.JSONata != "" {
		return true
	}
	return false
}

// Transform applies the JSONata expression to the structured data.
// The expression is applied to the Structured field of the input.
func (t *JSONataTransformer) Transform(_ HookContext, in OutputInput) (OutputOutput, error) {
	if in.Structured == nil {
		return OutputOutput{
			Raw:        in.Raw,
			Structured: nil,
		}, nil
	}

	// Use instance expression if set
	expression := t.expression
	if expression == "" {
		// No expression available, pass through unchanged
		return OutputOutput{
			Raw:        in.Raw,
			Structured: in.Structured,
		}, nil
	}

	// Compile if not already compiled
	compiled := t.compiled
	if compiled == nil && expression != "" {
		var err error
		compiled, err = jsonata.Compile(expression)
		if err != nil {
			return OutputOutput{}, apperrors.Wrap(
				apperrors.KindValidation,
				fmt.Sprintf("invalid JSONata expression: %s", err.Error()),
				err,
			)
		}
	}

	if compiled == nil {
		return OutputOutput{
			Raw:        in.Raw,
			Structured: in.Structured,
		}, nil
	}

	// Apply the JSONata expression
	result, err := compiled.Eval(in.Structured)
	if err != nil {
		return OutputOutput{}, apperrors.Wrap(
			apperrors.KindInternal,
			"JSONata transformation failed",
			err,
		)
	}

	// Convert result to JSON and back to ensure clean serialization
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return OutputOutput{}, apperrors.Wrap(
			apperrors.KindInternal,
			"failed to serialize transformation result",
			err,
		)
	}

	var cleanResult any
	if err := json.Unmarshal(jsonBytes, &cleanResult); err != nil {
		return OutputOutput{}, apperrors.Wrap(
			apperrors.KindInternal,
			"failed to deserialize transformation result",
			err,
		)
	}

	return OutputOutput{
		Raw:        jsonBytes,
		Structured: cleanResult,
	}, nil
}

// CompileJSONata validates a JSONata expression without executing it.
// Returns an error if the expression is invalid.
func CompileJSONata(expression string) error {
	_, err := jsonata.Compile(expression)
	return err
}

// ApplyJSONata applies a JSONata expression to arbitrary data.
// This is a convenience function for one-off transformations.
func ApplyJSONata(expression string, data any) (any, error) {
	compiled, err := jsonata.Compile(expression)
	if err != nil {
		return nil, apperrors.Wrap(
			apperrors.KindValidation,
			fmt.Sprintf("invalid JSONata expression: %s", err.Error()),
			err,
		)
	}

	result, err := compiled.Eval(data)
	if err != nil {
		return nil, apperrors.Wrap(
			apperrors.KindInternal,
			"JSONata transformation failed",
			err,
		)
	}

	return result, nil
}
