# grpc-java Proto Toolchain Patch

This directory contains source files for generating the grpc-java proto toolchain patch.

## Problem

grpc-java's `java_grpc_library.bzl` hardcodes `@com_google_protobuf//:protoc` as a dependency,
causing protoc to be compiled from source (~5min) even when prebuilt protoc is available.
See: https://github.com/grpc/grpc-java/issues/11152

## Solution

Patch `java_grpc_library.bzl` to use proto toolchain resolution (enabled by default in Bazel 9)
to get prebuilt protoc when `--@protobuf//bazel/toolchains:prefer_prebuilt_protoc` is set.

## Files

- `java_grpc_library.bzl.original` - Original file from grpc-java v1.70.0
- `java_grpc_library.bzl.modified` - Our patched version using proto toolchain

## Regenerating the Patch

```bash
# Generate patch for java_grpc_library.bzl changes
diff -u java_grpc_library.bzl.original java_grpc_library.bzl.modified > java_grpc_library.bzl.patch

# The actual patch applied is in ../contrib_rules_jvm_proto_toolchain.patch
# which patches contrib_rules_jvm's grpc-java.patch file
```

## References

- https://github.com/grpc/grpc-java/issues/11152
- https://blog.aspect.build/bazel-9-protobuf
- https://bazel.build/reference/command-line-reference#flag--incompatible_enable_proto_toolchain_resolution
