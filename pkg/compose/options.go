package compose

type Option func(*Engine)

func WithConfig(cfg *Config) Option {
	return func(e *Engine) {
		e.config = cfg
	}
}

func WithExecutor(ex Executor) Option {
	return func(e *Engine) {
		e.executor = ex
	}
}

func WithSkillsDir(dir string) Option {
	return func(e *Engine) {
		e.skillsDir = dir
	}
}

func WithStore(store RunStore) Option {
	return func(e *Engine) {
		e.store = store
	}
}

func WithHarnessResolver(r HarnessResolver) Option {
	return func(e *Engine) {
		e.harnessOverride = r
	}
}

func WithInferenceResolver(r InferenceResolver) Option {
	return func(e *Engine) {
		e.inferenceOverride = r
	}
}

func WithMCPResolver(r MCPResolver) Option {
	return func(e *Engine) {
		e.mcpOverride = r
	}
}
