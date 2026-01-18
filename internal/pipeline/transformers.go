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
