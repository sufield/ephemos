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

# Bazel features compatibility layer
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

# Protocol buffers rules - Updated for Bazel 7.x compatibility
http_archive(
    name = "rules_proto",
    sha256 = "14a225870ab4e91869652cfd69ef2028277fc1dc4910d65d353b62d6e0ae21f4",
    strip_prefix = "rules_proto-7.1.0",
    url = "https://github.com/bazelbuild/rules_proto/releases/download/7.1.0/rules_proto-7.1.0.tar.gz",
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

# Note: Java rules removed as project uses Go, not Java
# If needed by transitive dependencies, uncomment with latest version:
# http_archive(
#     name = "rules_java", 
#     urls = ["https://github.com/bazelbuild/rules_java/releases/download/8.15.1/rules_java-8.15.1.tar.gz"],
#     sha256 = "9b04cbbb0fee0632aeba628159938484cfadf4a9d2f5b1c356e8300c56467896",
# )

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
# Note: rules_java dependencies removed as project uses Go, not Java
load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

rules_cc_dependencies()

rules_cc_toolchains()

bazel_features_deps()

rules_proto_dependencies()

# Note: rules_java initialization removed as project uses Go, not Java

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

# Note: Custom Go proto toolchains disabled due to compatibility issues
# register_toolchains("//tools/toolchains:all")