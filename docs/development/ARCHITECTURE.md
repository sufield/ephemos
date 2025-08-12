# Ephemos Architecture Documentation

## Hexagonal Architecture (Ports & Adapters)

Ephemos follows a clean hexagonal architecture pattern that enforces strict dependency rules and separation of concerns.

## Dependency Rule Diagram

```mermaid
graph TB
    subgraph "External World"
        CLI[CLI Interface]
        API[API Interface]
        GRPC[gRPC Transport]
        HTTP[HTTP Transport]
        SPIFFE[SPIFFE/SPIRE]
        CONFIG[Config Files]
    end
    
    subgraph "Adapters Layer"
        subgraph "Primary Adapters (Driving)"
            PA_CLI[CLI Adapter<br/>internal/adapters/primary/cli]
            PA_API[API Adapter<br/>internal/adapters/primary/api]
        end
        
        subgraph "Secondary Adapters (Driven)"
            SA_GRPC[gRPC Adapter<br/>internal/adapters/grpc]
            SA_HTTP[HTTP Adapter<br/>internal/adapters/http]
            SA_SPIFFE[SPIFFE Adapter<br/>internal/adapters/secondary/spiffe]
            SA_CONFIG[Config Adapter<br/>internal/adapters/secondary/config]
            SA_TRANSPORT[Transport Adapter<br/>internal/adapters/secondary/transport]
            SA_MEMID[Memory Identity<br/>internal/adapters/secondary/memidentity]
        end
    end
    
    subgraph "Ports Layer"
        subgraph "Inbound Ports"
            IP_SERVICE[Service Port<br/>internal/core/ports/service.go]
        end
        
        subgraph "Outbound Ports"
            OP_CONFIG[Configuration Port<br/>internal/core/ports/configuration.go]
            OP_IDENTITY[Identity Provider Port<br/>internal/core/ports/identity_provider.go]
            OP_TRANSPORT[Transport Port<br/>internal/core/ports/transport.go]
        end
    end
    
    subgraph "Core Domain"
        DOMAIN[Domain Model<br/>internal/core/domain]
        SERVICES[Business Services<br/>internal/core/services]
        ERRORS[Domain Errors<br/>internal/core/errors]
    end
    
    %% External to Primary Adapters
    CLI --> PA_CLI
    API --> PA_API
    
    %% Primary Adapters to Ports
    PA_CLI --> IP_SERVICE
    PA_API --> IP_SERVICE
    
    %% Ports to Core
    IP_SERVICE --> SERVICES
    SERVICES --> DOMAIN
    SERVICES --> ERRORS
    
    %% Core to Outbound Ports
    SERVICES --> OP_CONFIG
    SERVICES --> OP_IDENTITY
    SERVICES --> OP_TRANSPORT
    
    %% Outbound Ports to Secondary Adapters
    OP_CONFIG --> SA_CONFIG
    OP_IDENTITY --> SA_SPIFFE
    OP_IDENTITY --> SA_MEMID
    OP_TRANSPORT --> SA_TRANSPORT
    SA_TRANSPORT --> SA_GRPC
    SA_TRANSPORT --> SA_HTTP
    
    %% Secondary Adapters to External
    SA_CONFIG --> CONFIG
    SA_SPIFFE --> SPIFFE
    SA_GRPC --> GRPC
    SA_HTTP --> HTTP
    
    style DOMAIN fill:#f9f,stroke:#333,stroke-width:4px
    style SERVICES fill:#f9f,stroke:#333,stroke-width:4px
    style IP_SERVICE fill:#9ff,stroke:#333,stroke-width:2px
    style OP_CONFIG fill:#9ff,stroke:#333,stroke-width:2px
    style OP_IDENTITY fill:#9ff,stroke:#333,stroke-width:2px
    style OP_TRANSPORT fill:#9ff,stroke:#333,stroke-width:2px
```

## Import Graph Snapshot

The following diagram shows the actual import dependencies between packages in the Ephemos architecture:

