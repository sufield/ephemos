# Ephemos

A lightweight HTTP over mTLS library built on the SPIFFE/SPIRE identity framework.

## Overview

Ephemos provides secure HTTP communication using mutual TLS (mTLS) authentication with SPIFFE identities. It simplifies the implementation of zero-trust networking patterns by abstracting the complexity of X.509 SVID-based authentication.

## Features

- ✅ **SPIFFE/SPIRE Integration**: Native support for SPIFFE identities and X.509 SVIDs
- ✅ **mTLS Transport**: Automatic mutual TLS with identity-based authentication  
- ✅ **HTTP over mTLS**: Standard HTTP semantics over secure mTLS connections
- ✅ **Configuration Validation**: Built-in validation for production configurations
- ✅ **Go-first Design**: Idiomatic Go library with minimal dependencies

## Quick Start

```go
import "github.com/sufield/ephemos/pkg/ephemos"

// Configure Ephemos client
config := &ephemos.Config{
    TrustDomain: "prod.company.com",
    ServiceName: "web-service",
}

client, err := ephemos.NewClient(config)
if err != nil {
    log.Fatal(err)
}

// Make secure HTTP requests
resp, err := client.Get("https://api.internal.com/data")
```

## Documentation

- [API Documentation](docs/api/) - Generated Go documentation
- [Configuration Guide](docs/configuration.md) - Configuration options and examples
- [Security Model](docs/security.md) - SPIFFE/SPIRE integration details

## Building

```bash
# Build the library
make build

# Run tests
make test

# Build CLI tools
go build -o bin/ephemos ./cmd/ephemos-cli
go build -o bin/config-validator ./cmd/config-validator
```

## Requirements

- Go 1.24+
- SPIRE server deployment
- Valid SPIFFE trust domain configuration

## Security

Ephemos follows security best practices:

- All communications use mTLS with SPIFFE identity verification
- No secrets or credentials are logged or exposed
- Regular security scanning and dependency updates
- Follows SPIFFE specification for identity management

Report security issues to: security@sufield.com

## License

See [LICENSE](LICENSE) file for details.

## Contributing

Contributions welcome! See [Contributing Guidelines](.github/CONTRIBUTING.md) for details.