package environment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvironments_NotExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-environments.yaml")
	ef, err := LoadEnvironments(path)
	if err != nil {
		t.Fatalf("LoadEnvironments returned error for missing file: %v", err)
	}
	if ef == nil {
		t.Fatal("LoadEnvironments returned nil EnvironmentFile")
	}
	if len(ef.Environments) != 0 {
		t.Fatalf("expected 0 environments, got %d", len(ef.Environments))
	}
}

func TestLoadEnvironments_ParseError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "environments.yaml")
	if err := os.WriteFile(path, []byte("environments: [\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if _, err := LoadEnvironments(path); err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestGetVariablesAndNames(t *testing.T) {
	ef := &EnvironmentFile{
		Environments: []Environment{
			{
				Name: "Development",
				Variables: map[string]Variable{
					"base_url": {Value: "http://localhost:8080"},
					"token":    {Value: "dev-token", Secret: true},
				},
			},
			{
				Name: "Production",
				Variables: map[string]Variable{
					"base_url": {Value: "https://api.example.com"},
				},
			},
		},
	}

	vars := ef.GetVariables("Development")
	if vars["base_url"] != "http://localhost:8080" {
		t.Fatalf("base_url mismatch: %q", vars["base_url"])
	}
	if vars["token"] != "dev-token" {
		t.Fatalf("token mismatch: %q", vars["token"])
	}

	missing := ef.GetVariables("NonExistent")
	if len(missing) != 0 {
		t.Fatalf("expected empty variables for unknown env, got %v", missing)
	}

	names := ef.Names()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "Development" || names[1] != "Production" {
		t.Fatalf("unexpected names order/content: %v", names)
	}
}

func TestLoadEnvironments_ValidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "environments.yaml")
	content := `environments:
  - name: Development
    variables:
      base_url:
        value: "http://localhost:8080"
      token:
        value: "dev-token"
        secret: true
  - name: Production
    variables:
      base_url:
        value: "https://api.example.com"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	ef, err := LoadEnvironments(path)
	if err != nil {
		t.Fatalf("LoadEnvironments failed: %v", err)
	}
	if len(ef.Environments) != 2 {
		t.Fatalf("expected 2 environments, got %d", len(ef.Environments))
	}
	if !ef.Environments[0].Variables["token"].Secret {
		t.Fatal("expected token variable to be marked secret")
	}
}
