// Bazelle is a polyglot BUILD file generator.
package main

import (
	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/cli"

	// Language extensions for gazelle
	"github.com/bazelbuild/bazel-gazelle/language"
	goLang "github.com/bazelbuild/bazel-gazelle/language/go"
	protoLang "github.com/bazelbuild/bazel-gazelle/language/proto"

	// External language extensions
	bzlLang "github.com/bazelbuild/bazel-skylib/gazelle"
	ccLang "github.com/EngFlow/gazelle_cc/language/cc"
	pythonLang "github.com/bazel-contrib/rules_python/gazelle/python"
	kotlinLang "github.com/albertocavalcante/bazelle/gazelle-kotlin/kotlin"
)

// Languages is the list of language extensions for gazelle
// Order matters: proto should come first, then language-specific extensions
var Languages = []language.Language{
	protoLang.NewLanguage(),
	goLang.NewLanguage(),
	bzlLang.NewLanguage(),
	pythonLang.NewLanguage(),
	ccLang.NewLanguage(),
	kotlinLang.NewLanguage(),
}

func main() {
	cli.SetLanguages(Languages)
	cli.Execute()
}
