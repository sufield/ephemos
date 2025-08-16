# Ephemos MVP: Single Authentication Method Focus

## The 4 SPIFFE Authentication Methods

SPIFFE/SPIRE supports 4 authentication patterns based on 2 protocols × 2 SVID types:

| **Protocol** | **X.509 SVIDs** | **JWT SVIDs** |
|--------------|-----------------|---------------|
| **HTTP** | ✅ **Option 3**: HTTP over mTLS using X.509 SVIDs | ⚡ **Option 1**: HTTP with JWT SVID in headers |
| **gRPC** | 🚀 **Option 4**: gRPC over mTLS using X.509 SVIDs | 🔐 **Option 2**: gRPC with JWT SVID in metadata |

### Authentication Method Breakdown:

1. **HTTP + JWT SVIDs**: SPIFFE to SPIFFE authentication using JWT SVIDs (HTTP transport)
2. **gRPC + JWT SVIDs**: SPIFFE to SPIFFE authentication using JWT SVIDs (gRPC transport)  
3. **HTTP + X.509 SVIDs**: HTTP over mTLS using X.509 SVIDs
4. **gRPC + X.509 SVIDs**: gRPC over mTLS using X.509 SVIDs

## Framework Capabilities Matrix

| **Framework/Tool** | **HTTP + X.509** | **HTTP + JWT** | **gRPC + X.509** | **gRPC + JWT** | **Best For** |
|-------------------|-------------------|-----------------|-------------------|-----------------|--------------|
| **Ephemos Core** | 🎯 **MVP Focus** | ⏳ Future | ⏳ Future | ⏳ Future | Production HTTP services |
| **Chi Middleware** | ✅ Full Support | ⏳ v2.0 | ❌ N/A | ❌ N/A | REST APIs, web services |
| **Gin Middleware** | ✅ Full Support | ⏳ v2.0 | ❌ N/A | ❌ N/A | REST APIs, JSON APIs |
| **gRPC Interceptors** | ⏳ v2.0 | ⏳ v3.0 | ⏳ v2.0 | ⏳ v3.0 | High-performance RPC |
| **go-spiffe SDK** | ✅ Native | ✅ Native | ✅ Native | ✅ Native | Direct SPIFFE integration |
| **SPIRE Agent** | ✅ Supported | ✅ Supported | ✅ Supported | ✅ Supported | Certificate/token issuance |

## Use Case Recommendation Matrix

| **Scenario** | **Recommended Option** | **Why** | **Ephemos Support** |
|--------------|------------------------|---------|-------------------|
| **Microservices with REST APIs** | HTTP + X.509 SVIDs | Mature mTLS, works with load balancers | ✅ **MVP** |
| **High-throughput service mesh** | gRPC + X.509 SVIDs | Best performance, native K8s support | ⏳ v2.0 |
| **Legacy systems integration** | HTTP + JWT SVIDs | No TLS changes needed, header-based auth | ⏳ v2.0 |
| **Multi-language environments** | HTTP + JWT SVIDs | Language-agnostic JWT validation | ⏳ v2.0 |
| **Serverless/FaaS** | HTTP + JWT SVIDs | Stateless, no persistent connections | ⏳ v2.0 |
| **Edge/IoT devices** | gRPC + JWT SVIDs | Lightweight tokens, efficient serialization | ⏳ v3.0 |
| **Browser-to-service** | HTTP + JWT SVIDs | CORS-friendly, no client certificates | ⏳ v2.0 |
| **Service-to-service (secure)** | HTTP/gRPC + X.509 SVIDs | Strongest security, automatic rotation | ✅ **MVP** (HTTP) |

## Security & Performance Comparison

| **Aspect** | **X.509 SVIDs** | **JWT SVIDs** |
|------------|-----------------|---------------|
| **Security** | 🔒 **Highest** - Private key never leaves workload | 🔐 Medium - Token can be intercepted/replayed |
| **Performance** | ⚡ Fast - TLS handshake caching | 🐌 Slower - Signature verification per request |
| **Network** | 📡 Efficient - Connection reuse | 📦 Overhead - Token in every request |
| **Debugging** | 🔍 Standard TLS tools (Wireshark, openssl) | 📋 JWT tools (jwt.io, debuggers) |
| **Rotation** | 🔄 Transparent - Background certificate renewal | 🕐 Visible - New token per request |
| **Firewall** | 🛡️ Standard HTTPS/443 ports | 🛡️ Standard HTTPS/443 ports |
| **Load Balancers** | ✅ Full support - Standard TLS termination | ⚠️ Limited - May need JWT passthrough |
| **Caching** | ✅ Connection-level caching | ❌ Token validation per request |

## Framework Suitability

### **HTTP + X.509 SVIDs** (MVP Choice)
- **Best for**: Production microservices, REST APIs, existing HTTP infrastructure
- **Frameworks**: Chi, Gin, Echo, Fiber, net/http
- **Pros**: Mature tooling, connection reuse, familiar TLS patterns
- **Cons**: Requires TLS configuration, client certificate management

### **HTTP + JWT SVIDs** 
- **Best for**: Legacy integration, multi-language environments, serverless
- **Frameworks**: Any HTTP framework with middleware support
- **Pros**: Language-agnostic, no TLS changes, stateless
- **Cons**: Performance overhead, token management, replay attacks

### **gRPC + X.509 SVIDs**
- **Best for**: High-performance service mesh, Kubernetes-native services
- **Frameworks**: gRPC-Go, gRPC interceptors, service mesh (Istio/Linkerd)
- **Pros**: Best performance, native K8s support, efficient serialization
- **Cons**: gRPC adoption required, protobuf complexity

### **gRPC + JWT SVIDs**
- **Best for**: Edge devices, constrained environments, token-based workflows  
- **Frameworks**: gRPC with custom auth, IoT platforms
- **Pros**: Lightweight, stateless, works with limited TLS
- **Cons**: Performance overhead, limited ecosystem support

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
- **Drop-in replacement**: Replace `r.Use(auth.APIKeyMiddleware)` with `r.Use(ephemos.IdentityMiddleware)`
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
│   Chi/Gin App   │ ──────────────   │   Chi/Gin App   │
│                 │   (X.509 SVIDs)  │                 │
│ + EphemosAuth   │                  │ + EphemosAuth   │
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
- **Chi middleware**: `chimiddleware.IdentityAuthentication(settings)`
- **Gin middleware**: `ginmiddleware.IdentityAuthentication(settings)`
- **Identity extraction**: Access to identity document in handlers via `GetIdentityDocument()`
- **Authorization policies**: Allow/deny based on identity document validation

#### ✅ **End-to-End Example**
```go
// Service A (Chi)
r := chi.NewRouter()
r.Use(chimiddleware.IdentityAuthentication(ephemos.IdentitySetting{
    AllowedServices: []string{"spiffe://prod.company.com/service-b/*"},
}))
r.Get("/api/data", func(w http.ResponseWriter, r *http.Request) {
    // Access identity document from authenticated request
    identityDoc := chimiddleware.GetIdentityDocument(r.Context())
    log.Printf("Request from: %s", identityDoc.ServiceName())
    json.NewEncoder(w).Encode(map[string]string{"data": "secret"})
})

// Service B (Gin) - calling Service A
r := gin.Default()
r.Use(ginmiddleware.IdentityAuthentication(ephemos.IdentitySetting{
    AllowedServices: []string{"spiffe://prod.company.com/service-a/*"},
}))
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

1. **Drop-in replacement**: Replace API key middleware with identity authentication in < 10 lines
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
- Basic authorization policies (allow/deny by identity document validation)

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