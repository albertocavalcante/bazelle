module github.com/albertocavalcante/bazelle

go 1.25

require (
	github.com/EngFlow/gazelle_cc v0.5.0
	github.com/bazel-contrib/rules_python/gazelle v0.0.0-20260120082853-1ac5a19c30c3
	github.com/bazelbuild/bazel-gazelle v0.47.0
	github.com/bazelbuild/bazel-skylib v0.0.0-20251220030559-ea054fcaf08c
	github.com/calsign/gazelle_rust v0.0.0-20260116161429-c029702902b7
	github.com/kisielk/errcheck v1.9.0
	github.com/malivvan/tree-sitter v0.0.1
	github.com/smacker/go-tree-sitter v0.0.0-20240827094217-dd81d9e9be82
	github.com/spf13/cobra v1.10.2
	go.uber.org/nilaway v0.0.0-20251208195206-89df5f7e6199
	golang.org/x/tools v0.36.0
)

replace github.com/bazelbuild/bazel-gazelle => github.com/albertocavalcante/fork-bazel-gazelle v0.0.0-20260120124537-c16e1e6df9fc

require (
	github.com/bazelbuild/buildtools v0.0.0-20250930140053-2eb4fccefb52 // indirect
	github.com/bmatcuk/doublestar/v4 v4.9.1 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/tetratelabs/wazero v1.8.2 // indirect
	golang.org/x/mod v0.27.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/tools/go/vcs v0.1.0-deprecated // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
