workspace(name = "ephemos")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# C++ rules (required by Bazel toolchain) - Using stable version compatible with Bazel 7.x
http_archive(
    name = "rules_cc",
    urls = ["https://github.com/bazelbuild/rules_cc/releases/download/0.0.16/rules_cc-0.0.16.tar.gz"],
    sha256 = "bbf1ae2f83305b7053b11e4467d317a7ba3517a12cef608543c1b1c5bf48a4df",
    strip_prefix = "rules_cc-0.0.16",
)

# Bazel Go rules
http_archive(
    name = "io_bazel_rules_go",
    sha256 = "33acc4ae0f70502db4b893c9fc1dd7a9bf998c23e7ff2c4517741d4049a976f8",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.48.0/rules_go-v0.48.0.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.48.0/rules_go-v0.48.0.zip",
    ],
)

# Gazelle for BUILD file generation  
http_archive(
    name = "bazel_gazelle",
    sha256 = "d76bf7a60fd8b050444090dfa2837a4eaf9829e1165618ee35dceca5cbdf58d5",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.37.0/bazel-gazelle-v0.37.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.37.0/bazel-gazelle-v0.37.0.tar.gz",
    ],
)

# Bazel skylib - Required by rules_java
http_archive(
    name = "bazel_skylib",
    sha256 = "bc283cdfcd526a52c3201279cda4bc298652efa898b10b4db0837dc51652756f",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.7.1/bazel-skylib-1.7.1.tar.gz",
        "https://github.com/bazelbuild/bazel-skylib/releases/download/1.7.1/bazel-skylib-1.7.1.tar.gz",
    ],
)

# Bazel features compatibility layer - Updated for latest compatibility
http_archive(
    name = "bazel_features",
    sha256 = "ba1282c1aa1d1fffdcf994ab32131d7c7551a9bc960fbf05f42d55a1b930cbfb",
    strip_prefix = "bazel_features-1.15.0",
    urls = [
        "https://github.com/bazel-contrib/bazel_features/releases/download/v1.15.0/bazel_features-v1.15.0.tar.gz",
    ],
)

# Note: Prebuilt protoc toolchain temporarily disabled due to version compatibility issues
# http_archive(
#     name = "toolchains_protoc",
#     sha256 = "3019f9ed1273d547334da2004e634340c896d9e24dd6d899911e03b694fdc1f5",
#     strip_prefix = "toolchains_protoc-0.4.3", 
#     urls = [
#         "https://github.com/aspect-build/toolchains_protoc/releases/download/v0.4.3/toolchains_protoc-v0.4.3.tar.gz",
#     ],
# )

# Protocol buffers rules - Using compatible version for Bazel 7.x
http_archive(
    name = "rules_proto",
    sha256 = "6fb6767d1bef535310547e03247f7518b03487740c11b6c6adb7952033fe1295",
    strip_prefix = "rules_proto-6.0.2",
    url = "https://github.com/bazelbuild/rules_proto/releases/download/6.0.2/rules_proto-6.0.2.tar.gz",
)

# Note: Protocol buffers now managed by rules_proto 7.1.0
# http_archive(
#     name = "com_google_protobuf",
#     strip_prefix = "protobuf-27.3",
#     urls = [
#         "https://github.com/protocolbuffers/protobuf/archive/v27.3.tar.gz",
#     ],
#     sha256 = "1535151efbc7893f38b0578e83cac584f2819974f065698976989ec71c1af84a",
# )

# Note: Java rules commented out - project uses Go, not Java  
# If needed by transitive dependencies, uncomment with compatible version:
# http_archive(
#     name = "rules_java", 
#     urls = ["https://github.com/bazelbuild/rules_java/releases/download/7.12.5/rules_java-7.12.5.tar.gz"],
#     sha256 = "17b18cb4f92ab7b94aa343ce78531b73960b1bed2ba166e5b02c9fdf0b0ac270",
# )

# Python rules - Updated to latest stable version compatible with Bazel 7.x
# This version resolves PyCcLinkParamsProvider issues and provides full Bazel 7 support
http_archive(
    name = "rules_python",
    sha256 = "0a1cefefb4a7b550fb0b43f54df67d6da95b7ba352637669e46c987f69986f6a",
    strip_prefix = "rules_python-1.5.3",
    urls = [
        "https://github.com/bazel-contrib/rules_python/releases/download/1.5.3/rules_python-1.5.3.tar.gz",
    ],
)

load("@rules_cc//cc:repositories.bzl", "rules_cc_dependencies", "rules_cc_toolchains")
load("@bazel_features//:deps.bzl", "bazel_features_deps")
load("@rules_proto//proto:repositories.bzl", "rules_proto_dependencies")
load("@rules_python//python:repositories.bzl", "py_repositories", "python_register_toolchains")
# Note: rules_java load commented out - project uses Go, not Java
load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

rules_cc_dependencies()

rules_cc_toolchains()

bazel_features_deps()

rules_proto_dependencies()

# Note: rules_java initialization commented out - project uses Go, not Java

py_repositories()

python_register_toolchains(
    name = "python3_11",
    python_version = "3.11",
)

go_rules_dependencies()

go_register_toolchains(version = "1.24.5")

gazelle_dependencies()

# Note: Prebuilt protoc toolchains disabled due to compatibility issues
# load("@toolchains_protoc//protoc:toolchain.bzl", "protoc_toolchains")
# 
# protoc_toolchains(
#     name = "protoc_toolchains",
#     version = "27.0",  # Use a stable protoc version supported by toolchains_protoc
# )

# Note: protobuf_deps no longer needed - managed by rules_proto 7.1.0
# load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")
# protobuf_deps()

# Go dependencies
load("//:deps.bzl", "go_dependencies")

# gazelle:repository_macro deps.bzl%go_dependencies
go_dependencies()

# Register custom Go proto toolchains for optimized builds
# Note: Custom toolchains temporarily disabled due to compatibility issues
# register_toolchains(
#     "//tools/toolchains:go_proto_toolchain",
#     "//tools/toolchains:go_grpc_toolchain", 
#     "//tools/toolchains:go_combined_toolchain",
# )