module github.com/sufield/ephemos/contrib/middleware/chi

go 1.24

require (
	github.com/go-chi/chi/v5 v5.2.2
	github.com/spiffe/go-spiffe/v2 v2.5.0
	github.com/stretchr/testify v1.10.0
	github.com/sufield/ephemos v0.0.0-00010101000000-000000000000
)

replace github.com/sufield/ephemos => ../../..

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-jose/go-jose/v4 v4.1.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/zeebo/errs v1.4.0 // indirect
	golang.org/x/crypto v0.41.0 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241202173237-19429a94021a // indirect
	google.golang.org/grpc v1.70.0 // indirect
	google.golang.org/protobuf v1.36.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
