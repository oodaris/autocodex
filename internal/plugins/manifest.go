package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const ProtocolVersion = 1

var supportedTransports = map[string]bool{
	"jsonrpc": true,
	"grpc":    true,
}

type Manifest struct {
	Name            string       `json:"name" yaml:"name"`
	Version         string       `json:"version" yaml:"version"`
	ProtocolVersion int          `json:"protocol_version" yaml:"protocol_version"`
	Entrypoint      string       `json:"entrypoint" yaml:"entrypoint"`
	Transport       string       `json:"transport" yaml:"transport"`
	Capabilities    []Capability `json:"capabilities" yaml:"capabilities"`
}

type Capability struct {
	Name         string `json:"name" yaml:"name"`
	InputSchema  string `json:"input_schema" yaml:"input_schema"`
	OutputSchema string `json:"output_schema" yaml:"output_schema"`
}

type Plugin struct {
	Manifest     Manifest
	ManifestPath string
	Dir          string
}

func Discover(paths []string) ([]Plugin, error) {
	var plugins []Plugin
	for _, root := range paths {
		if root == "" {
			continue
		}
		info, err := os.Stat(root)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			if isManifest(root) {
				plugin, err := LoadManifest(root)
				if err != nil {
					return nil, err
				}
				plugins = append(plugins, plugin)
			}
			continue
		}

		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, fmt.Errorf("read plugins dir: %w", err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			dir := filepath.Join(root, entry.Name())
			manifestPath := findManifest(dir)
			if manifestPath == "" {
				continue
			}
			plugin, err := LoadManifest(manifestPath)
			if err != nil {
				return nil, err
			}
			plugins = append(plugins, plugin)
		}
	}

	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Manifest.Name < plugins[j].Manifest.Name
	})

	return plugins, nil
}

func LoadManifest(path string) (Plugin, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Plugin{}, fmt.Errorf("read manifest: %w", err)
	}

	manifest := Manifest{}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &manifest); err != nil {
			return Plugin{}, fmt.Errorf("parse manifest json: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			return Plugin{}, fmt.Errorf("parse manifest yaml: %w", err)
		}
	default:
		return Plugin{}, fmt.Errorf("unsupported manifest extension: %s", ext)
	}

	if err := validateManifest(manifest); err != nil {
		return Plugin{}, fmt.Errorf("manifest %s: %w", path, err)
	}

	return Plugin{
		Manifest:     manifest,
		ManifestPath: path,
		Dir:          filepath.Dir(path),
	}, nil
}

func validateManifest(m Manifest) error {
	if m.Name == "" {
		return errors.New("name is required")
	}
	if m.ProtocolVersion == 0 {
		return errors.New("protocol_version is required")
	}
	if m.ProtocolVersion != ProtocolVersion {
		return fmt.Errorf("unsupported protocol_version %d", m.ProtocolVersion)
	}
	if m.Entrypoint == "" {
		return errors.New("entrypoint is required")
	}
	if m.Transport == "" {
		return errors.New("transport is required")
	}
	if !supportedTransports[m.Transport] {
		return fmt.Errorf("unsupported transport %s", m.Transport)
	}
	return nil
}

func findManifest(dir string) string {
	for _, name := range []string{"plugin.yaml", "plugin.yml", "plugin.json"} {
		path := filepath.Join(dir, name)
		if exists(path) {
			return path
		}
	}
	return ""
}

func isManifest(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	return name == "plugin.yaml" || name == "plugin.yml" || name == "plugin.json"
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
