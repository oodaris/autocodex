package config

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
	if c.Cleanup.RetentionDays == 0 {
		c.Cleanup.RetentionDays = 14
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
	if c.Loop.StopConditions.MaxHeartbeatSeconds == 0 {
		c.Loop.StopConditions.MaxHeartbeatSeconds = 180
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
	if len(c.Loop.Phases) == 0 {
		c.Loop.Phases = []string{"ideate", "plan", "implement", "review", "test"}
	}
	if c.Loop.PhaseIdleSecs == nil {
		c.Loop.PhaseIdleSecs = map[string]int{}
	}
	if c.Loop.Feedback.Mode == "" {
		if c.Autonomy.Enabled {
			c.Loop.Feedback.Mode = "on"
		} else {
			c.Loop.Feedback.Mode = "off"
		}
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
	if c.Autonomy.RequireActions == nil {
		enabled := c.Autonomy.Enabled
		c.Autonomy.RequireActions = &enabled
	}
	if c.Autonomy.RequireNext == nil {
		enabled := c.Autonomy.Enabled
		c.Autonomy.RequireNext = &enabled
	}
	if c.Autonomy.RequireBD == nil {
		enabled := c.Autonomy.Enabled
		c.Autonomy.RequireBD = &enabled
	}
	if c.Autonomy.SpecTemplate == "" {
		c.Autonomy.SpecTemplate = "docs/specs/TEMPLATE.md"
	}
	if c.Autonomy.PlanTemplate == "" {
		c.Autonomy.PlanTemplate = "docs/plans/TEMPLATE.md"
	}
	if c.Autonomy.TasksSchema == "" {
		c.Autonomy.TasksSchema = "docs/contracts/autonomy-tasks.schema.json"
	}
	if c.Autonomy.ActionsSchema == "" {
		c.Autonomy.ActionsSchema = "docs/contracts/autonomy-actions.schema.json"
	}
	if c.Autonomy.TasksOutputTemplate == "" {
		c.Autonomy.TasksOutputTemplate = "docs/plans/%s-tasks.json"
	}
	if c.Autonomy.StopConditions.MaxFixAttempts == 0 {
		c.Autonomy.StopConditions.MaxFixAttempts = 3
	}
	if c.Autonomy.StopConditions.StopOnGateFailure == nil {
		enabled := true
		c.Autonomy.StopConditions.StopOnGateFailure = &enabled
	}
}
