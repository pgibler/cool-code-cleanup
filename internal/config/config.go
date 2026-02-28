package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const (
	SourceDefault = "default"
	SourceConfig  = "config"
	SourceEnv     = "env"
	SourceCLI     = "cli"
)

type OpenAIConfig struct {
	APIKeyEnv   string `json:"api_key_env"`
	APIKeyValue string `json:"api_key_value"`
	Model       string `json:"model"`
}

type ModesConfig struct {
	Safe       bool `json:"safe"`
	Aggressive bool `json:"aggressive"`
	DryRun     bool `json:"dry_run"`
}

type ProfileConfig struct {
	IncludeRoutes            []string `json:"include_routes"`
	IgnoreRoutes             []string `json:"ignore_routes"`
	DependencyShortCircuit   bool     `json:"dependency_short_circuit"`
	ShortCircuitEnvVar       string   `json:"short_circuit_env_var"`
	UpdateEnvFile            bool     `json:"update_env_file"`
	SaveShortCircuitToConfig bool     `json:"save_short_circuit_to_config"`
	EditPermissionMode       string   `json:"edit_permission_mode"`
	AutoApply                bool     `json:"auto_apply"`
}

type CleanupConfig struct {
	RemoveRedundantGuards bool   `json:"remove_redundant_guards"`
	DryRefactor           bool   `json:"dry_refactor"`
	HardenErrorHandling   bool   `json:"harden_error_handling"`
	GateFeaturesEnv       bool   `json:"gate_features_env"`
	SplitFunctions        bool   `json:"split_functions"`
	StandardizeNaming     bool   `json:"standardize_naming"`
	SimplifyComplexLogic  bool   `json:"simplify_complex_logic"`
	DetectExpensive       bool   `json:"detect_expensive_functions"`
	EditPermissionMode    string `json:"edit_permission_mode"`
	AutoApply             bool   `json:"auto_apply"`
}

type GitConfig struct {
	AutoOfferBranchAndCommit bool `json:"auto_offer_branch_and_commit"`
}

type Config struct {
	OpenAI  OpenAIConfig  `json:"openai"`
	Modes   ModesConfig   `json:"modes"`
	Profile ProfileConfig `json:"profile"`
	Cleanup CleanupConfig `json:"cleanup"`
	Git     GitConfig     `json:"git"`
}

type CLIOverrides struct {
	ConfigPath     string
	ReportPath     string
	NonInteractive bool
	SafeSet        bool
	Safe           bool
	AggressiveSet  bool
	Aggressive     bool
	DryRunSet      bool
	DryRun         bool
}

type Effective struct {
	Config         Config              `json:"config"`
	SourceChains   map[string][]string `json:"source_chains"`
	ConfigPath     string              `json:"config_path"`
	ReportPath     string              `json:"report_path"`
	NonInteractive bool                `json:"non_interactive"`
}

func DefaultConfig() Config {
	return Config{
		OpenAI: OpenAIConfig{
			APIKeyEnv: "OPENAI_API_KEY",
			Model:     "gpt-5",
		},
		Modes: ModesConfig{
			Safe:       true,
			Aggressive: false,
			DryRun:     false,
		},
		Profile: ProfileConfig{
			DependencyShortCircuit:   true,
			ShortCircuitEnvVar:       "CoolCodeCleanupShortCircuit",
			UpdateEnvFile:            false,
			SaveShortCircuitToConfig: true,
			EditPermissionMode:       "per-file",
			AutoApply:                false,
		},
		Cleanup: CleanupConfig{
			RemoveRedundantGuards: true,
			DryRefactor:           true,
			HardenErrorHandling:   true,
			GateFeaturesEnv:       false,
			SplitFunctions:        false,
			StandardizeNaming:     true,
			SimplifyComplexLogic:  true,
			DetectExpensive:       true,
			EditPermissionMode:    "per-file",
			AutoApply:             false,
		},
		Git: GitConfig{
			AutoOfferBranchAndCommit: true,
		},
	}
}

