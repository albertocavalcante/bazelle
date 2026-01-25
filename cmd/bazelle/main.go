// Bazelle is a polyglot BUILD file generator.
package main

import (
	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/cli"
	"github.com/albertocavalcante/bazelle/pkg/config"
	"github.com/albertocavalcante/bazelle/pkg/registry"
)

func main() {
	// Load configuration from files (built-in -> user -> project -> env -> flags)
	cfg := config.Load()

	// Load languages based on configuration
	languages := registry.LoadLanguages(cfg)

	cli.SetLanguages(languages)
	cli.Execute()
}
