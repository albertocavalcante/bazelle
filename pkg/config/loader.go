package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// ConfigFileName is the name of the project-level config file.
const ConfigFileName = "bazelle.toml"

// ConfigDirName is the name of the project-level config directory.
const ConfigDirName = ".bazelle"

// GlobalConfigDir is the name of the global config directory inside user's config.
const GlobalConfigDir = "bazelle"

// Load loads configuration from all layers in order of precedence:
//  1. Built-in defaults
//  2. Global user config (~/.config/bazelle/config.toml)
//  3. Project config (.bazelle/config.toml or bazelle.toml)
//  4. Environment variables (BAZELLE_*)
//
// CLI flags are applied separately after Load() returns.
func Load() *Config {
	cfg := NewConfig()

	// Layer 2: Global user config
	if globalCfg := loadGlobalConfig(); globalCfg != nil {
		cfg.Merge(globalCfg)
	}

	// Layer 3: Project config
	if projectCfg := loadProjectConfig(); projectCfg != nil {
		cfg.Merge(projectCfg)
	}

	// Layer 4: Environment variables
	applyEnvironmentVariables(cfg)

	return cfg
}

// LoadFrom loads configuration starting from a specific directory.
func LoadFrom(dir string) *Config {
	cfg := NewConfig()

	// Layer 2: Global user config
	if globalCfg := loadGlobalConfig(); globalCfg != nil {
		cfg.Merge(globalCfg)
	}

	// Layer 3: Project config from specified directory
	if projectCfg := loadProjectConfigFrom(dir); projectCfg != nil {
		cfg.Merge(projectCfg)
	}

	// Layer 4: Environment variables
	applyEnvironmentVariables(cfg)

	return cfg
}

// loadGlobalConfig loads the global user configuration from ~/.config/bazelle/config.toml.
func loadGlobalConfig() *Config {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil
	}

	configPath := filepath.Join(configDir, GlobalConfigDir, "config.toml")
	return loadConfigFile(configPath)
}

// loadProjectConfig looks for project configuration in the current directory and parents.
func loadProjectConfig() *Config {
	wd, err := os.Getwd()
	if err != nil {
		return nil
	}
	return loadProjectConfigFrom(wd)
}

// loadProjectConfigFrom looks for project configuration starting from the given directory.
func loadProjectConfigFrom(dir string) *Config {
	// Search up the directory tree for config files
	current := dir
	for {
		// Check for .bazelle/config.toml first
		bazelleDir := filepath.Join(current, ConfigDirName, "config.toml")
		if cfg := loadConfigFile(bazelleDir); cfg != nil {
			return cfg
		}

		// Check for bazelle.toml in project root
		bazelleToml := filepath.Join(current, ConfigFileName)
		if cfg := loadConfigFile(bazelleToml); cfg != nil {
			return cfg
		}

		// Stop at filesystem root or git/bazel workspace root
		if isWorkspaceRoot(current) {
			break
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return nil
}

// isWorkspaceRoot checks if the directory is a workspace root (has .git, WORKSPACE, or MODULE.bazel).
func isWorkspaceRoot(dir string) bool {
	markers := []string{".git", "WORKSPACE", "WORKSPACE.bazel", "MODULE.bazel"}
	for _, marker := range markers {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	return false
}

// loadConfigFile loads a configuration from a TOML file.
func loadConfigFile(path string) *Config {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var cfg Config
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return nil
	}

	return &cfg
}

// applyEnvironmentVariables applies BAZELLE_* environment variables to the config.
func applyEnvironmentVariables(cfg *Config) {
	// BAZELLE_LANGUAGES_ENABLED: comma-separated list of languages to enable
	if langs := os.Getenv("BAZELLE_LANGUAGES_ENABLED"); langs != "" {
		cfg.Languages.Enabled = splitAndTrim(langs)
	}

	// BAZELLE_LANGUAGES_DISABLED: comma-separated list of languages to disable
	if langs := os.Getenv("BAZELLE_LANGUAGES_DISABLED"); langs != "" {
		cfg.Languages.Disabled = splitAndTrim(langs)
	}

	// Language-specific enabled flags
	applyBoolEnv("BAZELLE_GO_ENABLED", &cfg.Go.Enabled)
	applyBoolEnv("BAZELLE_KOTLIN_ENABLED", &cfg.Kotlin.Enabled)
	applyBoolEnv("BAZELLE_PYTHON_ENABLED", &cfg.Python.Enabled)
	applyBoolEnv("BAZELLE_JAVA_ENABLED", &cfg.Java.Enabled)
	applyBoolEnv("BAZELLE_SCALA_ENABLED", &cfg.Scala.Enabled)
	applyBoolEnv("BAZELLE_GROOVY_ENABLED", &cfg.Groovy.Enabled)
	applyBoolEnv("BAZELLE_PROTO_ENABLED", &cfg.Proto.Enabled)
	applyBoolEnv("BAZELLE_RUST_ENABLED", &cfg.Rust.Enabled)
	applyBoolEnv("BAZELLE_CC_ENABLED", &cfg.CC.Enabled)
	applyBoolEnv("BAZELLE_BZL_ENABLED", &cfg.Bzl.Enabled)

	// Go-specific settings
	if v := os.Getenv("BAZELLE_GO_NAMING_CONVENTION"); v != "" {
		cfg.Go.NamingConvention = v
	}
	if v := os.Getenv("BAZELLE_GO_NAMING_CONVENTION_EXTERNAL"); v != "" {
		cfg.Go.NamingConventionExternal = v
	}

	// Kotlin-specific settings
	if v := os.Getenv("BAZELLE_KOTLIN_PARSER_BACKEND"); v != "" {
		cfg.Kotlin.ParserBackend = v
	}
	applyBoolEnv("BAZELLE_KOTLIN_FQN_SCANNING", &cfg.Kotlin.FQNScanning)

	// Python-specific settings
	if v := os.Getenv("BAZELLE_PYTHON_STDLIB_MODULES_FILE"); v != "" {
		cfg.Python.StdlibModulesFile = v
	}
	if v := os.Getenv("BAZELLE_PYTHON_TEST_FRAMEWORK"); v != "" {
		cfg.Python.TestFramework = v
	}
}

// splitAndTrim splits a comma-separated string and trims whitespace.
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// applyBoolEnv applies a boolean environment variable to a pointer.
func applyBoolEnv(envVar string, target **bool) {
	if v := os.Getenv(envVar); v != "" {
		v = strings.ToLower(v)
		if v == "true" || v == "1" || v == "yes" {
			t := true
			*target = &t
		} else if v == "false" || v == "0" || v == "no" {
			f := false
			*target = &f
		}
	}
}

// GetGlobalConfigPath returns the path to the global config file.
func GetGlobalConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(configDir, GlobalConfigDir, "config.toml")
}

// GetProjectConfigPaths returns potential project config paths for a given directory.
func GetProjectConfigPaths(dir string) []string {
	return []string{
		filepath.Join(dir, ConfigDirName, "config.toml"),
		filepath.Join(dir, ConfigFileName),
	}
}
