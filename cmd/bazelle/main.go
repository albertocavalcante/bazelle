// Bazelle is a polyglot BUILD file generator.
package main

import (
	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/cli"

	// Language extensions for gazelle
	"github.com/bazelbuild/bazel-gazelle/language"
	goLang "github.com/bazelbuild/bazel-gazelle/language/go"
	protoLang "github.com/bazelbuild/bazel-gazelle/language/proto"
	kotlinLang "github.com/albertocavalcante/bazelle/gazelle-kotlin/kotlin"
)

// Languages is the list of language extensions for gazelle
var Languages = []language.Language{
	protoLang.NewLanguage(),
	goLang.NewLanguage(),
	kotlinLang.NewLanguage(),
}

func main() {
	cli.SetLanguages(Languages)
	cli.Execute()
}
