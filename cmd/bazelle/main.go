// Bazelle is a polyglot BUILD file generator.
package main

import (
	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/cli"
	"github.com/albertocavalcante/bazelle/internal/log"
	"github.com/albertocavalcante/bazelle/pkg/config"
	"github.com/albertocavalcante/bazelle/pkg/registry"
)

func main() {
	// Bootstrap logger with defaults (before config loads)
	// This allows logging during config loading
	log.Init(1, "text")

	// Load configuration from files (built-in -> user -> project -> env -> flags)
	cfg := config.Load()

	// Re-initialize logger with config values
	log.Init(cfg.Log.Verbosity, cfg.Log.Format)

	// Load languages based on configuration
	languages := registry.LoadLanguages(cfg)

	cli.SetLanguages(languages)
	cli.Execute()
}
