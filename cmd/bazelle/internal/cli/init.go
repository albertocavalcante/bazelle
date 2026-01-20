package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/detect"
)

var initFlags struct {
	languages []string
	check     bool
	dryRun    bool
	name      string
}

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize a bazel project with gazelle support",
	Long: `Initializes a bazel project with the necessary configuration for gazelle.

This command will:
1. Detect languages used in your project
2. Create or update MODULE.bazel with required dependencies
3. Create or update root BUILD.bazel with gazelle targets

Use --check to verify configuration without making changes (useful for CI).
Use --dry-run to preview changes without applying them.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringSliceVarP(&initFlags.languages, "languages", "l", nil,
		"Languages to configure (auto-detected if not specified)")
	initCmd.Flags().BoolVar(&initFlags.check, "check", false,
		"Check if project is properly configured (exit 1 if not)")
	initCmd.Flags().BoolVar(&initFlags.dryRun, "dry-run", false,
		"Show what would change without applying")
	initCmd.Flags().StringVar(&initFlags.name, "name", "",
		"Module name (defaults to directory name)")

	rootCmd.AddCommand(initCmd)
}

// langDependencies maps language to required bazel_dep entries
var langDependencies = map[string][]dependency{
	"go": {
		{name: "rules_go", version: "0.59.0"},
		{name: "gazelle", version: "0.47.0"},
	},
	"kotlin": {
		{name: "rules_go", version: "0.59.0"},
		{name: "gazelle", version: "0.47.0"},
		{name: "rules_kotlin", version: "2.2.2"},
	},
	"python": {
		{name: "rules_go", version: "0.59.0"},
		{name: "gazelle", version: "0.47.0"},
		{name: "rules_python", version: "1.8.0"},
		{name: "rules_python_gazelle_plugin", version: "1.6.3"},
	},
	"proto": {
		{name: "rules_go", version: "0.59.0"},
		{name: "gazelle", version: "0.47.0"},
		{name: "rules_proto", version: "7.1.0"},
	},
	"cc": {
		{name: "rules_go", version: "0.59.0"},
		{name: "gazelle", version: "0.47.0"},
		{name: "rules_cc", version: "0.2.16"},
		{name: "gazelle_cc", version: "0.4.0"},
	},
}

type dependency struct {
	name    string
	version string
}

func runInit(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Determine languages
	var languages []string
	if len(initFlags.languages) > 0 {
		languages = initFlags.languages
	} else {
		detected, err := detect.Languages(absPath)
		if err != nil {
			return fmt.Errorf("failed to detect languages: %w", err)
		}
		languages = detected
	}

	if len(languages) == 0 {
		fmt.Println("No languages detected. Use --languages to specify manually.")
		return nil
	}

	fmt.Printf("Languages: %s\n", strings.Join(languages, ", "))

	// Determine module name
	moduleName := initFlags.name
	if moduleName == "" {
		moduleName = filepath.Base(absPath)
	}

	// Check existing files
	moduleFile := filepath.Join(absPath, "MODULE.bazel")
	buildFile := filepath.Join(absPath, "BUILD.bazel")

	moduleExists := fileExists(moduleFile)
	buildExists := fileExists(buildFile)

	// Collect required dependencies
	deps := collectDependencies(languages)

	if initFlags.check {
		return runInitCheck(moduleExists, buildExists, moduleFile, buildFile, deps)
	}

	// Generate content
	moduleContent := generateModuleContent(moduleName, deps)
	buildContent := generateBuildContent()

	if initFlags.dryRun {
		return runInitDryRun(moduleExists, buildExists, moduleFile, buildFile, moduleContent, buildContent)
	}

	// Write files
	return runInitApply(moduleExists, buildExists, moduleFile, buildFile, moduleContent, buildContent)
}

func collectDependencies(languages []string) []dependency {
	seen := make(map[string]dependency)
	var deps []dependency

	for _, lang := range languages {
		for _, dep := range langDependencies[lang] {
			if _, ok := seen[dep.name]; !ok {
				seen[dep.name] = dep
				deps = append(deps, dep)
			}
		}
	}

	// Sort for deterministic output
	slices.SortFunc(deps, func(a, b dependency) int {
		return strings.Compare(a.name, b.name)
	})

	return deps
}

func generateModuleContent(name string, deps []dependency) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`module(
    name = "%s",
    version = "0.1.0",
)

