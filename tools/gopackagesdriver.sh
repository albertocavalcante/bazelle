#!/usr/bin/env bash
# Go packages driver for Bazel integration with IDEs.
#
# This script allows gopls and other Go tools to understand Bazel-managed
# Go packages, providing IDE features like autocomplete and go-to-definition.
#
# Setup:
#   export GOPACKAGESDRIVER=$PWD/tools/gopackagesdriver.sh
#   # Or add to your shell profile / IDE settings
#
# See: https://github.com/bazelbuild/rules_go/wiki/Editor-setup

set -euo pipefail

exec bazel run -- @rules_go//go/tools/gopackagesdriver "${@}"
