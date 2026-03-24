package config

import (
	"path/filepath"
	"testing"
)

func TestResolvePrecedenceCLIEnvProjectGlobalDefault(t *testing.T) {
	tmp := t.TempDir()
	globalPath := filepath.Join(tmp, "global", "config.json")
	projectPath := filepath.Join(tmp, "project", ".ccc", "config.json")

	globalCfg := DefaultConfig()
	globalCfg.OpenAI.Model = "gpt-global"
	globalCfg.OpenAI.APIKeyValue = "global-key"
	globalCfg.Modes.Safe = false
	if err := Save(globalPath, globalCfg); err != nil {
		t.Fatalf("save global config: %v", err)
	}

	projectCfg := DefaultConfig()
	projectCfg.OpenAI.Model = "gpt-project"
	projectCfg.Modes.Safe = true
	if err := Save(projectPath, projectCfg); err != nil {
		t.Fatalf("save project config: %v", err)
	}

	t.Setenv("CCC_OPENAI_MODEL", "gpt-env")
	t.Setenv("CCC_SAFE", "false")

	eff, err := Resolve(CLIOverrides{
		ProjectConfigPath: projectPath,
		GlobalConfigPath:  globalPath,
		SafeSet:           true,
		Safe:              true,
	})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if got, want := eff.Config.OpenAI.Model, "gpt-env"; got != want {
		t.Fatalf("openai model mismatch: got %q want %q", got, want)
	}
	if got, want := eff.Config.Modes.Safe, true; got != want {
		t.Fatalf("safe mismatch: got %v want %v", got, want)
	}
	if got, want := eff.Config.OpenAI.APIKeyValue, "global-key"; got != want {
		t.Fatalf("api key value mismatch: got %q want %q", got, want)
	}

	assertChain(t, eff.SourceChains["openai.model"], []string{
		SourceDefault, SourceGlobalConfig, SourceProjectConfig, SourceEnv,
	})
	assertChain(t, eff.SourceChains["modes.safe"], []string{
		SourceDefault, SourceGlobalConfig, SourceProjectConfig, SourceEnv, SourceCLI,
	})
	assertChain(t, eff.SourceChains["openai.api_key_value"], []string{
		SourceDefault, SourceGlobalConfig,
	})
}

func TestResolveConfigPathBackCompatAlias(t *testing.T) {
	tmp := t.TempDir()
	projectPath := filepath.Join(tmp, ".ccc", "config.json")

	cfg := DefaultConfig()
	cfg.OpenAI.Model = "gpt-project-alias"
	if err := Save(projectPath, cfg); err != nil {
		t.Fatalf("save project config: %v", err)
	}

	eff, err := Resolve(CLIOverrides{
		ConfigPath: projectPath,
	})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if got, want := eff.Config.OpenAI.Model, "gpt-project-alias"; got != want {
		t.Fatalf("openai model mismatch: got %q want %q", got, want)
	}
	if got, want := eff.ProjectConfigPath, projectPath; got != want {
		t.Fatalf("project config path mismatch: got %q want %q", got, want)
	}
}

func assertChain(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("chain length mismatch: got %v want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("chain mismatch at %d: got %v want %v", i, got, want)
		}
	}
}
