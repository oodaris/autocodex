package main

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/oodaris/autocodex/internal/config"
)

//go:embed all:bootstrap_harness_assets
var bootstrapHarnessAssetsFS embed.FS

func bootstrapHarnessPolicyAssets(cfg config.Config, force bool) error {
	rolePackPath := strings.TrimSpace(cfg.Autonomy.Harness.RolePackPath)
	if rolePackPath == "" {
		rolePackPath = ".codex"
	}

	if err := writeEmbeddedAssetTree("bootstrap_harness_assets/role-pack", rolePackPath, force); err != nil {
		return err
	}
	if err := writeEmbeddedAssetTree("bootstrap_harness_assets/docs", "docs", force); err != nil {
		return err
	}
	if err := writeEmbeddedAssetTree("bootstrap_harness_assets/scripts", "scripts", force); err != nil {
		return err
	}
	return nil
}

func writeEmbeddedAssetTree(sourceRoot string, targetRoot string, force bool) error {
	return fs.WalkDir(bootstrapHarnessAssetsFS, sourceRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		relPath := strings.TrimPrefix(path, sourceRoot+"/")
		if relPath == path || strings.TrimSpace(relPath) == "" {
			return fmt.Errorf("resolve embedded path %s under %s", path, sourceRoot)
		}
		targetPath := filepath.Join(targetRoot, filepath.FromSlash(relPath))

		content, err := bootstrapHarnessAssetsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded asset %s: %w", path, err)
		}

		mode := fs.FileMode(0o644)
		if strings.HasSuffix(targetPath, ".sh") {
			mode = 0o755
		}
		if err := writeFileIfMissingWithMode(targetPath, content, force, mode); err != nil {
			return err
		}
		return nil
	})
}
