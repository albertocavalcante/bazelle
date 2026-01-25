package jvm

import "github.com/bazelbuild/bazel-gazelle/config"

// Config is the interface that all JVM language configs must implement.
type Config interface {
	// IsEnabled returns whether the extension is enabled.
	IsEnabled() bool

	// SetEnabled sets whether the extension is enabled.
	SetEnabled(enabled bool)

	// GetLibraryMacro returns the rule kind to use for libraries.
	GetLibraryMacro() string

	// SetLibraryMacro sets the rule kind to use for libraries.
	SetLibraryMacro(macro string)

	// GetTestMacro returns the rule kind to use for tests.
	GetTestMacro() string

	// SetTestMacro sets the rule kind to use for tests.
	SetTestMacro(macro string)

	// GetVisibility returns the default visibility for generated targets.
	GetVisibility() string

	// SetVisibility sets the default visibility for generated targets.
	SetVisibility(visibility string)

	// GetLoadPath returns the path to load custom macros from.
	GetLoadPath() string

	// SetLoadPath sets the path to load custom macros from.
	SetLoadPath(path string)

	// Clone creates a copy of the config for child directories.
	Clone() Config
}

// BaseConfig provides common configuration fields for JVM language extensions.
// Language-specific configs should embed this struct.
type BaseConfig struct {
	// Enabled indicates whether the extension is enabled.
	Enabled bool

	// LibraryMacro is the rule kind to use for libraries.
	LibraryMacro string

	// TestMacro is the rule kind to use for tests.
	TestMacro string

	// Visibility is the default visibility for generated targets.
	Visibility string

	// LoadPath is the path to load custom macros from.
	LoadPath string
}

// IsEnabled implements Config.
func (c *BaseConfig) IsEnabled() bool {
	return c.Enabled
}

// SetEnabled implements Config.
func (c *BaseConfig) SetEnabled(enabled bool) {
	c.Enabled = enabled
}

// GetLibraryMacro implements Config.
func (c *BaseConfig) GetLibraryMacro() string {
	return c.LibraryMacro
}

// SetLibraryMacro implements Config.
func (c *BaseConfig) SetLibraryMacro(macro string) {
	c.LibraryMacro = macro
}

// GetTestMacro implements Config.
func (c *BaseConfig) GetTestMacro() string {
	return c.TestMacro
}

// SetTestMacro implements Config.
func (c *BaseConfig) SetTestMacro(macro string) {
	c.TestMacro = macro
}

// GetVisibility implements Config.
func (c *BaseConfig) GetVisibility() string {
	return c.Visibility
}

// SetVisibility implements Config.
func (c *BaseConfig) SetVisibility(visibility string) {
	c.Visibility = visibility
}

// GetLoadPath implements Config.
func (c *BaseConfig) GetLoadPath() string {
	return c.LoadPath
}

// SetLoadPath implements Config.
func (c *BaseConfig) SetLoadPath(path string) {
	c.LoadPath = path
}

// CloneBase creates a copy of the BaseConfig.
func (c *BaseConfig) CloneBase() BaseConfig {
	return BaseConfig{
		Enabled:      c.Enabled,
		LibraryMacro: c.LibraryMacro,
		TestMacro:    c.TestMacro,
		Visibility:   c.Visibility,
		LoadPath:     c.LoadPath,
	}
}

// NewBaseConfig creates a new BaseConfig with default values.
func NewBaseConfig(lang Language) BaseConfig {
	switch lang {
	case Kotlin:
		return BaseConfig{
			Enabled:      false,
			LibraryMacro: "kt_jvm_library",
			TestMacro:    "kt_jvm_test",
			Visibility:   "//visibility:public",
			LoadPath:     "",
		}
	case Groovy:
		return BaseConfig{
			Enabled:      false,
			LibraryMacro: "groovy_library",
			TestMacro:    "groovy_test",
			Visibility:   "//visibility:public",
			LoadPath:     "",
		}
	case Java:
		return BaseConfig{
			Enabled:      false,
			LibraryMacro: "java_library",
			TestMacro:    "java_test",
			Visibility:   "//visibility:public",
			LoadPath:     "",
		}
	case Scala:
		return BaseConfig{
			Enabled:      false,
			LibraryMacro: "scala_library",
			TestMacro:    "scala_test",
			Visibility:   "//visibility:public",
			LoadPath:     "",
		}
	default:
		return BaseConfig{
			Enabled:    false,
			Visibility: "//visibility:public",
		}
	}
}

// GetConfig extracts a JVM config from the Gazelle config using the given key.
func GetConfig[T Config](c *config.Config, key string) T {
	cfg, ok := c.Exts[key].(T)
	if !ok {
		var zero T
		return zero
	}
	return cfg
}

// SetConfig stores a JVM config in the Gazelle config using the given key.
func SetConfig(c *config.Config, key string, cfg Config) {
	c.Exts[key] = cfg
}
