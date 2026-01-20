package config

import "os"

const DefaultConfigFile = "autocodex.yaml"

func ResolveConfigPath() string {
	if v := os.Getenv("AUTOCODEX_CONFIG"); v != "" {
		return v
	}
	return DefaultConfigFile
}

type Config struct {
	Version  string         `yaml:"version"`
	Mode     string         `yaml:"mode"`
	Codex    CodexConfig    `yaml:"codex"`
	Paths    PathsConfig    `yaml:"paths"`
	Skills   SkillsConfig   `yaml:"skills"`
	Plugins  PluginsConfig  `yaml:"plugins"`
	Hub      HubConfig      `yaml:"hub"`
	API      APIConfig      `yaml:"api"`
	UI       UIConfig       `yaml:"ui"`
	Beads    BeadsConfig    `yaml:"beads"`
	Cleanup  CleanupConfig  `yaml:"cleanup"`
	Logging  LoggingConfig  `yaml:"logging"`
	Auth     AuthConfig     `yaml:"auth"`
	Loop     LoopConfig     `yaml:"loop"`
	Autonomy AutonomyConfig `yaml:"autonomy"`
}

type CodexConfig struct {
	CLIPath         string            `yaml:"cli_path"`
	Model           string            `yaml:"model"`
	ReasoningEffort string            `yaml:"reasoning_effort"`
	TimeoutSeconds  int               `yaml:"timeout_seconds"`
	ExtraArgs       []string          `yaml:"extra_args"`
	ApprovalPolicy  string            `yaml:"approval_policy"`
	SandboxMode     string            `yaml:"sandbox_mode"`
	JSONOutput      bool              `yaml:"json_output"`
	OutputLast      bool              `yaml:"output_last_message"`
	PromptStdin     bool              `yaml:"prompt_stdin"`
	Env             map[string]string `yaml:"env"`
}

type PathsConfig struct {
	StateDir     string `yaml:"state_dir"`
	MemoryDir    string `yaml:"memory_dir"`
	LogsDir      string `yaml:"logs_dir"`
	ArtifactsDir string `yaml:"artifacts_dir"`
	RunsDir      string `yaml:"runs_dir"`
}

type SkillsConfig struct {
	Paths     []string `yaml:"paths"`
	Allowlist []string `yaml:"allowlist"`
	Denylist  []string `yaml:"denylist"`
}

type PluginsConfig struct {
	Enabled        bool     `yaml:"enabled"`
	Paths          []string `yaml:"paths"`
	TimeoutSeconds int      `yaml:"timeout_seconds"`
}

type HubConfig struct {
	Enabled    bool              `yaml:"enabled"`
	Workspaces []WorkspaceConfig `yaml:"workspaces"`
}

type WorkspaceConfig struct {
	ID         string `yaml:"id"`
	Name       string `yaml:"name"`
	Root       string `yaml:"root"`
	ConfigPath string `yaml:"config_path"`
}

type APIConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	BasePath string `yaml:"base_path"`
}

type UIConfig struct {
	Enabled bool   `yaml:"enabled"`
	Origin  string `yaml:"origin"`
}

type BeadsConfig struct {
	Enabled    bool `yaml:"enabled"`
	AutoCreate bool `yaml:"auto_create"`
	AutoUpdate bool `yaml:"auto_update"`
}

type CleanupConfig struct {
	RetentionDays int `yaml:"retention_days"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type AuthConfig struct {
	Enabled  bool     `yaml:"enabled"`
	TokenEnv string   `yaml:"token_env"`
	Tokens   []string `yaml:"tokens"`
}

type LoopConfig struct {
	Mode             string                 `yaml:"mode"`
	MaxIterations    int                    `yaml:"max_iterations"`
	Phases           []string               `yaml:"phases"`
	StopConditions   StopConditionsConfig   `yaml:"stop_conditions"`
	PhaseIdleSecs    map[string]int         `yaml:"phase_idle_seconds"`
	PromptGuardrails PromptGuardrailsConfig `yaml:"prompt_guardrails"`
	Feedback         FeedbackConfig         `yaml:"feedback"`
}

type AutonomyConfig struct {
	Enabled             bool                   `yaml:"enabled"`
	RequireActions      *bool                  `yaml:"require_actions"`
	RequireNext         *bool                  `yaml:"require_next"`
	RequireBD           *bool                  `yaml:"require_bd"`
	FailOnSchemaError   *bool                  `yaml:"fail_on_schema_error"`
	AllowFallbackTasks  *bool                  `yaml:"allow_fallback_tasks"`
	SpecTemplate        string                 `yaml:"spec_template"`
	PlanTemplate        string                 `yaml:"plan_template"`
	TasksSchema         string                 `yaml:"tasks_schema"`
	ActionsSchema       string                 `yaml:"actions_schema"`
	TasksOutputTemplate string                 `yaml:"tasks_output_template"`
	StopConditions      AutonomyStopConditions `yaml:"stop_conditions"`
}

type AutonomyStopConditions struct {
	MaxFixAttempts    int   `yaml:"max_fix_attempts"`
	MaxBeads          int   `yaml:"max_beads"`
	StopOnGateFailure *bool `yaml:"stop_on_gate_failure"`
}

type StopConditionsConfig struct {
	MaxDurationSeconds     int `yaml:"max_duration_seconds"`
	MaxIdleSeconds         int `yaml:"max_idle_seconds"`
	MaxConsecutiveFailures int `yaml:"max_consecutive_failures"`
	MaxHeartbeatSeconds    int `yaml:"max_heartbeat_seconds"`
}

type PromptGuardrailsConfig struct {
	ReviewMaxBytes int `yaml:"review_max_bytes"`
}

type FeedbackConfig struct {
	Mode         string   `yaml:"mode"`
	Sources      []string `yaml:"sources"`
	MaxArtifacts int      `yaml:"max_artifacts"`
	MaxEvents    int      `yaml:"max_events"`
	MaxBytes     int      `yaml:"max_bytes"`
	MemoryGlob   string   `yaml:"memory_glob"`
	SnapshotPath string   `yaml:"snapshot_path"`
}