func Resolve(cli CLIOverrides) (Effective, error) {
	effective := Effective{
		Config:         DefaultConfig(),
		SourceChains:   map[string][]string{},
		ConfigPath:     cli.ConfigPath,
		ReportPath:     cli.ReportPath,
		NonInteractive: cli.NonInteractive,
	}

	effective.SourceChains["modes.safe"] = []string{SourceDefault}
	effective.SourceChains["modes.aggressive"] = []string{SourceDefault}
	effective.SourceChains["modes.dry_run"] = []string{SourceDefault}
	effective.SourceChains["profile.edit_permission_mode"] = []string{SourceDefault}
	effective.SourceChains["cleanup.edit_permission_mode"] = []string{SourceDefault}
	effective.SourceChains["openai.model"] = []string{SourceDefault}
	effective.SourceChains["profile.short_circuit_env_var"] = []string{SourceDefault}
	effective.SourceChains["profile.dependency_short_circuit"] = []string{SourceDefault}
	effective.SourceChains["cleanup.remove_redundant_guards"] = []string{SourceDefault}
	effective.SourceChains["cleanup.dry_refactor"] = []string{SourceDefault}
	effective.SourceChains["cleanup.harden_error_handling"] = []string{SourceDefault}
	effective.SourceChains["cleanup.gate_features_env"] = []string{SourceDefault}
	effective.SourceChains["cleanup.split_functions"] = []string{SourceDefault}
	effective.SourceChains["cleanup.standardize_naming"] = []string{SourceDefault}
	effective.SourceChains["cleanup.simplify_complex_logic"] = []string{SourceDefault}
	effective.SourceChains["cleanup.detect_expensive_functions"] = []string{SourceDefault}

	cfgFile, exists, err := loadConfigFile(cli.ConfigPath)
	if err != nil {
		return Effective{}, err
	}
	if exists {
		effective.Config = mergeConfig(effective.Config, cfgFile, effective.SourceChains, SourceConfig)
	}

	applyEnv(&effective)
	applyCLI(&effective, cli)

	if err := validate(effective.Config); err != nil {
		return Effective{}, err
	}

	return effective, nil
}

func loadConfigFile(path string) (Config, bool, error) {
	clean := filepath.Clean(path)
	data, err := os.ReadFile(clean)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, false, nil
		}
		return Config{}, false, fmt.Errorf("read config %s: %w", clean, err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, false, fmt.Errorf("parse config %s: %w", clean, err)
	}
	return cfg, true, nil
}

func mergeConfig(base Config, overlay Config, chains map[string][]string, source string) Config {
	if overlay.Modes.Safe != base.Modes.Safe {
		base.Modes.Safe = overlay.Modes.Safe
		chains["modes.safe"] = append(chains["modes.safe"], source)
	}
	if overlay.Modes.Aggressive != base.Modes.Aggressive {
		base.Modes.Aggressive = overlay.Modes.Aggressive
		chains["modes.aggressive"] = append(chains["modes.aggressive"], source)
	}
	if overlay.Modes.DryRun != base.Modes.DryRun {
		base.Modes.DryRun = overlay.Modes.DryRun
		chains["modes.dry_run"] = append(chains["modes.dry_run"], source)
	}
	if overlay.OpenAI.Model != "" && overlay.OpenAI.Model != base.OpenAI.Model {
		base.OpenAI.Model = overlay.OpenAI.Model
		chains["openai.model"] = append(chains["openai.model"], source)
	}
	if overlay.Profile.EditPermissionMode != "" && overlay.Profile.EditPermissionMode != base.Profile.EditPermissionMode {
		base.Profile.EditPermissionMode = overlay.Profile.EditPermissionMode
		chains["profile.edit_permission_mode"] = append(chains["profile.edit_permission_mode"], source)
	}
	if overlay.Profile.ShortCircuitEnvVar != "" && overlay.Profile.ShortCircuitEnvVar != base.Profile.ShortCircuitEnvVar {
		base.Profile.ShortCircuitEnvVar = overlay.Profile.ShortCircuitEnvVar
		chains["profile.short_circuit_env_var"] = append(chains["profile.short_circuit_env_var"], source)
	}
	if overlay.Cleanup.EditPermissionMode != "" && overlay.Cleanup.EditPermissionMode != base.Cleanup.EditPermissionMode {
		base.Cleanup.EditPermissionMode = overlay.Cleanup.EditPermissionMode
		chains["cleanup.edit_permission_mode"] = append(chains["cleanup.edit_permission_mode"], source)
	}
	if overlay.Profile.DependencyShortCircuit != base.Profile.DependencyShortCircuit {
		base.Profile.DependencyShortCircuit = overlay.Profile.DependencyShortCircuit
		chains["profile.dependency_short_circuit"] = append(chains["profile.dependency_short_circuit"], source)
	}
	if overlay.Cleanup.RemoveRedundantGuards != base.Cleanup.RemoveRedundantGuards {
		base.Cleanup.RemoveRedundantGuards = overlay.Cleanup.RemoveRedundantGuards
		chains["cleanup.remove_redundant_guards"] = append(chains["cleanup.remove_redundant_guards"], source)
	}
	if overlay.Cleanup.DryRefactor != base.Cleanup.DryRefactor {
		base.Cleanup.DryRefactor = overlay.Cleanup.DryRefactor
		chains["cleanup.dry_refactor"] = append(chains["cleanup.dry_refactor"], source)
	}
	if overlay.Cleanup.HardenErrorHandling != base.Cleanup.HardenErrorHandling {
		base.Cleanup.HardenErrorHandling = overlay.Cleanup.HardenErrorHandling
		chains["cleanup.harden_error_handling"] = append(chains["cleanup.harden_error_handling"], source)
	}
	if overlay.Cleanup.GateFeaturesEnv != base.Cleanup.GateFeaturesEnv {
		base.Cleanup.GateFeaturesEnv = overlay.Cleanup.GateFeaturesEnv
		chains["cleanup.gate_features_env"] = append(chains["cleanup.gate_features_env"], source)
	}
	if overlay.Cleanup.SplitFunctions != base.Cleanup.SplitFunctions {
		base.Cleanup.SplitFunctions = overlay.Cleanup.SplitFunctions
		chains["cleanup.split_functions"] = append(chains["cleanup.split_functions"], source)
	}
	if overlay.Cleanup.StandardizeNaming != base.Cleanup.StandardizeNaming {
		base.Cleanup.StandardizeNaming = overlay.Cleanup.StandardizeNaming
		chains["cleanup.standardize_naming"] = append(chains["cleanup.standardize_naming"], source)
	}
	if overlay.Cleanup.SimplifyComplexLogic != base.Cleanup.SimplifyComplexLogic {
		base.Cleanup.SimplifyComplexLogic = overlay.Cleanup.SimplifyComplexLogic
		chains["cleanup.simplify_complex_logic"] = append(chains["cleanup.simplify_complex_logic"], source)
	}
	if overlay.Cleanup.DetectExpensive != base.Cleanup.DetectExpensive {
		base.Cleanup.DetectExpensive = overlay.Cleanup.DetectExpensive
		chains["cleanup.detect_expensive_functions"] = append(chains["cleanup.detect_expensive_functions"], source)
	}
	if len(overlay.Profile.IncludeRoutes) > 0 {
		base.Profile.IncludeRoutes = dedupe(overlay.Profile.IncludeRoutes)
		appendSourceIfMissing(chains, "profile.include_routes", source)
	}
	if len(overlay.Profile.IgnoreRoutes) > 0 {
		base.Profile.IgnoreRoutes = dedupe(overlay.Profile.IgnoreRoutes)
		appendSourceIfMissing(chains, "profile.ignore_routes", source)
	}
	return base
}

