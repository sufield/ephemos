load("@bazel_gazelle//:deps.bzl", "go_repository")

def go_dependencies():
    go_repository(
        name = "com_github_microsoft_go_winio",
        importpath = "github.com/Microsoft/go-winio",
        sum = "h1:F2VQgta7ecxGYO8k3ZZz3RS8fVIXVxONVUPlNERoyfY=",
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
        sum = "h1:TK/7NqRQZfgAh+Td8AlsrvtPoUyiHh0LqVvokh+1vHI=",
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
        name = "com_github_spf13_cobra",
        importpath = "github.com/spf13/cobra",
        sum = "h1:CXSaggrXdbHK9CF+8ywj8Amf7PBRmPCOJugH954Nnlo=",
        version = "v1.9.1",
    )
    go_repository(
        name = "com_github_spf13_pflag",
        importpath = "github.com/spf13/pflag",
        sum = "h1:vN6T9TfwStFPFM5XzjsvmzZkLuaLX+HS+0SeFLRgU6M=",
        version = "v1.0.7",
    )
    go_repository(
        name = "com_github_spiffe_go_spiffe_v2",
        importpath = "github.com/spiffe/go-spiffe/v2",
        sum = "h1:N2I01KCUkv1FAjZXJMwh95KK1ZIQLYbPfhaxw8WS0hE=",
        version = "v2.5.0",
    )
    go_repository(
        name = "com_github_stretchr_testify",
        importpath = "github.com/stretchr/testify",
        sum = "h1:Xv5erBjTwe/5IxqUQTdXv5kgmIvbHo3QQyRwhJsOfJA=",
        version = "v1.10.0",
    )
    go_repository(
        name = "com_github_zeebo_errs",
        importpath = "github.com/zeebo/errs",
        sum = "h1:XNdoD/RRMKP7HD0UhJnIzUy74ISdGGxURlYG8HSWSfM=",
        version = "v1.4.0",
    )
    go_repository(
        name = "org_golang_x_crypto",
        importpath = "golang.org/x/crypto",
        sum = "h1:WKYxWedPGCTVVl5+WHSSrOBT0O8lx32+zxmHxijgXp4=",
        version = "v0.41.0",
    )
    go_repository(
        name = "org_golang_x_mod",
        importpath = "golang.org/x/mod",
        sum = "h1:kb+q2PyFnEADO2IEF935ehFUXlWiNjJWtRNgBLSfbxQ=",
        version = "v0.27.0",
    )
    go_repository(
        name = "org_golang_x_net",
        importpath = "golang.org/x/net",
        sum = "h1:lat02VYK2j4aLzMzecihNvTlJNQUq316m2Mr9rnM6YE=",
        version = "v0.43.0",
    )
    go_repository(
        name = "org_golang_x_sync",
        importpath = "golang.org/x/sync",
        sum = "h1:ycBJEhp9p4vXvUZNszeOq0kGTPghopOL8q0fq3vstxw=",
        version = "v0.16.0",
    )
    go_repository(
        name = "org_golang_x_sys",
        importpath = "golang.org/x/sys",
        sum = "h1:vz1N37gP5bs89s7He8XuIYXpyY0+QlsKmzipCbUtyxI=",
        version = "v0.35.0",
    )
    go_repository(
        name = "org_golang_x_text",
        importpath = "golang.org/x/text",
        sum = "h1:rhazDwis8INMIwQ4tpjLDzUhx6RlXqZNPEM0huQojng=",
        version = "v0.28.0",
    )
    go_repository(
        name = "org_golang_x_tools",
        importpath = "golang.org/x/tools",
        sum = "h1:kWS0uv/zsvHEle1LbV5LE8QujrxB3wfQyxHfhOk0Qkg=",
        version = "v0.36.0",
    )
    go_repository(
        name = "org_golang_google_genproto_googleapis_rpc",
        importpath = "google.golang.org/genproto/googleapis/rpc",
        sum = "h1:zPKJod4w6F1+nRGDI9ubnXYhU9NSWoFAijkHkUXeTK8=",
        version = "v0.0.0-20250804133106-a7a43d27e69b",
    )
    go_repository(
        name = "org_golang_google_grpc",
        importpath = "google.golang.org/grpc",
        sum = "h1:WoosgB65DlWVC9FqI82dGsZhWFNBSLjQ84bjROOpMu4=",
        version = "v1.74.2",
    )
    go_repository(
        name = "org_golang_google_protobuf",
        importpath = "google.golang.org/protobuf",
        sum = "h1:IgrO7UwFQGJdRNXH/sQux4R1Dj1WAKcLElzeeRaXV2A=",
        version = "v1.36.7",
    )
    go_repository(
        name = "in_gopkg_yaml_v3",
        importpath = "gopkg.in/yaml.v3",
        sum = "h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=",
        version = "v3.0.1",
    )
