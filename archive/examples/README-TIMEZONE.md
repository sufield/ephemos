# Timezone Service Example - Clean Ephemos Usage

A simple, relatable example showing how backend developers use Ephemos library for identity-based authentication.

## 🕐 **The Service**

**Business Logic**: Client sends timezone string, server responds with current time in that timezone

**Security**: Only authenticated clients (with SPIFFE certificates) can access the service

## 🖥️ **Time Server** (`time-server/`)

A backend developer creating a time service:

```go
// Normal HTTP handler - your business logic
func timeHandler(w http.ResponseWriter, r *http.Request) {
    // Client already authenticated by Ephemos
    var req TimeRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    location, _ := time.LoadLocation(req.Timezone)
    now := time.Now().In(location)
    
    response := TimeResponse{
        Timezone:    req.Timezone,
        CurrentTime: now.Format("2006-01-02 15:04:05 MST"),
        Service:     "time-server",
    }
    json.NewEncoder(w).Encode(response)
}

// Add identity authentication - one line!
server, _ := ephemos.NewIdentityServer(ctx, "")
server.ServeHTTP(ctx, ":8080", mux)
```

**Endpoints:**
- `POST /time` - Send `{"timezone": "America/New_York"}`, get current time
- `GET /health` - Health check

## 💻 **Time Client** (`time-client/`)

A backend developer requesting times from the service:

```go
// Create authenticated client
client, _ := ephemos.NewIdentityClient(ctx, "")
httpClient, _ := client.NewHTTPClient("time-server", "localhost:8080")

// Make requests - authentication automatic
for _, tz := range []string{"UTC", "America/New_York", "Asia/Tokyo"} {
    request := TimeRequest{Timezone: tz}
    jsonData, _ := json.Marshal(request)
    
    resp, _ := httpClient.Post("http://localhost:8080/time", 
        "application/json", bytes.NewReader(jsonData))
    // Gets current time in that timezone
}
```

## 🚀 **Running the Example**

```bash
# Terminal 1 - Start server
cd time-server
go run main.go

# Terminal 2 - Run client  
cd time-client
go run main.go
```

**Output:**
```
⏰ Time received timezone=UTC time="2024-01-15 14:30:25 UTC"
⏰ Time received timezone=America/New_York time="2024-01-15 09:30:25 EST" 
⏰ Time received timezone=Asia/Tokyo time="2024-01-15 23:30:25 JST"
```

## ✅ **What Makes This Clean**

1. **Relatable business logic** - Everyone understands timezones
2. **No protobuf needed** - Standard HTTP JSON API
3. **Clear separation** - Business logic vs. authentication
4. **Real-world pattern** - How microservices actually work
5. **Lightweight** - Pure HTTP over mTLS, no gRPC complexity

## 🔐 **Authentication Flow**

```
Time Client → [Ephemos: Present SPIFFE cert] → Time Server
Time Server → [Ephemos: Verify SPIFFE cert] → Business Logic  
Business Logic → [Process timezone request] → Return time
```

Both developers just focus on their business logic. Ephemos handles all identity complexity.

## 📦 **Dependencies**

- ✅ Just `github.com/sufield/ephemos/pkg/ephemos`
- ❌ No protoc
- ❌ No .proto files  
- ❌ No gRPC
- ❌ No overengineering

This is the **clean pattern** for 0.1 release.