`, name))

	sb.WriteString("# Core dependencies\n")
	sb.WriteString(`bazel_dep(name = "bazel_skylib", version = "1.9.0")`)
	sb.WriteString("\n\n")

	sb.WriteString("# Language support\n")
	for _, dep := range deps {
		sb.WriteString(fmt.Sprintf(`bazel_dep(name = "%s", version = "%s")`, dep.name, dep.version))
		sb.WriteString("\n")
	}

	return sb.String()
}

func generateBuildContent() string {
	return `load("@gazelle//:def.bzl", "gazelle")

# gazelle:prefix github.com/your-org/your-repo

gazelle(
    name = "gazelle",
)
`
}

func runInitCheck(moduleExists, buildExists bool, moduleFile, buildFile string, deps []dependency) error {
	issues := []string{}

	if !moduleExists {
		issues = append(issues, fmt.Sprintf("MODULE.bazel not found at %s", moduleFile))
	} else {
		// TODO: Check if required deps are present
		content, err := os.ReadFile(moduleFile)
		if err == nil {
			for _, dep := range deps {
				if !strings.Contains(string(content), dep.name) {
					issues = append(issues, fmt.Sprintf("MODULE.bazel missing dependency: %s", dep.name))
				}
			}
		}
	}

	if !buildExists {
		issues = append(issues, fmt.Sprintf("BUILD.bazel not found at %s", buildFile))
	} else {
		content, err := os.ReadFile(buildFile)
		if err == nil {
			if !strings.Contains(string(content), "gazelle") {
				issues = append(issues, "BUILD.bazel missing gazelle target")
			}
		}
	}

	if len(issues) > 0 {
		fmt.Fprintln(os.Stderr, "Project configuration issues:")
		for _, issue := range issues {
			fmt.Fprintf(os.Stderr, "  - %s\n", issue)
		}
		fmt.Fprintln(os.Stderr, "\nRun 'bazelle init' to fix")
		os.Exit(1)
	}

	fmt.Println("Project is properly configured")
	return nil
}

func runInitDryRun(moduleExists, buildExists bool, moduleFile, buildFile, moduleContent, buildContent string) error {
	if !moduleExists {
		fmt.Printf("Would create %s:\n", moduleFile)
		fmt.Println(moduleContent)
	} else {
		fmt.Printf("MODULE.bazel exists at %s (would not modify)\n", moduleFile)
	}

	fmt.Println()

	if !buildExists {
		fmt.Printf("Would create %s:\n", buildFile)
		fmt.Println(buildContent)
	} else {
		fmt.Printf("BUILD.bazel exists at %s (would not modify)\n", buildFile)
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func runInitApply(moduleExists, buildExists bool, moduleFile, buildFile, moduleContent, buildContent string) error {
	if !moduleExists {
		if err := os.WriteFile(moduleFile, []byte(moduleContent), 0o644); err != nil {
			return fmt.Errorf("failed to write MODULE.bazel: %w", err)
		}
		fmt.Printf("Created %s\n", moduleFile)
	} else {
		fmt.Printf("MODULE.bazel already exists (skipping)\n")
	}

	if !buildExists {
		if err := os.WriteFile(buildFile, []byte(buildContent), 0o644); err != nil {
			return fmt.Errorf("failed to write BUILD.bazel: %w", err)
		}
		fmt.Printf("Created %s\n", buildFile)
	} else {
		fmt.Printf("BUILD.bazel already exists (skipping)\n")
	}

	fmt.Println("\nNext steps:")
	fmt.Println("  1. Update the gazelle:prefix directive in BUILD.bazel")
	fmt.Println("  2. Run 'bazelle update' to generate BUILD files")

	return nil
}