func applyEnv(e *Effective) {
	if model := strings.TrimSpace(os.Getenv("CCC_OPENAI_MODEL")); model != "" {
		e.Config.OpenAI.Model = model
		e.SourceChains["openai.model"] = append(e.SourceChains["openai.model"], SourceEnv)
	}
	if safe, ok := boolEnv("CCC_SAFE"); ok {
		e.Config.Modes.Safe = safe
		e.SourceChains["modes.safe"] = append(e.SourceChains["modes.safe"], SourceEnv)
	}
	if aggressive, ok := boolEnv("CCC_AGGRESSIVE"); ok {
		e.Config.Modes.Aggressive = aggressive
		e.SourceChains["modes.aggressive"] = append(e.SourceChains["modes.aggressive"], SourceEnv)
	}
	if dryRun, ok := boolEnv("CCC_DRY_RUN"); ok {
		e.Config.Modes.DryRun = dryRun
		e.SourceChains["modes.dry_run"] = append(e.SourceChains["modes.dry_run"], SourceEnv)
	}
	if mode := strings.TrimSpace(os.Getenv("CCC_EDIT_PERMISSION_MODE")); mode != "" {
		e.Config.Profile.EditPermissionMode = mode
		e.Config.Cleanup.EditPermissionMode = mode
		e.SourceChains["profile.edit_permission_mode"] = append(e.SourceChains["profile.edit_permission_mode"], SourceEnv)
		e.SourceChains["cleanup.edit_permission_mode"] = append(e.SourceChains["cleanup.edit_permission_mode"], SourceEnv)
	}
	if include := strings.TrimSpace(os.Getenv("CCC_PROFILE_INCLUDE_ROUTES")); include != "" {
		e.Config.Profile.IncludeRoutes = ParseCSV(include)
		e.SourceChains["profile.include_routes"] = []string{SourceEnv}
	}
	if ignore := strings.TrimSpace(os.Getenv("CCC_PROFILE_IGNORE_ROUTES")); ignore != "" {
		e.Config.Profile.IgnoreRoutes = ParseCSV(ignore)
		e.SourceChains["profile.ignore_routes"] = []string{SourceEnv}
	}
	if shortEnv := strings.TrimSpace(os.Getenv("CCC_PROFILE_SHORT_CIRCUIT_ENV_VAR")); shortEnv != "" {
		e.Config.Profile.ShortCircuitEnvVar = shortEnv
		e.SourceChains["profile.short_circuit_env_var"] = append(e.SourceChains["profile.short_circuit_env_var"], SourceEnv)
	}
	if shortCircuit, ok := boolEnv("CCC_PROFILE_SHORT_CIRCUIT"); ok {
		e.Config.Profile.DependencyShortCircuit = shortCircuit
		e.SourceChains["profile.dependency_short_circuit"] = append(e.SourceChains["profile.dependency_short_circuit"], SourceEnv)
	}
	if v, ok := boolEnv("CCC_CLEANUP_REMOVE_REDUNDANT_GUARDS"); ok {
		e.Config.Cleanup.RemoveRedundantGuards = v
		e.SourceChains["cleanup.remove_redundant_guards"] = append(e.SourceChains["cleanup.remove_redundant_guards"], SourceEnv)
	}
	if v, ok := boolEnv("CCC_CLEANUP_DRY_REFACTOR"); ok {
		e.Config.Cleanup.DryRefactor = v
		e.SourceChains["cleanup.dry_refactor"] = append(e.SourceChains["cleanup.dry_refactor"], SourceEnv)
	}
	if v, ok := boolEnv("CCC_CLEANUP_HARDEN_ERROR_HANDLING"); ok {
		e.Config.Cleanup.HardenErrorHandling = v
		e.SourceChains["cleanup.harden_error_handling"] = append(e.SourceChains["cleanup.harden_error_handling"], SourceEnv)
	}
	if v, ok := boolEnv("CCC_CLEANUP_GATE_FEATURES_ENV"); ok {
		e.Config.Cleanup.GateFeaturesEnv = v
		e.SourceChains["cleanup.gate_features_env"] = append(e.SourceChains["cleanup.gate_features_env"], SourceEnv)
	}
	if v, ok := boolEnv("CCC_CLEANUP_SPLIT_FUNCTIONS"); ok {
		e.Config.Cleanup.SplitFunctions = v
		e.SourceChains["cleanup.split_functions"] = append(e.SourceChains["cleanup.split_functions"], SourceEnv)
	}
	if v, ok := boolEnv("CCC_CLEANUP_STANDARDIZE_NAMING"); ok {
		e.Config.Cleanup.StandardizeNaming = v
		e.SourceChains["cleanup.standardize_naming"] = append(e.SourceChains["cleanup.standardize_naming"], SourceEnv)
	}
	if v, ok := boolEnv("CCC_CLEANUP_SIMPLIFY_COMPLEX_LOGIC"); ok {
		e.Config.Cleanup.SimplifyComplexLogic = v
		e.SourceChains["cleanup.simplify_complex_logic"] = append(e.SourceChains["cleanup.simplify_complex_logic"], SourceEnv)
	}
	if v, ok := boolEnv("CCC_CLEANUP_DETECT_EXPENSIVE_FUNCTIONS"); ok {
		e.Config.Cleanup.DetectExpensive = v
		e.SourceChains["cleanup.detect_expensive_functions"] = append(e.SourceChains["cleanup.detect_expensive_functions"], SourceEnv)
	}
}