```mermaid
graph LR
    subgraph "pkg/ephemos (Public API)"
        PKG_EPHEMOS[pkg/ephemos]
    end
    
    subgraph "Core Domain (No External Dependencies)"
        CORE_DOMAIN[internal/core/domain]
        CORE_ERRORS[internal/core/errors]
        CORE_PORTS[internal/core/ports]
        CORE_SERVICES[internal/core/services]
    end
    
    subgraph "Contract Layer"
        CONTRACT_CONFIG[internal/contract/<br/>configurationprovider]
        CONTRACT_IDENTITY[internal/contract/<br/>identityprovider]
        CONTRACT_TRANSPORT[internal/contract/<br/>transportprovider]
    end
    
    subgraph "Primary Adapters"
        PRIMARY_CLI[internal/adapters/<br/>primary/cli]
        PRIMARY_API[internal/adapters/<br/>primary/api]
    end
    
    subgraph "Secondary Adapters"
        SEC_CONFIG[internal/adapters/<br/>secondary/config]
        SEC_SPIFFE[internal/adapters/<br/>secondary/spiffe]
        SEC_MEMIDENTITY[internal/adapters/<br/>secondary/memidentity]
        SEC_TRANSPORT[internal/adapters/<br/>secondary/transport]
    end
    
    subgraph "Transport Adapters"
        ADAPT_GRPC[internal/adapters/grpc]
        ADAPT_HTTP[internal/adapters/http]
        ADAPT_INTERCEPTORS[internal/adapters/<br/>interceptors]
        ADAPT_LOGGING[internal/adapters/logging]
    end
    
    %% Core dependencies (none - pure domain)
    CORE_SERVICES --> CORE_DOMAIN
    CORE_SERVICES --> CORE_ERRORS
    CORE_SERVICES --> CORE_PORTS
    
    %% Contract implementations
    CONTRACT_CONFIG --> CORE_PORTS
    CONTRACT_IDENTITY --> CORE_PORTS
    CONTRACT_TRANSPORT --> CORE_PORTS
    
    %% Primary adapter dependencies
    PRIMARY_CLI --> CORE_PORTS
    PRIMARY_CLI --> CORE_SERVICES
    PRIMARY_CLI --> CONTRACT_CONFIG
    PRIMARY_CLI --> SEC_CONFIG
    
    PRIMARY_API --> CORE_PORTS
    PRIMARY_API --> CORE_SERVICES
    
    %% Secondary adapter dependencies
    SEC_CONFIG --> CORE_PORTS
    SEC_CONFIG --> CORE_DOMAIN
    
    SEC_SPIFFE --> CORE_PORTS
    SEC_SPIFFE --> CORE_DOMAIN
    SEC_SPIFFE -.->|uses| SPIFFE_LIBS[spiffe/go-spiffe/v2]
    
    SEC_MEMIDENTITY --> CORE_PORTS
    SEC_MEMIDENTITY --> CORE_DOMAIN
    
    SEC_TRANSPORT --> CORE_PORTS
    SEC_TRANSPORT --> ADAPT_GRPC
    SEC_TRANSPORT --> ADAPT_HTTP
    
    %% Transport adapter dependencies
    ADAPT_GRPC -.->|uses| GRPC_LIBS[google.golang.org/grpc]
    ADAPT_HTTP -.->|uses| HTTP_LIBS[net/http]
    ADAPT_INTERCEPTORS --> ADAPT_GRPC
    ADAPT_LOGGING --> CORE_DOMAIN
    
    %% Public API dependencies
    PKG_EPHEMOS --> PRIMARY_CLI
    PKG_EPHEMOS --> PRIMARY_API
    PKG_EPHEMOS --> SEC_CONFIG
    PKG_EPHEMOS --> SEC_SPIFFE
    
    style CORE_DOMAIN fill:#e1f5fe,stroke:#01579b,stroke-width:3px
    style CORE_SERVICES fill:#e1f5fe,stroke:#01579b,stroke-width:3px
    style CORE_PORTS fill:#e1f5fe,stroke:#01579b,stroke-width:3px
    style CORE_ERRORS fill:#e1f5fe,stroke:#01579b,stroke-width:3px
```

