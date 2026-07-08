package compose

type HarnessProfile struct {
	Name       string            `yaml:"name,omitempty"`
	Image      string            `yaml:"image"`
	EnvMapping EnvMapping        `yaml:"env-mapping"`
	Entrypoint []string          `yaml:"entrypoint"`
	Tools      []string          `yaml:"tools"`
}

type EnvMapping struct {
	Endpoint string `yaml:"endpoint"`
	Key      string `yaml:"key"`
	Model    string `yaml:"model"`
}

type InferenceSpec struct {
	Name         string   `yaml:"name,omitempty"`
	Endpoint     string   `yaml:"endpoint"`
	Provider     string   `yaml:"provider"`
	DefaultModel string   `yaml:"default-model"`
	Egress       []string `yaml:"egress"`
}

type MCPSpec struct {
	Name     string   `yaml:"name,omitempty"`
	Provider string   `yaml:"provider"`
	Egress   []string `yaml:"egress"`
}

type Policy struct {
	Name string `yaml:"name,omitempty"`
	Path string `yaml:"path,omitempty"`
}

type SandboxConfig struct {
	Scope string `yaml:"scope"` // session | agent | shared
	Mode  string `yaml:"mode"`  // all | non-main | off
}

type Defaults struct {
	Inference string        `yaml:"inference"`
	Policy    string        `yaml:"policy"`
	Sandbox   SandboxConfig `yaml:"sandbox"`
}

type Agent struct {
	Name       string            `yaml:"name"`
	Harness    string            `yaml:"harness,omitempty"`
	Image      string            `yaml:"image,omitempty"`
	Prompt     string            `yaml:"prompt,omitempty"`
	PromptFile string            `yaml:"prompt-file,omitempty"`
	Inference  string            `yaml:"inference,omitempty"`
	Model      string            `yaml:"model,omitempty"`
	MCP        []string          `yaml:"mcp,omitempty"`
	Skills     []string          `yaml:"skills,omitempty"`
	Tools      []string          `yaml:"tools,omitempty"`
	Policy     string            `yaml:"policy,omitempty"`
	Sandbox    SandboxConfig     `yaml:"sandbox,omitempty"`
	Env        map[string]string `yaml:"env,omitempty"`
	EnvMapping *EnvMapping       `yaml:"env-mapping,omitempty"`
	Entrypoint []string          `yaml:"entrypoint,omitempty"`
	Workspace  string            `yaml:"workspace,omitempty"`
}

type Mount struct {
	Source string
	Target string
}

type ResolvedSpec struct {
	Name        string
	Labels      map[string]string
	Image       string
	Entrypoint  []string
	Providers   []string
	Env         map[string]string
	Egress      []string
	Policy      string
	Tools       []string
	Prompt      string
	SkillMounts []Mount
	Workspace   string
}

type SandboxState string

const (
	SandboxRunning SandboxState = "running"
	SandboxStopped SandboxState = "stopped"
	SandboxUnknown SandboxState = "unknown"
)

type Run struct {
	ID        string
	Agent     string
	Sandbox   string
	StartedAt int64
	StoppedAt int64
	Status    SandboxState
}

type AgentStatus struct {
	Name    string
	Sandbox string
	Status  SandboxState
	Since   int64
}

type Skill struct {
	Name          string   `yaml:"name"`
	Prompt        string   `yaml:"-"`
	RequiresMCP   []string `yaml:"mcp,omitempty"`
	RequiresTools []string `yaml:"tools,omitempty"`
	References    []string `yaml:"-"` // file paths relative to skill dir
}

type SkillFrontmatter struct {
	Requires struct {
		MCP   []string `yaml:"mcp"`
		Tools []string `yaml:"tools"`
	} `yaml:"requires"`
}
