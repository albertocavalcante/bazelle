#!/bin/bash
:<<"::CMDLITERAL"
@ECHO OFF
GOTO :CMDSCRIPT
::CMDLITERAL

# Go packages driver for Bazel integration with IDEs.
#
# This script allows gopls and other Go tools to understand Bazel-managed
# Go packages, providing IDE features like autocomplete and go-to-definition.
#
# Setup:
#   export GOPACKAGESDRIVER=$PWD/tools/gopackagesdriver.cmd
#   # Or add to your shell profile / IDE settings
#
# See: https://github.com/bazelbuild/rules_go/wiki/Editor-setup

set -euo pipefail
root="$(cd "$(dirname "$0")/.."; pwd)"
exec "$root/bazel.cmd" run -- @rules_go//go/tools/gopackagesdriver "$@"

:CMDSCRIPT
REM Go packages driver for Bazel integration with IDEs.
REM
REM This script allows gopls and other Go tools to understand Bazel-managed
REM Go packages, providing IDE features like autocomplete and go-to-definition.
REM
REM Setup:
REM   set GOPACKAGESDRIVER=%CD%\tools\gopackagesdriver.cmd
REM   # Or add to your shell profile / IDE settings
REM
REM See: https://github.com/bazelbuild/rules_go/wiki/Editor-setup

"%~dp0..\bazel.cmd" run -- @rules_go//go/tools/gopackagesdriver %*
