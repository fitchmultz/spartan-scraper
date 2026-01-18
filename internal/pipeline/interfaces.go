package pipeline

type Plugin interface {
	Name() string
	Stages() []Stage
	Priority() int
	Enabled(target Target, opts Options) bool

	PreFetch(ctx HookContext, in FetchInput) (FetchInput, error)
	PostFetch(ctx HookContext, in FetchInput, out FetchOutput) (FetchOutput, error)

	PreExtract(ctx HookContext, in ExtractInput) (ExtractInput, error)
	PostExtract(ctx HookContext, in ExtractInput, out ExtractOutput) (ExtractOutput, error)

	PreOutput(ctx HookContext, in OutputInput) (OutputInput, error)
	PostOutput(ctx HookContext, in OutputInput, out OutputOutput) (OutputOutput, error)
}

type BasePlugin struct{}

func (BasePlugin) Stages() []Stage {
	return AllStages()
}

func (BasePlugin) Priority() int {
	return 0
}

func (BasePlugin) Enabled(_ Target, _ Options) bool {
	return true
}

func (BasePlugin) PreFetch(_ HookContext, in FetchInput) (FetchInput, error) {
	return in, nil
}

func (BasePlugin) PostFetch(_ HookContext, _ FetchInput, out FetchOutput) (FetchOutput, error) {
	return out, nil
}

func (BasePlugin) PreExtract(_ HookContext, in ExtractInput) (ExtractInput, error) {
	return in, nil
}

func (BasePlugin) PostExtract(_ HookContext, _ ExtractInput, out ExtractOutput) (ExtractOutput, error) {
	return out, nil
}

func (BasePlugin) PreOutput(_ HookContext, in OutputInput) (OutputInput, error) {
	return in, nil
}

func (BasePlugin) PostOutput(_ HookContext, _ OutputInput, out OutputOutput) (OutputOutput, error) {
	return out, nil
}
