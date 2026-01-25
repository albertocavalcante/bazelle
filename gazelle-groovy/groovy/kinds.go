package groovy

import (
	"github.com/albertocavalcante/bazelle/pkg/jvm"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Kinds implements language.Language.
func (*groovyLang) Kinds() map[string]rule.KindInfo {
	return jvm.StandardGroovyKinds()
}

// Loads implements language.Language.
func (*groovyLang) Loads() []rule.LoadInfo {
	return []rule.LoadInfo{
		{
			Name:    "@rules_groovy//groovy:groovy.bzl",
			Symbols: []string{"groovy_library", "groovy_test", "spock_test"},
		},
	}
}
