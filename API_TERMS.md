# Ephemos API Terms & Definitions

This document defines the intent-based, self-explaining terms used throughout the Ephemos API.

## 🎯 **Core Domain Terms**

### **IdentitySetting**
- **Intent**: Configuration for how identity authentication should behave
- **Replaces**: `AuthConfig`, `SPIFFEAuthConfig`, `IdentityConfig`  
- **Usage**: `ephemos.IdentitySetting{AllowedServices: []string{...}}`
- **Why**: Reduces semantic gap - these are settings for identity behavior, not just "config"

### **IdentityDocument**  
- **Intent**: A document that proves/represents a service's identity
- **Replaces**: `AuthenticatedIdentity` (in some contexts), `SPIFFEIdentity`
- **Usage**: `identityDoc := chimiddleware.GetIdentityDocument(ctx)`
- **Why**: Maps to how it actually works - services present identity documents for verification

### **IdentityAuthentication**
- **Intent**: Middleware that authenticates services using their identity documents  
- **Replaces**: `SPIFFEAuth`, `AuthMiddleware`
- **Usage**: `r.Use(chimiddleware.IdentityAuthentication(settings))`
- **Why**: Self-explaining - this middleware performs identity authentication

### **GetIdentityDocument**
- **Intent**: Retrieve the identity document from an authenticated request
- **Replaces**: `GetSPIFFEIdentity`, `GetAuthenticatedIdentity`
- **Usage**: `identityDoc := chimiddleware.GetIdentityDocument(r.Context())`
- **Why**: Clear intent - getting the document that proves identity

## 🔄 **Domain-Accurate Terms (Keep)**

### **AuthenticatedIdentity**
- **Intent**: Represents a verified, authenticated identity (domain concept)
- **Keep**: Yes - this accurately represents the domain concept
- **Usage**: `var identity *AuthenticatedIdentity = identityDoc.AuthenticatedIdentity()`
- **Why**: Correct domain term for the verified identity concept

### **IdentityServer** & **IdentityClient**
- **Intent**: Server/client that handle identity-based authentication
- **Keep**: Yes - already good domain terms
- **Usage**: `server, _ := ephemos.IdentityServer(ctx, options...)`
- **Why**: Clear intent and good domain abstraction

## 📦 **Framework-Specific Naming**

### **Package Naming Pattern**
- **Chi**: `chimiddleware.IdentityAuthentication()`
- **Gin**: `ginmiddleware.IdentityAuthentication()`
- **Core**: `ephemos.IdentitySetting{}`

### **Why This Pattern**
- **Clear dependencies**: Immediately see which library provides what capability
- **Consistent naming**: Same function name across frameworks
- **Intent-focused**: Function names explain what they do, not how they work

## 🔍 **Before vs After Comparison**

| **Before (Implementation-Focused)** | **After (Intent-Focused)** |
|-------------------------------------|----------------------------|
| `SPIFFEAuth()` | `IdentityAuthentication()` |
| `AuthConfig` | `IdentitySetting` |
| `GetSPIFFEIdentity()` | `GetIdentityDocument()` |
| `SPIFFEAuthConfig` | `IdentitySetting` |

## ✨ **Self-Explaining Code Examples**

### **Chi Example**
```go
// Intent is immediately clear from reading the code
r := chi.NewRouter()
r.Use(chimiddleware.IdentityAuthentication(ephemos.IdentitySetting{
    AllowedServices: []string{"payment-service"},
}))

func handler(w http.ResponseWriter, r *http.Request) {
    // Clear intent: getting the identity document
    identityDoc := chimiddleware.GetIdentityDocument(r.Context())
    log.Printf("Request from: %s", identityDoc.ServiceName())
}
```

### **Gin Example**  
```go
// Same clear intent, different framework
r := gin.Default()
r.Use(ginmiddleware.IdentityAuthentication(ephemos.IdentitySetting{
    AllowedServices: []string{"payment-service"},
}))

func handler(c *gin.Context) {
    // Clear intent: getting the identity document
    identityDoc := ginmiddleware.GetIdentityDocument(c.Request.Context())
    log.Printf("Request from: %s", identityDoc.ServiceName())
}
```

## 🎯 **Design Principles**

### **1. Intent Over Implementation**
- ✅ `IdentityAuthentication()` (what it does)
- ❌ `SPIFFEAuth()` (how it works)

### **2. Self-Explaining**
- Code should read like English
- Function names explain purpose
- No need for extensive documentation to understand intent

### **3. Domain Accuracy**  
- Terms should map to how the system actually works
- Services authenticate using identity documents
- Settings configure behavior
- Documents prove identity

### **4. Semantic Gap Reduction**
- Reduce distance between code and domain concepts
- Use terms that domain experts would recognize
- Avoid technical jargon when domain terms exist

## 🔮 **Future Term Additions**

As functionality is implemented, additional terms will be defined following these principles:
- Intent-based naming
- Self-explaining code
- Domain accuracy
- Semantic gap reduction

**Note**: This document will be updated as the API evolves and new terms are needed.