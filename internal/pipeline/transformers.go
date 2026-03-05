// Package pipeline provides a plugin system for extending scrape and crawl workflows.
// It handles plugin hooks at pre/post stages of fetch, extract, and output operations,
// plugin registration, and JavaScript plugin execution.
// It does NOT handle workflow execution or plugin implementations.
package pipeline

type Transformer interface {
	Name() string
	Priority() int
	Enabled(target Target, opts Options) bool
	Transform(ctx HookContext, in OutputInput) (OutputOutput, error)
}

type BaseTransformer struct{}

func (BaseTransformer) Priority() int {
	return 0
}

func (BaseTransformer) Enabled(_ Target, _ Options) bool {
	return true
}

func (BaseTransformer) Transform(_ HookContext, in OutputInput) (OutputOutput, error) {
	return OutputOutput{
		Raw:        in.Raw,
		Structured: in.Structured,
	}, nil
}
