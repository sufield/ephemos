# Protobuf-Free Examples

These examples demonstrate the **correct** way to use the Ephemos library - without requiring protoc, proto files, or any protobuf tooling.

## ✅ Correct Approach (No Protobuf Needed)

The Ephemos library is designed to work as a **transport layer** that adds identity-based authentication to your existing services:

```
Your Service ← Ephemos (identity layer) ← Network (gRPC/HTTP/TCP)
```

### Benefits:
- ✅ **No protoc compiler needed**
- ✅ **No .proto files needed** 
- ✅ **Works with existing HTTP/gRPC services**
- ✅ **Just add authentication to existing code**

## Examples

### 1. Simple HTTP Client (`simple-client/`)
Shows how to use Ephemos with standard HTTP clients:

```go
// Create identity-aware client
client, err := ephemos.NewIdentityClient(ctx, "")

// Connect with automatic SPIFFE authentication
conn, err := client.Connect(ctx, "my-service", "localhost:8080")

// Use with standard HTTP client - no protobuf!
httpClient := &http.Client{...}
resp, err := httpClient.Get("http://localhost:8080/api/status")
```

### 2. Simple HTTP Server (`simple-server/`)
Shows how to add identity authentication to existing HTTP services:

```go
// Your existing HTTP handlers - no changes needed!
mux := http.NewServeMux()
mux.HandleFunc("/api/status", statusHandler)

// Add identity authentication with Ephemos
server, err := ephemos.NewIdentityServer(ctx, "")
// Ephemos handles all SPIFFE/SPIRE complexity
```

## ❌ Problematic Examples (Uses Protobuf)

The following examples in this archive directory are **outdated** and require protobuf:

- `echo-client/` - Uses custom proto package ❌
- `echo-server/` - Uses custom proto package ❌  
- `proto/` - Contains protobuf definitions ❌

**These examples are incorrect** for a library that should work like go-spiffe.

## How This Should Work (Like go-spiffe)

When developers use go-spiffe, they don't need protoc:

```go
// go-spiffe example - no protoc needed
source, err := workloadapi.NewX509Source(ctx)
client := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: tlsconfig.MTLSClientConfig(source, source, tlsconfig.AuthorizeAny()),
    },
}
```

Similarly, Ephemos should work the same way:

```go  
// Ephemos should work like this - no protoc needed
client, err := ephemos.NewIdentityClient(ctx, "")
conn, err := client.Connect(ctx, "service", "addr")
// Use conn with any transport (HTTP, gRPC, TCP)
```

## Recommended Migration

1. **Remove protobuf examples** (`echo-client/`, `echo-server/`, `proto/`)
2. **Use simple examples** that show transport-layer identity
3. **Document as library** not as protobuf service framework

This aligns with the 0.1 release goal of providing a clean, simple identity library.