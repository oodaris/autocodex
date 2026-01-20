package config

import "path/filepath"

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
