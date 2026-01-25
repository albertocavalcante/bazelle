package kotlin

import (
	"github.com/albertocavalcante/bazelle/pkg/jvm"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Kinds implements language.Language.
func (*kotlinLang) Kinds() map[string]rule.KindInfo {
	return jvm.StandardKotlinKinds()
}

// Loads implements language.Language.
func (k *kotlinLang) Loads() []rule.LoadInfo {
	return []rule.LoadInfo{
		{
			Name:    "@rules_kotlin//kotlin:jvm.bzl",
			Symbols: []string{"kt_jvm_library", "kt_jvm_test", "kt_jvm_binary"},
		},
	}
}
