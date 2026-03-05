// Package pipeline provides a plugin system for extending scrape and crawl workflows.
// It handles plugin hooks at pre/post stages of fetch, extract, and output operations,
// plugin registration, and JavaScript plugin execution.
// It does NOT handle workflow execution or plugin implementations.
package pipeline

import (
	"encoding/json"
	"fmt"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/jmespath/go-jmespath"
)

// JMESPathTransformer applies JMESPath expressions to structured data.
// It compiles the expression at creation time for efficient reuse.
type JMESPathTransformer struct {
	BaseTransformer
	expression string
	compiled   *jmespath.JMESPath
}

// JMESPathOption configures a JMESPathTransformer.
type JMESPathOption func(*JMESPathTransformer)

// WithJMESPathExpression sets the JMESPath expression.
func WithJMESPathExpression(expression string) JMESPathOption {
	return func(t *JMESPathTransformer) {
		t.expression = expression
	}
}

// NewJMESPathTransformer creates a transformer with a compiled JMESPath expression.
// Returns an error if the expression cannot be compiled.
func NewJMESPathTransformer(opts ...JMESPathOption) (*JMESPathTransformer, error) {
	t := &JMESPathTransformer{}
	for _, opt := range opts {
		opt(t)
	}

	if t.expression != "" {
		compiled, err := jmespath.Compile(t.expression)
		if err != nil {
			return nil, apperrors.Wrap(
				apperrors.KindValidation,
				fmt.Sprintf("invalid JMESPath expression: %s", err.Error()),
				err,
			)
		}
		t.compiled = compiled
	}

	return t, nil
}

// Name returns "jmespath".
func (t *JMESPathTransformer) Name() string {
	return "jmespath"
}

// Priority returns 50 (runs after validator at 100, before default transforms).
func (t *JMESPathTransformer) Priority() int {
	return 50
}

// Enabled checks if expression is non-empty.
func (t *JMESPathTransformer) Enabled(_ Target, opts Options) bool {
	if t.expression != "" {
		return true
	}
	if opts.JMESPath != "" {
		return true
	}
	return false
}

// Transform applies the JMESPath expression to the structured data.
// The expression is applied to the Structured field of the input.
func (t *JMESPathTransformer) Transform(_ HookContext, in OutputInput) (OutputOutput, error) {
	if in.Structured == nil {
		return OutputOutput{
			Raw:        in.Raw,
			Structured: nil,
		}, nil
	}

	// Use instance expression if set, otherwise check options
	expression := t.expression
	if expression == "" {
		// Try to get from context options if available
		// This would require passing options through HookContext
		// For now, pass through unchanged
		return OutputOutput{
			Raw:        in.Raw,
			Structured: in.Structured,
		}, nil
	}

	// Compile if not already compiled (when using options-based expression)
	compiled := t.compiled
	if compiled == nil && expression != "" {
		var err error
		compiled, err = jmespath.Compile(expression)
		if err != nil {
			return OutputOutput{}, apperrors.Wrap(
				apperrors.KindValidation,
				fmt.Sprintf("invalid JMESPath expression: %s", err.Error()),
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

	// Apply the JMESPath expression
	result, err := compiled.Search(in.Structured)
	if err != nil {
		return OutputOutput{}, apperrors.Wrap(
			apperrors.KindInternal,
			"JMESPath transformation failed",
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

	var cleanResult interface{}
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

// CompileJMESPath validates a JMESPath expression without executing it.
// Returns an error if the expression is invalid.
func CompileJMESPath(expression string) error {
	_, err := jmespath.Compile(expression)
	return err
}

// ApplyJMESPath applies a JMESPath expression to arbitrary data.
// This is a convenience function for one-off transformations.
func ApplyJMESPath(expression string, data interface{}) (interface{}, error) {
	compiled, err := jmespath.Compile(expression)
	if err != nil {
		return nil, apperrors.Wrap(
			apperrors.KindValidation,
			fmt.Sprintf("invalid JMESPath expression: %s", err.Error()),
			err,
		)
	}

	result, err := compiled.Search(data)
	if err != nil {
		return nil, apperrors.Wrap(
			apperrors.KindInternal,
			"JMESPath transformation failed",
			err,
		)
	}

	return result, nil
}
