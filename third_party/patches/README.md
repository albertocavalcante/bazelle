# Third-Party Patches for Bazel 9 Compatibility

This directory contains patches for third-party dependencies that have Bazel 9 compatibility issues.

## Java/contrib_rules_jvm - ProtoInfo Issue

### Problem

`contrib_rules_jvm` (v0.31.1) depends on `grpc-java` (v1.71.0), which uses the built-in `ProtoInfo` symbol. Bazel 9 removed built-in `ProtoInfo` - it must now be loaded from `@rules_proto//proto:defs.bzl`.

**Error:**
```
name 'ProtoInfo' is not defined
```

### Root Cause

- **grpc-java 1.71.0**: Uses built-in `ProtoInfo` (works on Bazel 7/8, fails on Bazel 9)
- **grpc-java 1.76+**: Loads `ProtoInfo` from `@rules_proto` (works on Bazel 9)
- **contrib_rules_jvm**: Pins grpc-java to 1.71.0 via `http_archive`

### References

- Issue: https://github.com/grpc/grpc-java/issues/12315
- Fix PR: https://github.com/grpc/grpc-java/pull/12312
- grpc-java releases: https://github.com/grpc/grpc-java/releases
- BCR grpc-java: https://registry.bazel.build/modules/grpc-java (latest: 1.75.0)

### Fix Options

#### Option 1: Wait for Upstream (Recommended)

Wait for either:
- `contrib_rules_jvm` to update to grpc-java >= 1.76
- BCR to add grpc-java >= 1.76

#### Option 2: Patch contrib_rules_jvm

Patch `contrib_rules_jvm` to:
1. Update grpc-java from 1.71.0 to 1.78.0
2. Update `third_party/grpc-java.patch` to work with 1.78.0's BUILD files

**Challenges:**
- contrib_rules_jvm's `grpc-java.patch` adds `repository_name = "contrib_rules_jvm_deps"` to all `artifact()` calls
- This patch was written for grpc-java 1.71.0's BUILD file structure
- grpc-java 1.78.0 has different BUILD files, so the patch doesn't apply

#### Option 3: Patch grpc-java's java_grpc_library.bzl

Add this load statement to the beginning of `java_grpc_library.bzl`:
```starlark
load("@rules_proto//proto:defs.bzl", "ProtoInfo")
```

**Challenges:**
- Requires patching contrib_rules_jvm's `third_party/grpc-java.patch` to include this fix
- Patch-within-patch format is error-prone

### Patch Files in This Directory

| File | Purpose | Status |
|------|---------|--------|
| `contrib_rules_jvm_grpc_java_1.78.patch` | Attempt to patch MODULE.bazel + grpc-java.patch | WIP |
| `contrib_rules_jvm_grpc_java_1.78.patch.v1` | Earlier attempt (grpc-java 1.78 without patches) | Failed (@maven not found) |
| `grpc_java_protoinfo.patch` | Standalone ProtoInfo fix for java_grpc_library.bzl | Reference |

### Current Status

Java support is **DISABLED** in bazelle until upstream fixes are available.

To re-enable Java when ready:
1. Uncomment Java deps in `MODULE.bazel`
2. Uncomment `@contrib_rules_jvm//java/gazelle` in `cmd/bazelle/BUILD.bazel`
3. Test with `bazel build //cmd/bazelle`
