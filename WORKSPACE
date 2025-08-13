workspace(name = "ephemos")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# C++ rules (required by Bazel toolchain) - Updated for Bazel 7.x compatibility
http_archive(
    name = "rules_cc",
    urls = ["https://github.com/bazelbuild/rules_cc/releases/download/0.0.10/rules_cc-0.0.10.tar.gz"],
    sha256 = "65b67b81c6da378f136cc7e7e14ee08d5b9375973427eceb8c773a4f69fa7e49",
    strip_prefix = "rules_cc-0.0.10",
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

# Bazel features compatibility layer
http_archive(
    name = "bazel_features",
    sha256 = "ba1282c1aa1d1fffdcf994ab32131d7c7551a9bc960fbf05f42d55a1b930cbfb",
    strip_prefix = "bazel_features-1.15.0",
    urls = [
        "https://github.com/bazel-contrib/bazel_features/releases/download/v1.15.0/bazel_features-v1.15.0.tar.gz",
    ],
)

# Prebuilt protoc toolchain for faster builds
http_archive(
    name = "toolchains_protoc",
    sha256 = "3019f9ed1273d547334da2004e634340c896d9e24dd6d899911e03b694fdc1f5",
    strip_prefix = "toolchains_protoc-0.4.3", 
    urls = [
        "https://github.com/aspect-build/toolchains_protoc/releases/download/v0.4.3/toolchains_protoc-v0.4.3.tar.gz",
    ],
)

# Protocol buffers rules
http_archive(
    name = "rules_proto",
    sha256 = "6fb6767d1bef535310547e03247f7518b03487740c11b6c6adb7952033fe1295",
    strip_prefix = "rules_proto-6.0.2",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_proto/releases/download/6.0.2/rules_proto-6.0.2.tar.gz",
        "https://github.com/bazelbuild/rules_proto/releases/download/6.0.2/rules_proto-6.0.2.tar.gz",
    ],
)

# Protocol buffers
http_archive(
    name = "com_google_protobuf",
    strip_prefix = "protobuf-27.3",
    urls = [
        "https://github.com/protocolbuffers/protobuf/archive/v27.3.tar.gz",
    ],
    sha256 = "1535151efbc7893f38b0578e83cac584f2819974f065698976989ec71c1af84a",
)

# Python rules
http_archive(
    name = "rules_python",
    sha256 = "778aaeab3e6cfd56d681c89f5c10d7ad6bf8d2f1a72de9de55b23081b2d31618",
    strip_prefix = "rules_python-0.34.0",
    urls = [
        "https://github.com/bazelbuild/rules_python/releases/download/0.34.0/rules_python-0.34.0.tar.gz",
    ],
)

load("@rules_cc//cc:repositories.bzl", "rules_cc_dependencies", "rules_cc_toolchains")
load("@bazel_features//:deps.bzl", "bazel_features_deps")
load("@rules_proto//proto:repositories.bzl", "rules_proto_dependencies")
load("@rules_python//python:repositories.bzl", "py_repositories", "python_register_toolchains")
load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

rules_cc_dependencies()

rules_cc_toolchains()

bazel_features_deps()

rules_proto_dependencies()

py_repositories()

python_register_toolchains(
    name = "python3_11",
    python_version = "3.11",
)

go_rules_dependencies()

go_register_toolchains(version = "1.24.5")

gazelle_dependencies()

# Register prebuilt protoc toolchains
load("@toolchains_protoc//protoc:toolchain.bzl", "protoc_toolchains")

protoc_toolchains(
    name = "protoc_toolchains",
    version = "28.3",  # Use a stable protoc version
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

# Go dependencies
load("//:deps.bzl", "go_dependencies")

# gazelle:repository_macro deps.bzl%go_dependencies
go_dependencies()

# Register custom Go proto toolchains
register_toolchains("//tools/toolchains:all")