## Dependency Rules

### ✅ Allowed Dependencies

1. **Core Domain**: 
   - ❌ MUST NOT import from adapters
   - ❌ MUST NOT import from external libraries
   - ✅ CAN import from other core packages

2. **Ports (Interfaces)**:
   - ✅ CAN import from core/domain
   - ❌ MUST NOT import from adapters
   - ❌ MUST NOT import from external libraries

3. **Primary Adapters** (Driving/Inbound):
   - ✅ CAN import from core/ports
   - ✅ CAN import from core/services
   - ✅ CAN import from contracts
   - ✅ CAN import from external libraries
   - ❌ MUST NOT import from other adapters directly

4. **Secondary Adapters** (Driven/Outbound):
   - ✅ CAN import from core/ports
   - ✅ CAN import from core/domain
   - ✅ CAN import from external libraries
   - ❌ MUST NOT import from primary adapters
   - ❌ MUST NOT import from core/services

### 🔍 Import Analysis Summary

Based on the current codebase analysis:

```
Core Domain Packages:
├── internal/core/domain       → No external dependencies ✅
├── internal/core/errors       → No external dependencies ✅
├── internal/core/ports        → Only core/domain imports ✅
└── internal/core/services     → Only core/* imports ✅

Primary Adapters:
├── internal/adapters/primary/cli → Imports core + contracts ✅
└── internal/adapters/primary/api → Imports core + contracts ✅

Secondary Adapters:
├── internal/adapters/secondary/config      → Imports core/ports ✅
├── internal/adapters/secondary/spiffe      → Imports core + SPIFFE lib ✅
├── internal/adapters/secondary/memidentity → Imports core/ports ✅
└── internal/adapters/secondary/transport   → Imports core + transport adapters ✅

Transport Implementations:
├── internal/adapters/grpc         → External gRPC libraries ✅
├── internal/adapters/http         → Standard net/http ✅
├── internal/adapters/interceptors → gRPC interceptors ✅
└── internal/adapters/logging      → Logging utilities ✅
```

## Key Architectural Principles

1. **Dependency Inversion**: Core domain defines interfaces (ports) that adapters implement
2. **Single Responsibility**: Each adapter has one clear responsibility
3. **Interface Segregation**: Ports are small and focused
4. **Clean Boundaries**: No circular dependencies between layers
5. **Testability**: Core can be tested without any external dependencies

## Testing Architecture Compliance

The architecture is enforced through automated tests:

```go
// internal/core/ports/architecture_test.go
func TestNoCoreImportsAdapters(t *testing.T) {
    // Verifies core packages don't import from adapters
}

func TestNoCircularDependencies(t *testing.T) {
    // Ensures no circular imports exist
}
```

## Benefits of This Architecture

1. **Maintainability**: Changes in external systems don't affect core business logic
2. **Testability**: Core domain can be tested in isolation
3. **Flexibility**: Easy to swap implementations (e.g., SPIFFE → OAuth)
4. **Clarity**: Clear separation of concerns and responsibilities
5. **Security**: Security concerns isolated in specific adapters

## Example: Adding a New Transport

To add a new transport (e.g., WebSocket):

1. Create adapter: `internal/adapters/websocket/`
2. Implement transport port: `internal/core/ports/transport.go`
3. Register in transport adapter: `internal/adapters/secondary/transport/`
4. No changes needed in core domain! ✅

## Import Verification Commands

```bash
# Check for illegal imports from core to adapters
go list -f '{{.ImportPath}} {{.Imports}}' ./internal/core/... | grep adapters

# Visualize dependency graph
go mod graph | grep internal/

# Check for circular dependencies
go list -f '{{join .Deps "\n"}}' ./internal/... | sort | uniq -c | sort -rn
```

---

*Last Updated: December 2024*
*Architecture Version: 2.0*