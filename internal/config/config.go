package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
	API     APIConfig     `yaml:"api"`
	UI      UIConfig      `yaml:"ui"`
	Beads   BeadsConfig   `yaml:"beads"`
	Logging LoggingConfig `yaml:"logging"`
	Loop    LoopConfig    `yaml:"loop"`
}

type CodexConfig struct {
	CLIPath        string            `yaml:"cli_path"`
	Model          string            `yaml:"model"`
	TimeoutSeconds int               `yaml:"timeout_seconds"`
	ExtraArgs      []string          `yaml:"extra_args"`
	ApprovalPolicy string            `yaml:"approval_policy"`
	SandboxMode    string            `yaml:"sandbox_mode"`
	Env            map[string]string `yaml:"env"`
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

type LoopConfig struct {
	MaxIterations int      `yaml:"max_iterations"`
	Phases        []string `yaml:"phases"`
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
	if c.Codex.TimeoutSeconds == 0 {
		c.Codex.TimeoutSeconds = 900
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
	if c.Loop.MaxIterations == 0 {
		c.Loop.MaxIterations = 50
	}
	if c.Loop.Phases == nil || len(c.Loop.Phases) == 0 {
		c.Loop.Phases = []string{"ideate", "plan", "implement", "review", "test"}
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
