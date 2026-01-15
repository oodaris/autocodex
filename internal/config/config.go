package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const DefaultConfigFile = "autocodex.yaml"

func ResolveConfigPath() string {
	if v := os.Getenv("AUTOCODEX_CONFIG"); v != "" {
		return v
	}
	return DefaultConfigFile
}

type Config struct {
	Version string        `yaml:"version"`
	Mode    string        `yaml:"mode"`
	Codex   CodexConfig   `yaml:"codex"`
	Paths   PathsConfig   `yaml:"paths"`
	Skills  SkillsConfig  `yaml:"skills"`
	Plugins PluginsConfig `yaml:"plugins"`
	Hub     HubConfig     `yaml:"hub"`
	API     APIConfig     `yaml:"api"`
	UI      UIConfig      `yaml:"ui"`
	Beads   BeadsConfig   `yaml:"beads"`
	Logging LoggingConfig `yaml:"logging"`
	Auth    AuthConfig    `yaml:"auth"`
	Loop    LoopConfig    `yaml:"loop"`
}

type CodexConfig struct {
	CLIPath         string            `yaml:"cli_path"`
	Model           string            `yaml:"model"`
	ReasoningEffort string            `yaml:"reasoning_effort"`
	TimeoutSeconds  int               `yaml:"timeout_seconds"`
	ExtraArgs       []string          `yaml:"extra_args"`
	ApprovalPolicy  string            `yaml:"approval_policy"`
	SandboxMode     string            `yaml:"sandbox_mode"`
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
	Mode           string               `yaml:"mode"`
	MaxIterations  int                  `yaml:"max_iterations"`
	Phases         []string             `yaml:"phases"`
	StopConditions StopConditionsConfig `yaml:"stop_conditions"`
	Feedback       FeedbackConfig       `yaml:"feedback"`
}

type StopConditionsConfig struct {
	MaxDurationSeconds     int `yaml:"max_duration_seconds"`
	MaxIdleSeconds         int `yaml:"max_idle_seconds"`
	MaxConsecutiveFailures int `yaml:"max_consecutive_failures"`
}

