package compose

type RuntimeProfile struct {
	Name       string            `yaml:"name,omitempty"`
	Kind       string            `yaml:"kind"`            // harness | framework | raw
	Image      string            `yaml:"image"`
	EnvMapping map[string]string `yaml:"env-mapping"`     // N-var template map
	Entrypoint []string          `yaml:"entrypoint"`
	Tools      []string          `yaml:"tools"`
	Providers  []string          `yaml:"providers,omitempty"` // OpenShell provider profiles to attach
	Binaries   []string          `yaml:"binaries,omitempty"` // Full binary paths for egress policy rules
	MCPConfig  MCPConfig         `yaml:"mcp-config,omitempty"`
}

type InferenceSpec struct {
	Name         string            `yaml:"name,omitempty"`
	Endpoint     string            `yaml:"endpoint"`
	Provider     string            `yaml:"provider"`
	DefaultModel string            `yaml:"default-model"`
	Models       map[string]string `yaml:"models,omitempty"` // tier overrides: opus, haiku, etc.
	Egress       []string          `yaml:"egress"`
}

type MCPSpec struct {
	Name     string            `yaml:"name,omitempty"`
	Type     string            `yaml:"type,omitempty"`    // stdio | http
	Command  string            `yaml:"command,omitempty"` // binary for stdio servers
	Args     []string          `yaml:"args,omitempty"`
	URL      string            `yaml:"url,omitempty"`     // endpoint for http servers
	Env      map[string]string `yaml:"env,omitempty"`     // env vars for the MCP server process
	Provider string            `yaml:"provider"`
	Egress   []string          `yaml:"egress"`
}

type ResolvedMCP struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	URL     string            `json:"url,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

type MCPConfig struct {
	Format string `yaml:"format,omitempty"` // claude | codex | goose
	Path   string `yaml:"path,omitempty"`   // target path inside sandbox
}

type Policy struct {
	Name string `yaml:"name,omitempty"`
	Path string `yaml:"path,omitempty"`
}

type SandboxOpts struct {
	Scope string `yaml:"scope" json:"scope,omitempty"` // session | agent | shared
	Mode  string `yaml:"mode" json:"mode,omitempty"`   // all | non-main | off
	TTL   string `yaml:"ttl" json:"ttl,omitempty"`     // e.g. "30m"
}

type Defaults struct {
	Inference string      `yaml:"inference"`
	Policy    string      `yaml:"policy"`
	Sandbox   SandboxOpts `yaml:"sandbox"`
}

type Agent struct {
	Name       string            `yaml:"name"`
	Runtime    string            `yaml:"runtime,omitempty"`
	Image      string            `yaml:"image,omitempty"`
	Prompt     string            `yaml:"prompt,omitempty"`
	Inference  string            `yaml:"inference,omitempty"`
	Model      string            `yaml:"model,omitempty"`
	MCP        []string          `yaml:"mcp,omitempty"`
	Skills     []string          `yaml:"skills,omitempty"`
	Tools      []string          `yaml:"tools,omitempty"`
	Policy     string            `yaml:"policy,omitempty"`
	Sandbox    SandboxOpts       `yaml:"sandbox,omitempty"`
	Env        map[string]string `yaml:"env,omitempty"`
	EnvMapping map[string]string `yaml:"env-mapping,omitempty"`
	Entrypoint []string          `yaml:"entrypoint,omitempty"`
	Workspace  string            `yaml:"workspace,omitempty"`
}

type Mount struct {
	Source string
	Target string
}

type ResolvedSpec struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels"`
	RuntimeKind string            `json:"runtime_kind"` // harness | framework | raw
	Image       string            `json:"image"`
	Entrypoint  []string          `json:"entrypoint"`
	Binaries    []string          `json:"binaries,omitempty"` // Full binary paths for egress policy rules
	Providers   []string          `json:"providers"`
	Env         map[string]string `json:"env"`
	Egress      []string          `json:"egress"`
	Policy      string            `json:"policy"`
	Tools       []string          `json:"tools"`
	Sandbox     SandboxOpts       `json:"sandbox"`
	Prompt      string            `json:"prompt"`
	MCPServers    []ResolvedMCP   `json:"mcp_servers,omitempty"`
	MCPConfigPath string         `json:"mcp_config_path,omitempty"`
	SkillMounts []Mount           `json:"skill_mounts,omitempty"`
	Workspace   string            `json:"workspace"`
}

type AttachOpts struct {
	Shell bool
}

type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	return e.Err
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
