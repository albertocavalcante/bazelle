# Gazelle-Kotlin

Gazelle extension for Kotlin projects using [rules_kotlin](https://github.com/bazelbuild/rules_kotlin).

## Features

- Generate `kt_jvm_library` targets from `src/main/kotlin/**/*.kt`
- Generate `kt_jvm_test` targets from `src/test/kotlin/**/*.kt`
- Auto-detect test packages for JUnit discovery
- Support for custom macros via directives

## Installation

Add to your `MODULE.bazel`:

```starlark
bazel_dep(name = "gazelle_kotlin", version = "0.1.0")
```

## Usage

Add to your root `BUILD.bazel`:

```starlark
load("@gazelle//:def.bzl", "gazelle")
load("@gazelle_kotlin//:def.bzl", "gazelle_kotlin_binary")

gazelle(
    name = "gazelle",
    gazelle = "@gazelle_kotlin//:gazelle-kotlin",
)
```

Run Gazelle:

```bash
bazel run //:gazelle
```

## Directives

| Directive | Default | Description |
|-----------|---------|-------------|
| `kotlin_enabled` | `false` | Enable Kotlin extension |
| `kotlin_library_macro` | `kt_jvm_library` | Library rule kind |
| `kotlin_test_macro` | `kt_jvm_test` | Test rule kind |
| `kotlin_visibility` | `//visibility:public` | Default visibility |

Example:

```starlark
# gazelle:kotlin_enabled true
# gazelle:kotlin_library_macro kt_library
# gazelle:kotlin_test_macro kt_test
```

## License

Apache-2.0