type FeedbackConfig struct {
	Mode         string   `yaml:"mode"`
	Sources      []string `yaml:"sources"`
	MaxArtifacts int      `yaml:"max_artifacts"`
	MaxEvents    int      `yaml:"max_events"`
	MaxBytes     int      `yaml:"max_bytes"`
	MemoryGlob   string   `yaml:"memory_glob"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c *Config) ApplyDefaults() {
	if c.Version == "" {
		c.Version = "v1"
	}
	if c.Mode == "" {
		c.Mode = "yolo"
	}
	if c.Codex.CLIPath == "" {
		c.Codex.CLIPath = "codex"
	}
	if c.Codex.Model == "" {
		c.Codex.Model = "gpt-5.2-codex"
	}
	if c.Codex.TimeoutSeconds == 0 {
		c.Codex.TimeoutSeconds = 900
	}
	if c.Codex.ReasoningEffort == "" {
		c.Codex.ReasoningEffort = "xhigh"
	}
	if c.Codex.ExtraArgs == nil {
		c.Codex.ExtraArgs = []string{}
	}
	if c.Codex.ApprovalPolicy == "" {
		c.Codex.ApprovalPolicy = "on-request"
	}
	if c.Codex.SandboxMode == "" {
		c.Codex.SandboxMode = "workspace-write"
	}
	if c.Codex.Env == nil {
		c.Codex.Env = map[string]string{}
	}
	if c.Paths.StateDir == "" {
		c.Paths.StateDir = ".autocodex"
	}
	if c.Paths.MemoryDir == "" {
		c.Paths.MemoryDir = "memory"
	}
	if c.Paths.LogsDir == "" {
		c.Paths.LogsDir = "logs"
	}
	if c.Paths.ArtifactsDir == "" {
		c.Paths.ArtifactsDir = "artifacts"
	}
	if c.Paths.RunsDir == "" {
		c.Paths.RunsDir = "runs"
	}
	if c.Skills.Paths == nil {
		c.Skills.Paths = []string{}
	}
	if c.Skills.Allowlist == nil {
		c.Skills.Allowlist = []string{}
	}
	if c.Skills.Denylist == nil {
		c.Skills.Denylist = []string{}
	}
	if c.Plugins.Paths == nil {
		c.Plugins.Paths = []string{"plugins"}
	}
	if c.Plugins.TimeoutSeconds == 0 {
		c.Plugins.TimeoutSeconds = 60
	}
	if c.Hub.Workspaces == nil {
		c.Hub.Workspaces = []WorkspaceConfig{}
	}
	if c.API.Host == "" {
		c.API.Host = "127.0.0.1"
	}
	if c.API.Port == 0 {
		c.API.Port = 7788
	}
	if c.API.BasePath == "" {
		c.API.BasePath = "/"
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
	if c.Auth.Tokens == nil {
		c.Auth.Tokens = []string{}
	}
	if c.Loop.MaxIterations == 0 {
		c.Loop.MaxIterations = 50
	}
	if c.Loop.Mode == "" {
		c.Loop.Mode = "bounded"
	}
	if c.Loop.Phases == nil || len(c.Loop.Phases) == 0 {
		c.Loop.Phases = []string{"ideate", "plan", "implement", "review", "test"}
	}
	if c.Loop.Feedback.Mode == "" {
		c.Loop.Feedback.Mode = "off"
	}
	if c.Loop.Feedback.Sources == nil {
		c.Loop.Feedback.Sources = []string{"memory", "events", "artifacts"}
	}
	if c.Loop.Feedback.MaxArtifacts == 0 {
		c.Loop.Feedback.MaxArtifacts = 20
	}
	if c.Loop.Feedback.MaxEvents == 0 {
		c.Loop.Feedback.MaxEvents = 200
	}
	if c.Loop.Feedback.MemoryGlob == "" {
		c.Loop.Feedback.MemoryGlob = "*.md"
	}
}

func (c Config) Validate() error {
	if c.Mode != "yolo" && c.Mode != "safe" {
		return fmt.Errorf("invalid mode: %s", c.Mode)
	}
	if c.Version != "v1" {
		return fmt.Errorf("unsupported config version: %s", c.Version)
	}
	if c.Codex.ApprovalPolicy != "" {
		if !oneOf(c.Codex.ApprovalPolicy, []string{"untrusted", "on-failure", "on-request", "never"}) {
			return fmt.Errorf("invalid codex.approval_policy: %s", c.Codex.ApprovalPolicy)
		}
	}
	if c.Codex.SandboxMode != "" {
		if !oneOf(c.Codex.SandboxMode, []string{"read-only", "workspace-write", "danger-full-access"}) {
			return fmt.Errorf("invalid codex.sandbox_mode: %s", c.Codex.SandboxMode)
		}
	}
	if c.Codex.ReasoningEffort != "" {
		if !oneOf(c.Codex.ReasoningEffort, []string{"minimal", "low", "medium", "high", "xhigh"}) {
			return fmt.Errorf("invalid codex.reasoning_effort: %s", c.Codex.ReasoningEffort)
		}
	}
	if c.Hub.Enabled {
		seen := map[string]bool{}
		for _, ws := range c.Hub.Workspaces {
			if ws.ID == "" {
				return errors.New("hub.workspaces.id is required")
			}
			if strings.TrimSpace(ws.Root) == "" {
				return fmt.Errorf("hub workspace %s root is required", ws.ID)
			}
			if seen[ws.ID] {
				return fmt.Errorf("duplicate hub workspace id: %s", ws.ID)
			}
			seen[ws.ID] = true
		}
	}
	if c.Auth.Enabled && len(c.Auth.Tokens) == 0 && strings.TrimSpace(c.Auth.TokenEnv) == "" {
		return errors.New("auth.tokens or auth.token_env is required when auth is enabled")
	}
	if c.Loop.Mode != "" {
		if !oneOf(c.Loop.Mode, []string{"bounded", "continuous"}) {
			return fmt.Errorf("invalid loop.mode: %s", c.Loop.Mode)
		}
	}
	if c.Loop.StopConditions.MaxDurationSeconds < 0 {
		return fmt.Errorf("invalid loop.stop_conditions.max_duration_seconds: %d", c.Loop.StopConditions.MaxDurationSeconds)
	}
	if c.Loop.StopConditions.MaxIdleSeconds < 0 {
		return fmt.Errorf("invalid loop.stop_conditions.max_idle_seconds: %d", c.Loop.StopConditions.MaxIdleSeconds)
	}
	if c.Loop.StopConditions.MaxConsecutiveFailures < 0 {
		return fmt.Errorf("invalid loop.stop_conditions.max_consecutive_failures: %d", c.Loop.StopConditions.MaxConsecutiveFailures)
	}
	if c.Loop.Feedback.Mode != "" {
		if !oneOf(c.Loop.Feedback.Mode, []string{"off", "on"}) {
			return fmt.Errorf("invalid loop.feedback.mode: %s", c.Loop.Feedback.Mode)
		}
	}
	for _, source := range c.Loop.Feedback.Sources {
		if !oneOf(source, []string{"memory", "events", "artifacts"}) {
			return fmt.Errorf("invalid loop.feedback.sources entry: %s", source)
		}
	}
	if c.Loop.Feedback.MaxArtifacts < 0 {
		return fmt.Errorf("invalid loop.feedback.max_artifacts: %d", c.Loop.Feedback.MaxArtifacts)
	}
	if c.Loop.Feedback.MaxEvents < 0 {
		return fmt.Errorf("invalid loop.feedback.max_events: %d", c.Loop.Feedback.MaxEvents)
	}
	if c.Loop.Feedback.MaxBytes < 0 {
		return fmt.Errorf("invalid loop.feedback.max_bytes: %d", c.Loop.Feedback.MaxBytes)
	}
	if c.API.Port < 1 || c.API.Port > 65535 {
		return fmt.Errorf("invalid api.port: %d", c.API.Port)
	}
	if len(c.Loop.Phases) == 0 {
		return errors.New("loop.phases must not be empty")
	}
	return nil
}

func oneOf(value string, allowed []string) bool {
	for _, v := range allowed {
		if value == v {
			return true
		}
	}
	return false
}

func (c Config) StateDir() string {
	return filepath.Clean(c.Paths.StateDir)
}

func (c Config) MemoryDir() string {
	return filepath.Join(c.StateDir(), c.Paths.MemoryDir)
}

func (c Config) LogsDir() string {
	return filepath.Join(c.StateDir(), c.Paths.LogsDir)
}

func (c Config) RunsDir() string {
	return filepath.Join(c.StateDir(), c.Paths.RunsDir)
}

func (c Config) ArtifactsDir() string {
	return filepath.Join(c.StateDir(), c.Paths.ArtifactsDir)
}
