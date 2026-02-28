package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureBaseFileCreatesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cleanup.rules.json")
	if err := EnsureBaseFile(path); err != nil {
		t.Fatalf("ensure base file: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected rules file to exist: %v", err)
	}
}

func TestLoadMergesLocalOverrides(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "cleanup.rules.json")
	local := filepath.Join(dir, "cleanup.local.json")
	if err := EnsureBaseFile(base); err != nil {
		t.Fatalf("ensure base file: %v", err)
	}
	localJSON := `{
  "schema_version": 1,
  "rules": [
    {
      "id": "split_functions",
      "enabled": true,
      "title": "Split up functions into reusable pieces",
      "description": "Override description",
      "details": "Override details",
      "risk_level": "aggressive"
    },
    {
      "id": "custom_rule",
      "enabled": true,
      "title": "Custom Rule",
      "description": "Custom description",
      "details": "Custom details",
      "risk_level": "safe"
    }
  ]
}`
	if err := os.WriteFile(local, []byte(localJSON), 0o644); err != nil {
		t.Fatalf("write local: %v", err)
	}

	loaded, warnings, err := Load(base, local)
	if err != nil {
		t.Fatalf("load rules: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got: %v", warnings)
	}
	foundCustom := false
	foundSplit := false
	for _, r := range loaded {
		if r.ID == "custom_rule" {
			foundCustom = true
		}
		if r.ID == "split_functions" && r.Description == "Override description" && r.Enabled {
			foundSplit = true
		}
	}
	if !foundCustom || !foundSplit {
		t.Fatalf("expected merged rules to include custom and override")
	}
}
