# Bazel Protobuf Build Optimization

This directory contains toolchain configurations to optimize Protocol Buffer builds in Bazel by using prebuilt `protoc` binaries instead of compiling from source.

## Problem Solved

By default, Bazel builds protoc from C++ source code during each build, which:
- Takes significant time (especially on first build)
- Often results in cache misses due to environmental sensitivity
- Blocks parallel builds while protoc compiles
- Is unnecessary when only generating Go code from protos

## Solution

We use the `toolchains_protoc` repository to provide prebuilt protoc binaries, which:
- Downloads prebuilt binaries instead of compiling from source
- Caches effectively across builds and machines
- Significantly reduces build times
- Works consistently across different environments

## Configuration

### 1. WORKSPACE Configuration

The main optimization is configured in the root `WORKSPACE` file:

```python
# Prebuilt protoc toolchain for faster builds
http_archive(
    name = "toolchains_protoc",
    sha256 = "3019f9ed1273d547334da2004e634340c896d9e24dd6d899911e03b694fdc1f5",
    strip_prefix = "toolchains_protoc-0.4.3", 
    urls = [
        "https://github.com/aspect-build/toolchains_protoc/releases/download/v0.4.3/toolchains_protoc-v0.4.3.tar.gz",
    ],
)

# Register prebuilt protoc toolchains
load("@toolchains_protoc//protoc:toolchain.bzl", "protoc_toolchains")

protoc_toolchains(
    name = "protoc_toolchains",
    version = "28.3",  # Use a stable protoc version
)
```

### 2. .bazelrc Configuration  

The `.bazelrc` file contains optimization flags:

```bash
# Protocol buffer optimization settings
common --incompatible_enable_proto_toolchain_resolution

# Prevent accidental source builds of protobuf/gRPC (fail fast if they try to build from source)
common --per_file_copt=external/.*protobuf.*@--PROTOBUF_WAS_NOT_SUPPOSED_TO_BE_BUILT
common --host_per_file_copt=external/.*protobuf.*@--PROTOBUF_WAS_NOT_SUPPOSED_TO_BE_BUILT
common --per_file_copt=external/.*grpc.*@--GRPC_WAS_NOT_SUPPOSED_TO_BE_BUILT
common --host_per_file_copt=external/.*grpc.*@--GRPC_WAS_NOT_SUPPOSED_TO_BE_BUILT
```

### 3. Custom Go Toolchains

This directory provides custom Go protobuf toolchains in `BUILD.bazel`:

```python
# Go protobuf toolchain using prebuilt protoc
proto_lang_toolchain(
    name = "protoc_go_toolchain",
    command_line = "--go_out=%s --go_opt=paths=source_relative",
    progress_message = "Generating Go proto_library %{label}",
    runtime = "@org_golang_google_protobuf//:protobuf_go",
    toolchain_type = "@rules_go//proto:toolchain_type",
)
```

## Usage

After configuration, continue using Bazel normally:

```bash
# Clean build to see the performance improvement
bazel clean --expunge
bazel build //...
```

## Performance Benefits

Expected improvements:
- **First build**: 30-60% faster due to no protoc compilation
- **Incremental builds**: Much faster as protoc binary is cached
- **CI builds**: Consistent performance across runs
- **Parallel builds**: No blocking on protoc compilation

## Verification

To verify the optimization is working:

1. **Check toolchain usage**:
   ```bash
   bazel build //... --toolchain_resolution_debug
   ```

2. **Monitor build actions**:
   ```bash
   bazel build //... --execution_log_binary_file=/tmp/execution.log
   ```

3. **Verify no source compilation**:
   The fail-fast flags in `.bazelrc` will cause builds to fail if protobuf/gRPC
   tries to compile from source, helping identify configuration issues.

## Troubleshooting

### Build fails with "PROTOBUF_WAS_NOT_SUPPOSED_TO_BE_BUILT"

This means protobuf is trying to build from source. Check:
- Toolchain registration in WORKSPACE
- Proto toolchain configuration
- Dependencies that might force source builds

### Slow builds persist

- Verify toolchains are registered correctly
- Check that `--incompatible_enable_proto_toolchain_resolution` is set
- Ensure remote caching is enabled if available

### Version conflicts

- Update toolchains_protoc to latest version
- Ensure protoc version matches project requirements
- Check compatibility with rules_proto version

## Remote Caching

To maximize benefits, enable Bazel remote caching:

```bash
# In .bazelrc
common --remote_cache=https://your-cache-endpoint
common --remote_upload_local_results
```

This allows sharing the prebuilt protoc binaries and generated code across:
- Different developers
- CI/CD runs
- Local and remote builds

## References

- [toolchains_protoc repository](https://github.com/aspect-build/toolchains_protoc)
- [Bazel proto rules documentation](https://bazel.build/reference/be/protocol-buffer)
- [rules_go proto documentation](https://github.com/bazelbuild/rules_go/blob/master/docs/go/core/rules.md#proto)