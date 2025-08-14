load("@bazel_gazelle//:deps.bzl", "go_repository")

def go_dependencies():
    go_repository(
        name = "com_github_spf13_cobra",
        importpath = "github.com/spf13/cobra",
        sum = "h1:e5/vxKd/rZsfSJMUX1agtjeTDf+qv1/JdBF8gg5k9ZM=",
        version = "v1.9.1",
    )
    go_repository(
        name = "com_github_spiffe_go_spiffe_v2",
        importpath = "github.com/spiffe/go-spiffe/v2",
        sum = "h1:HUc3Xl4ATh9g6F5zMTKIIzBCLWv5Vge0jyOuwC9IWa4=",
        version = "v2.5.0",
    )
    go_repository(
        name = "com_github_stretchr_testify",
        importpath = "github.com/stretchr/testify",
        sum = "h1:Xv5erBjTwe/5IxqUQTdXv5kgmIvbHo3QQyRwhJsOfJA=",
        version = "v1.10.0",
    )
    go_repository(
        name = "org_golang_x_tools",
        importpath = "golang.org/x/tools",
        sum = "h1:+j4qImWuHwjAmISqrHJgoY8sIzdc7RLmEjW6xp92PbE=",
        version = "v0.36.0",
    )
    go_repository(
        name = "org_golang_google_grpc",
        importpath = "google.golang.org/grpc",
        sum = "h1:h2hn1RaG8cTHGu1dBz7k2d29GNUO+cFCiPZaHPwBmuc=",
        version = "v1.74.2",
    )
    go_repository(
        name = "org_golang_google_protobuf",
        importpath = "google.golang.org/protobuf",
        sum = "h1:xYDXYNhS97bVRhq5bUlwOu7X3b1OVUaRODVPsXUxKqg=",
        version = "v1.36.7",
    )
    go_repository(
        name = "in_gopkg_yaml_v3",
        importpath = "gopkg.in/yaml.v3",
        sum = "h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=",
        version = "v3.0.1",
    )
    
    # Indirect dependencies
    go_repository(
        name = "com_github_microsoft_go_winio",
        importpath = "github.com/Microsoft/go-winio",
        sum = "h1:lwRDcWSL3V5eKE9/8gJqr9YSGgzTIk1L4RZFd6UWBNs=",
        version = "v0.6.2",
    )
    go_repository(
        name = "com_github_davecgh_go_spew",
        importpath = "github.com/davecgh/go-spew",
        sum = "h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_go_jose_go_jose_v4",
        importpath = "github.com/go-jose/go-jose/v4",
        sum = "h1:X4YLjr0ufJF9kfXPBcJg4uPjHVMRHAGRcD6vPLqWf68=",
        version = "v4.1.2",
    )
    go_repository(
        name = "com_github_inconshreveable_mousetrap",
        importpath = "github.com/inconshreveable/mousetrap",
        sum = "h1:wN+x4NVGpMsO7ErUn/mUI3vEoE6Jt13X2s0bqwp9tc8=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_kr_text",
        importpath = "github.com/kr/text",
        sum = "h1:5Nx0Ya0ZqY2ygV366QzturHI13Jq95ApcVaJBhpS+AY=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_pmezard_go_difflib",
        importpath = "github.com/pmezard/go-difflib",
        sum = "h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_spf13_pflag",
        importpath = "github.com/spf13/pflag",
        sum = "h1:jg4VlVrcC5zI1yTwFZYKk3SiIZOyB3ksLyQ3pG21nBk=",
        version = "v1.0.7",
    )
    go_repository(
        name = "com_github_zeebo_errs",
        importpath = "github.com/zeebo/errs",
        sum = "h1:5J6C/NStFcIwAXrEHGjBxCqg7t8JPJR1IwUiZvFE8K0=",
        version = "v1.4.0",
    )
    go_repository(
        name = "org_golang_x_crypto",
        importpath = "golang.org/x/crypto",
        sum = "h1:gf2VzMQUJJ/ksP1yPVyGC9o3TgDhOQjy+k+TzpM5e6c=",
        version = "v0.41.0",
    )
    go_repository(
        name = "org_golang_x_mod",
        importpath = "golang.org/x/mod",
        sum = "h1:L8iymdnm/LQw5Z2Mep3QJUfuwCRZHD82V0F1F72Apfw=",
        version = "v0.27.0",
    )
    go_repository(
        name = "org_golang_x_net",
        importpath = "golang.org/x/net",
        sum = "h1:5x7E9ZqTb5cg4hNGAcjn7s8m8XU+TwNQpCTNtP8kz6o=",
        version = "v0.43.0",
    )
    go_repository(
        name = "org_golang_x_sync",
        importpath = "golang.org/x/sync",
        sum = "h1:Y8nWskOOhJV8+qj+RmcVmZPDbYfUHsQBGD/tWuEe2Ek=",
        version = "v0.16.0",
    )
    go_repository(
        name = "org_golang_x_sys",
        importpath = "golang.org/x/sys",
        sum = "h1:kL7nJQYXJ38VrzMXHQmNgBD1fP2nfRYhSq7mTCwxUX4=",
        version = "v0.35.0",
    )
    go_repository(
        name = "org_golang_x_text",
        importpath = "golang.org/x/text",
        sum = "h1:ybGApRDj5a4PdW0hLpbKGUXCkSR6wmjlOZEd21VKSks=",
        version = "v0.28.0",
    )
    go_repository(
        name = "org_golang_google_genproto_googleapis_rpc",
        importpath = "google.golang.org/genproto/googleapis/rpc",
        sum = "h1:AHkiHJEgDBnUJKqVrqxOuL7onZQfX7JKzWKCYJMTmBk=",
        version = "v0.0.0-20250804133106-a7a43d27e69b",
    )