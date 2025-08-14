workspace(name = "ephemos")

# Load Bazel rules for Go
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# Go rules
http_archive(
    name = "io_bazel_rules_go",
    integrity = "sha256-fHbWI2so/2laoozzX5XeMXqUcv0fsUrHl8m/aE8Js3w=",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.50.1/rules_go-v0.50.1.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.50.1/rules_go-v0.50.1.zip",
    ],
)

# Gazelle for automatic BUILD file generation
http_archive(
    name = "bazel_gazelle",
    integrity = "sha256-12v3pg/YsFBEQJDfooN6Tq+YKeEWVhjuNdzspcvfWNU=",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.37.0/bazel-gazelle-v0.37.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.37.0/bazel-gazelle-v0.37.0.tar.gz",
    ],
)

# Protocol Buffers
http_archive(
    name = "com_google_protobuf",
    sha256 = "4fc5ff1b2c339fb86cd3a25f0b5311478ab081e65ad258c6789359cd84d421f8",
    strip_prefix = "protobuf-26.1",
    urls = [
        "https://mirror.bazel.build/github.com/protocolbuffers/protobuf/archive/v26.1.tar.gz",
        "https://github.com/protocolbuffers/protobuf/archive/v26.1.tar.gz",
    ],
)

# gRPC - compatible with Go rules
http_archive(
    name = "com_github_grpc_grpc",
    sha256 = "72ce7d6bdaaf4cc82d1c7ab8a42bfeadf1e4a4b8f2fa83c3be1c4b26f8e86ced",
    strip_prefix = "grpc-1.64.0",
    urls = [
        "https://mirror.bazel.build/github.com/grpc/grpc/archive/v1.64.0.tar.gz",
        "https://github.com/grpc/grpc/archive/v1.64.0.tar.gz",
    ],
)

# Rules_java - Bazel 7.x compatible version
# Using version 7.6.5 which is compatible with Bazel 7.x and does not have _has_launcher_maker_toolchain
http_archive(
    name = "rules_java",
    sha256 = "4da3761f6855ad916568e2bfe86213ba6d2637f56b8360538a7fb6125abf6518",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_java/releases/download/7.6.5/rules_java-7.6.5.tar.gz",
        "https://github.com/bazelbuild/rules_java/releases/download/7.6.5/rules_java-7.6.5.tar.gz",
    ],
)

# Initialize Go rules
load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")
load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")
load("@rules_java//java:repositories.bzl", "rules_java_dependencies", "rules_java_toolchains")

go_rules_dependencies()
go_register_toolchains(version = "1.24")

gazelle_dependencies()
protobuf_deps()

rules_java_dependencies()
rules_java_toolchains()

# Load Go dependencies
load("//:deps.bzl", "go_dependencies")
go_dependencies()