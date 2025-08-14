# Simple HTTP Examples - Lightweight & Clean

Ephemos is a lightweight abstraction over [go-spiffe HTTP](https://github.com/spiffe/go-spiffe/tree/main/examples/spiffe-http).

**What it does**: HTTP over mTLS using X.509 SVIDs

**What it removes**: All the go-spiffe complexity

## 🔧 **Server** (`http-server-simple/`)

```go
// Your normal HTTP handlers
func statusHandler(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Add SPIFFE mTLS - one line
mux := http.NewServeMux()
mux.HandleFunc("/status", statusHandler)
server, _ := ephemos.NewHTTPServer(ctx, ":8080", mux)
server.ListenAndServe()
```

## 💻 **Client** (`http-client-simple/`)

```go
// HTTP client with automatic SPIFFE mTLS
client, _ := ephemos.NewHTTPClient(ctx, "localhost:8080") 
resp, _ := client.Get("http://localhost:8080/status")
```

## ✅ **That's It!**

- **No gRPC** - Pure HTTP
- **No protobuf** - Standard JSON
- **No complexity** - 10 lines of code
- **Automatic mTLS** - X.509 SVIDs handled transparently

## 🎯 **Advanced Example**

See `time-server/` and `time-client/` for a complete timezone service example.

## 📦 **Dependencies**

```go
import "github.com/sufield/ephemos/pkg/ephemos"
```

Just one import. Ephemos handles all the go-spiffe complexity internally.