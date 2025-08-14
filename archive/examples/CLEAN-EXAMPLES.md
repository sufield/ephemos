# Clean Ephemos Examples (No Protobuf)

These examples show how backend developers should use the Ephemos library - simple, clean, and without any protobuf requirements.

## 🎯 **The Pattern**

Both client and server developers use the same simple pattern:

```
Your Code ← Ephemos (identity layer) ← Network
```

## 🖥️ **Server Developer** (`http-server/`)

```go
// 1. Write normal HTTP handlers
func statusHandler(w http.ResponseWriter, r *http.Request) {
    // Client is already authenticated when this runs
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// 2. Create handlers as usual
mux := http.NewServeMux()
mux.HandleFunc("/api/status", statusHandler)

// 3. Add identity authentication with one line
server, _ := ephemos.NewIdentityServer(ctx, "")
server.ServeHTTP(ctx, ":8080", mux)  // Only authenticated clients can connect
```

**That's it!** Your HTTP handlers run unchanged, but only authenticated clients can reach them.

## 💻 **Client Developer** (`http-client/`)

```go
// 1. Create identity-aware client
client, _ := ephemos.NewIdentityClient(ctx, "")

// 2. Create HTTP client with automatic authentication
httpClient, _ := client.NewHTTPClient("service-name", "localhost:8080")

// 3. Use like normal HTTP client
resp, _ := httpClient.Get("http://localhost:8080/api/status")
```

**That's it!** Works like standard `http.Client` but with automatic SPIFFE authentication.

## ✅ **Benefits**

1. **No protoc needed** - Just import the library
2. **No .proto files** - Works with existing HTTP APIs  
3. **No gRPC knowledge** - Uses standard HTTP
4. **Existing code works** - Just wrap with Ephemos
5. **Like go-spiffe** - Transport-layer identity

## 🔄 **Client ↔ Server Flow**

```
[Client Code] → [Ephemos Client] → [Network with mTLS] → [Ephemos Server] → [Server Code]
                   ↓                                           ↓
              Presents SPIFFE cert                     Verifies SPIFFE cert
```

Both sides just use the library - all SPIFFE/SPIRE complexity is hidden.

## 🚫 **What You DON'T Need**

- ❌ protoc compiler
- ❌ .proto files  
- ❌ protoc-gen-go plugins
- ❌ gRPC knowledge
- ❌ SPIFFE/SPIRE expertise

## 📁 **Example Structure**

```
http-client/     # Backend dev making authenticated HTTP requests
http-server/     # Backend dev serving HTTP with authentication  
```

This is the **clean, simple pattern** for the 0.1 release.