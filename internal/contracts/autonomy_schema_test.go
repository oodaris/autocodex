package contracts_test

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

func TestAutonomyTasksSchema(t *testing.T) {
	validateSchemaExample(t,
		"docs/contracts/autonomy-tasks.schema.json",
		"docs/contracts/autonomy-tasks.example.json",
	)
}

func TestAutonomyActionsSchema(t *testing.T) {
	validateSchemaExample(t,
		"docs/contracts/autonomy-actions.schema.json",
		"docs/contracts/autonomy-actions.example.json",
	)
}

func validateSchemaExample(t *testing.T, schemaPath, examplePath string) {
	root := repoRoot(t)
	absSchema := mustAbs(t, filepath.Join(root, schemaPath))
	tasksSchema := filepath.Join(root, "docs/contracts/autonomy-tasks.schema.json")
	actionsSchema := filepath.Join(root, "docs/contracts/autonomy-actions.schema.json")
	schema := mustCompileSchema(t, absSchema, tasksSchema, actionsSchema)

	data, err := os.ReadFile(mustAbs(t, filepath.Join(root, examplePath)))
	if err != nil {
		t.Fatalf("read example: %v", err)
	}

	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("parse example: %v", err)
	}

	if err := schema.Validate(payload); err != nil {
		t.Fatalf("schema validation failed: %v", err)
	}
}

func mustCompileSchema(t *testing.T, absPath, tasksSchema, actionsSchema string) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	compiler.LoadURL = func(url string) (io.ReadCloser, error) {
		switch url {
		case "https://autocodex.dev/contracts/autonomy-tasks.schema.json":
			return os.Open(tasksSchema)
		case "https://autocodex.dev/contracts/autonomy-actions.schema.json":
			return os.Open(actionsSchema)
		default:
			return jsonschema.LoadURL(url)
		}
	}

	schema, err := compiler.Compile(fileURL(absPath))
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return schema
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	return abs
}

func fileURL(path string) string {
	return "file://" + filepath.ToSlash(path)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("go.mod not found from %s", dir)
	return ""
}
