# Ephemos MVP: Single Authentication Method Focus

## The 4 SPIFFE Authentication Methods

SPIFFE/SPIRE supports 4 authentication patterns:

1. **SPIFFE to SPIFFE authentication using X.509 SVIDs**
2. **SPIFFE to SPIFFE authentication using JWT SVIDs**  
3. **HTTP over mTLS using X.509 SVIDs**
4. **gRPC over mTLS using X.509 SVIDs**

## MVP Decision: HTTP over mTLS using X.509 SVIDs

For the MVP release, we are focusing **exclusively** on:

### ✅ **Option 3: HTTP over mTLS using X.509 SVIDs**

**End-to-end flow**: Chi/Gin HTTP services ↔ Ephemos ↔ SPIFFE/SPIRE

### Why This Choice for MVP:

#### 🎯 **Maximum Developer Impact**
- **HTTP/REST dominance**: Most Go services are HTTP-based (Chi, Gin, net/http)
- **Immediate adoption**: Developers can replace API keys in existing HTTP services
- **Familiar patterns**: HTTP middleware is well-understood in Go ecosystem

#### 🚀 **Fastest Time-to-Value**
- **Drop-in replacement**: Replace `r.Use(auth.APIKeyMiddleware)` with `r.Use(chimiddleware.SPIFFEAuth)`
- **Existing infrastructure**: Most teams already have HTTP load balancers, monitoring
- **No protocol migration**: Teams don't need to migrate from HTTP to gRPC

#### 💼 **Enterprise Reality**  
- **Legacy compatibility**: Existing HTTP APIs need authentication without rewriting
- **Gradual adoption**: Teams can add SPIFFE auth to existing HTTP services incrementally
- **Multi-language support**: HTTP works with any language, gRPC requires more tooling

#### 🔧 **Technical Simplicity**
- **Standard TLS**: Uses familiar HTTP/TLS patterns with X.509 certificates
- **Framework integration**: Chi and Gin have established middleware patterns
- **Debugging**: HTTP + TLS is easier to debug than gRPC for most developers

### MVP Architecture

```
┌─────────────────┐    HTTP/mTLS     ┌─────────────────┐
│   Chi/Gin App   │ ────────────── │   Chi/Gin App   │
│                 │   (X.509 SVIDs) │                 │
│ + SPIFFEAuth    │                  │ + SPIFFEAuth    │
│   middleware    │                  │   middleware    │
└─────────────────┘                  └─────────────────┘
         │                                     │
         │                                     │
         └─────────────┬─────────────────┬─────┘
                       │                 │
                ┌─────────────────┐ ┌─────────────────┐
                │ Ephemos Core    │ │   SPIRE Agent   │
                │ - X.509 certs   │ │ - Certificate   │
                │ - Trust bundles │ │   rotation      │
                │ - HTTP helpers  │ │ - Identity      │
                └─────────────────┘ └─────────────────┘
```

### What's Included in MVP:

#### ✅ **Core Components**
- **X.509 SVID management**: Certificate fetching and rotation
- **Trust bundle handling**: Peer certificate validation
- **HTTP/TLS configuration**: Building `http.Transport` with mTLS
- **SPIRE integration**: Agent socket communication

#### ✅ **Contrib Middleware**  
- **Chi middleware**: `chimiddleware.SPIFFEAuth(config)`
- **Gin middleware**: `ginmiddleware.SPIFFEAuth(config)`
- **Identity extraction**: Access to peer SPIFFE ID in handlers
- **Authorization policies**: Allow/deny based on SPIFFE ID patterns

#### ✅ **End-to-End Example**
```go
// Service A (Chi)
r := chi.NewRouter()
r.Use(chimiddleware.SPIFFEAuth(ephemos.AuthConfig{
    AllowedServices: []string{"spiffe://prod.company.com/service-b/*"},
}))
r.Get("/api/data", func(w http.ResponseWriter, r *http.Request) {
    // Access authenticated peer identity
    identity := chimiddleware.GetSPIFFEIdentity(r.Context())
    log.Printf("Request from: %s", identity.SPIFFEID)
    json.NewEncoder(w).Encode(map[string]string{"data": "secret"})
})

// Service B (Gin) - calling Service A
r := gin.Default()
r.Use(ginmiddleware.SPIFFEAuth(authConfig))
r.GET("/proxy", func(c *gin.Context) {
    // Use Ephemos HTTP client with automatic mTLS
    client := ephemos.HTTPClient(config)
    resp, _ := client.Get("https://service-a.prod.company.com/api/data")
    // Forward response...
})
```

### What's Deferred (Post-MVP):

#### ⏳ **gRPC Support**
- **Rationale**: Requires protobuf generation, gRPC expertise
- **Timeline**: v2.0 after HTTP patterns are proven

#### ⏳ **JWT SVIDs**
- **Rationale**: X.509 certificates are more common and secure for mTLS
- **Timeline**: v3.0 for specific use cases requiring JWT

#### ⏳ **Generic SPIFFE-to-SPIFFE**
- **Rationale**: Too abstract - developers need concrete transport (HTTP/gRPC)
- **Timeline**: Framework-agnostic after transport-specific patterns mature

### Success Criteria for MVP:

1. **Drop-in replacement**: Replace API key middleware with SPIFFE middleware in < 10 lines
2. **Zero config complexity**: Works with default SPIRE setup
3. **Framework parity**: Chi and Gin have equivalent functionality  
4. **Production ready**: Handles certificate rotation, connection pooling, error cases
5. **Documentation complete**: Migration guide from API keys to SPIFFE

### MVP Scope Boundaries:

#### ✅ **In Scope**
- HTTP services with X.509 SVID authentication
- Chi and Gin middleware implementations
- Client-to-service HTTP calls with mTLS
- Certificate rotation and trust bundle management
- Basic authorization policies (allow/deny by SPIFFE ID)

#### ❌ **Out of Scope**  
- gRPC transport layer
- JWT SVID support
- WebSocket authentication
- Non-HTTP protocols
- Advanced authorization policies (RBAC, ABAC)
- Multi-cluster trust domains

### Why This Focused Approach Works:

1. **Proven pattern**: HTTP + mTLS is well-established in production
2. **Incremental adoption**: Teams can migrate one service at a time
3. **Lower risk**: HTTP is more familiar than gRPC for most teams
4. **Faster feedback**: Shorter development cycle to validate approach
5. **Foundation building**: HTTP patterns inform future gRPC design

**Bottom Line**: MVP delivers immediate value by solving the API key problem for HTTP services, which represent 80%+ of Go microservices in production. Once HTTP patterns are proven, we expand to gRPC and other transports.