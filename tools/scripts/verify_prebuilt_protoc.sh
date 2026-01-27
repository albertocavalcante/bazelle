#!/usr/bin/env bash
# Verify that @protobuf//:protoc (compiled from source) is NOT in the dependency graph.
# This ensures we're using prebuilt protoc via proto toolchain resolution.
#
# Usage: bazel run //tools/scripts:verify_prebuilt_protoc
#    or: ./tools/scripts/verify_prebuilt_protoc.sh [target]
#
# References:
# - https://github.com/grpc/grpc-java/issues/11152
# - https://blog.aspect.build/bazel-9-protobuf

set -euo pipefail

TARGET="${1:-//:bazelle}"

echo "Checking that @protobuf//:protoc is NOT a dependency of ${TARGET}..."

# Query for any path from target to the compiled protoc binary
RESULT=$(bazel query "somepath(${TARGET}, @protobuf//:protoc)" 2>/dev/null || true)

if [ -n "$RESULT" ]; then
    echo "ERROR: Found dependency path to @protobuf//:protoc (compiled from source)!"
    echo "This means protoc will be compiled from source instead of using prebuilt."
    echo ""
    echo "Dependency path:"
    echo "$RESULT"
    echo ""
    echo "To fix: Check that grpc-java patch is applied correctly."
    echo "See: third_party/patches/grpc_java_proto_toolchain/README.md"
    exit 1
fi

echo "OK: @protobuf//:protoc is NOT in dependency graph."
echo "Prebuilt protoc will be used via proto toolchain resolution."
