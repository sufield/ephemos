# Ephemos Examples

This directory contains comprehensive examples and documentation for building secure, identity-based services with the Ephemos library.

## 📖 Documentation

| Guide | Purpose | Audience |
|-------|---------|----------|
| **[Getting Started](GETTING_STARTED.md)** | Step-by-step tutorial building your first secure service | New users |
| **[Architecture](ARCHITECTURE.md)** | Technical deep-dive into Ephemos internals and design | Architects, senior developers |
| **[Deployment](DEPLOYMENT.md)** | Production deployment patterns and best practices | DevOps, platform engineers |
| **[Troubleshooting](TROUBLESHOOTING.md)** | Common issues and solutions | All developers |

## 🚀 Quick Start

Choose your path:

- **📚 First time?** → [Getting Started Guide](GETTING_STARTED.md)
- **⚡ Want to see it work?** → Run `make demo` from project root
- **🏗️ Building a service?** → Copy templates from [`proto/`](proto/) directory
- **🔍 Need examples?** → See [`echo-server/`](echo-server/) and [`echo-client/`](echo-client/)

## 📁 Directory Structure

```
examples/
├── README.md              # 📋 This overview and navigation guide
├── GETTING_STARTED.md     # 🎓 Complete tutorial for beginners
├── ARCHITECTURE.md        # 🏗️ Technical architecture and design patterns
├── DEPLOYMENT.md          # 🚀 Production deployment guide
├── TROUBLESHOOTING.md     # 🔧 Common issues and solutions
├── proto/                 # 📄 Copy-paste templates for new services
│   ├── echo.proto         # Example protocol definition
│   ├── echo.pb.go         # Generated protobuf code
│   ├── echo_grpc.pb.go    # Generated gRPC interfaces
│   ├── client.go          # Generic client patterns
│   ├── registrar.go       # Service registration utilities
│   └── README.md          # Template usage instructions
├── echo-server/           # 🖥️ Complete server implementation
│   └── main.go
└── echo-client/           # 📱 Complete client implementation
    └── main.go
```

## What Ephemos Provides

Ephemos is an identity-based authentication library for Go services using SPIFFE/SPIRE:

- **Automatic mTLS**: All service communication secured with mutual TLS  
- **Identity-based auth**: Services authenticate using cryptographic identities, not passwords
- **Zero-config security**: Certificate management handled automatically
- **Service-agnostic**: Works with any gRPC service
- **Production-ready**: Comprehensive error handling, logging, and resource management

## Building the Project

Run `make build` to compile binaries to the `bin/` directory:

```bash
make build     # Builds CLI tool (ephemos) 
make examples  # Builds example applications (echo-server, echo-client)
make all       # Builds everything (proto, CLI, examples)
```

All build artifacts are stored in `bin/` and excluded from version control.

---

**Ready to get started?** Head to the **[Getting Started Guide](GETTING_STARTED.md)** for a complete tutorial, or explore the specific documentation guides above based on your needs.