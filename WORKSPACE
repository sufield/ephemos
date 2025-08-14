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

# Bazel skylib - Core Bazel utilities
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
load("@rules_python//python:repositories.bzl", "py_repositories", "python_register_toolchains")
load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

rules_cc_dependencies()

rules_cc_toolchains()

bazel_features_deps()

# Note: Library uses HTTP over mTLS for service communication

py_repositories()

python_register_toolchains(
    name = "python3_11",
    python_version = "3.11",
)

go_rules_dependencies()

go_register_toolchains(version = "1.24.5")

gazelle_dependencies()


# Go dependencies
load("//:deps.bzl", "go_dependencies")

# gazelle:repository_macro deps.bzl%go_dependencies
go_dependencies()