func applyCLI(e *Effective, cli CLIOverrides) {
	if cli.SafeSet {
		e.Config.Modes.Safe = cli.Safe
		e.SourceChains["modes.safe"] = append(e.SourceChains["modes.safe"], SourceCLI)
	}
	if cli.AggressiveSet {
		e.Config.Modes.Aggressive = cli.Aggressive
		e.SourceChains["modes.aggressive"] = append(e.SourceChains["modes.aggressive"], SourceCLI)
	}
	if cli.DryRunSet {
		e.Config.Modes.DryRun = cli.DryRun
		e.SourceChains["modes.dry_run"] = append(e.SourceChains["modes.dry_run"], SourceCLI)
	}
}

func validate(cfg Config) error {
	validModes := map[string]bool{
		"per-edit": true,
		"per-file": true,
	}
	if !validModes[cfg.Profile.EditPermissionMode] {
		return fmt.Errorf("invalid profile edit_permission_mode %q (expected per-edit or per-file)", cfg.Profile.EditPermissionMode)
	}
	if !validModes[cfg.Cleanup.EditPermissionMode] {
		return fmt.Errorf("invalid cleanup edit_permission_mode %q (expected per-edit or per-file)", cfg.Cleanup.EditPermissionMode)
	}
	return nil
}

func Save(path string, cfg Config) error {
	clean := filepath.Clean(path)
	if err := os.MkdirAll(filepath.Dir(clean), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.WriteFile(clean, append(out, '\n'), 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// ParseCSV parses a comma-separated string into a de-duplicated slice of trimmed values.
func ParseCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return dedupe(out)
}

func dedupe(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if !slices.Contains(out, item) {
			out = append(out, item)
		}
	}
	return out
}

func appendSourceIfMissing(chains map[string][]string, key, source string) {
	cur := chains[key]
	if !slices.Contains(cur, source) {
		chains[key] = append(cur, source)
	}
}

func boolEnv(name string) (bool, bool) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return false, false
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, false
	}
	return v, true
}
