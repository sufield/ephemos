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

## Sequence Diagrams

### Server Boot Sequence (SVID Fetch & mTLS Setup)

```mermaid
sequenceDiagram
    participant App as Application
    participant SvcPort as Service Port
    participant SvcCore as Core Service
    participant ConfigPort as Config Port
    participant IdentPort as Identity Port
    participant TransPort as Transport Port
    participant SpiffeAdapter as SPIFFE Adapter
    participant SpireAgent as SPIRE Agent
    participant GrpcAdapter as gRPC Adapter
    
    Note over App, GrpcAdapter: Server Initialization Phase
    
    App->>+SvcPort: NewServer(configPath)
    SvcPort->>+SvcCore: CreateService()
    
    Note over SvcCore, ConfigPort: Configuration Loading
    SvcCore->>+ConfigPort: LoadConfig(configPath)
    ConfigPort->>ConfigPort: ValidateConfig()
    ConfigPort-->>-SvcCore: Config{Service, Transport, SPIFFE}
    
    Note over SvcCore, IdentPort: Identity Bootstrap
    SvcCore->>+IdentPort: InitializeIdentity(spiffeConfig)
    IdentPort->>+SpiffeAdapter: Connect()
    
    SpiffeAdapter->>+SpireAgent: Workload API Connection
    SpireAgent-->>-SpiffeAdapter: Connection Established
    
    SpiffeAdapter->>+SpireAgent: FetchJWTSVID()
    SpireAgent->>SpireAgent: Validate Workload
    SpireAgent-->>-SpiffeAdapter: JWT-SVID + Private Key
    
    SpiffeAdapter->>+SpireAgent: FetchX509SVID()
    SpireAgent-->>-SpiffeAdapter: X.509-SVID + Private Key + Bundle
    
    SpiffeAdapter-->>-IdentPort: Identity{SVID, PrivateKey, TrustBundle}
    IdentPort-->>-SvcCore: Identity Ready
    
    Note over SvcCore, TransPort: Transport Setup
    SvcCore->>+TransPort: InitializeTransport(identity, config)
    TransPort->>+GrpcAdapter: CreateServer(tlsConfig)
    
    GrpcAdapter->>GrpcAdapter: Configure mTLS
    Note right of GrpcAdapter: - Server cert from X.509-SVID<br/>- Client CA from trust bundle<br/>- Require client certs
    
    GrpcAdapter-->>-TransPort: Server{Listener, TLSConfig}
    TransPort-->>-SvcCore: Transport Ready
    
    SvcCore-->>-SvcPort: Service Ready
    SvcPort-->>-App: Server Instance
    
    Note over App, GrpcAdapter: Server Start Phase
    App->>SvcPort: Start()
    SvcPort->>SvcCore: StartServices()
    SvcCore->>TransPort: Listen()
    TransPort->>GrpcAdapter: Serve()
    
    Note over GrpcAdapter: Server accepting mTLS connections
    GrpcAdapter-->>App: Server Running
```

### Client Connect Sequence

```mermaid
sequenceDiagram
    participant Client as Client App
    participant ClientSvc as Client Service
    participant IdentPort as Identity Port
    participant TransPort as Transport Port
    participant SpiffeAdapter as SPIFFE Adapter
    participant SpireAgent as SPIRE Agent (Client)
    participant GrpcAdapter as gRPC Adapter
    participant Server as Server Process
    participant SpireAgentSrv as SPIRE Agent (Server)
    
    Note over Client, SpireAgentSrv: Client Connection Establishment
    
    Client->>+ClientSvc: NewClient(configPath)
    
    Note over ClientSvc, IdentPort: Client Identity Setup
    ClientSvc->>+IdentPort: GetIdentity()
    IdentPort->>+SpiffeAdapter: FetchCurrentIdentity()
    
    SpiffeAdapter->>+SpireAgent: FetchX509SVID()
    SpireAgent->>SpireAgent: Validate Client Workload
    SpireAgent-->>-SpiffeAdapter: X.509-SVID + Key + Bundle
    
    SpiffeAdapter-->>-IdentPort: ClientIdentity{SVID, Key, Bundle}
    IdentPort-->>-ClientSvc: Identity Ready
    
    Note over ClientSvc, TransPort: Connection Setup
    ClientSvc->>+TransPort: Connect(serverAddress, identity)
    TransPort->>+GrpcAdapter: CreateConnection(tlsConfig)
    
    Note right of GrpcAdapter: Client TLS Config:<br/>- Client cert from X.509-SVID<br/>- Server CA from trust bundle<br/>- SPIFFE ID verification
    
    GrpcAdapter->>+Server: TLS Handshake
    
    Note over Server, SpireAgentSrv: Server-side Validation
    Server->>Server: Validate Client Certificate
    Server->>Server: Extract SPIFFE ID
    Server->>Server: Check Authorized Clients List
    
    alt Client Authorized
        Server-->>GrpcAdapter: TLS Handshake Complete
        GrpcAdapter-->>-TransPort: Secure Connection Established
        TransPort-->>-ClientSvc: Connection Ready
        ClientSvc-->>-Client: Client Ready
        
        Note over Client, Server: Secure mTLS Communication
        Client->>+ClientSvc: MakeRequest(data)
        ClientSvc->>TransPort: Send(encryptedData)
        TransPort->>GrpcAdapter: gRPC Call
        GrpcAdapter->>+Server: Authenticated Request
        Server->>Server: Process Request
        Server-->>-GrpcAdapter: Response
        GrpcAdapter-->>TransPort: gRPC Response
        TransPort-->>ClientSvc: DecryptedResponse
        ClientSvc-->>-Client: Result
        
    else Client Not Authorized
        Server-->>GrpcAdapter: TLS Handshake Failed
        GrpcAdapter-->>-TransPort: Connection Rejected
        TransPort-->>-ClientSvc: AuthorizationError
        ClientSvc-->>-Client: Connection Failed
    end
```

### Certificate Rotation Sequence

```mermaid
sequenceDiagram
    participant App as Application
    participant IdentWatcher as Identity Watcher
    participant SpiffeAdapter as SPIFFE Adapter
    participant SpireAgent as SPIRE Agent
    participant TransService as Transport Service
    participant GrpcServer as gRPC Server
    participant ActiveConns as Active Connections
    participant NewConns as New Connections
    
    Note over App, NewConns: Automatic Certificate Rotation
    
    Note over IdentWatcher, SpireAgent: Background Identity Monitoring
    IdentWatcher->>+SpiffeAdapter: WatchIdentity()
    SpiffeAdapter->>+SpireAgent: StreamSVIDs()
    
    loop Every 30 seconds (configurable)
        SpireAgent->>SpireAgent: Check Certificate Expiry
        
        alt Certificate Near Expiry (< 1/3 lifetime)
            SpireAgent->>SpireAgent: Generate New X.509-SVID
            SpireAgent-->>SpiffeAdapter: NewSVID{Cert, Key, Bundle}
            SpiffeAdapter-->>-IdentWatcher: IdentityUpdated Event
            
            Note over IdentWatcher, TransService: Hot Certificate Reload
            IdentWatcher->>+TransService: UpdateIdentity(newSVID)
            
            TransService->>TransService: Create New TLS Config
            TransService->>+GrpcServer: UpdateTLSConfig(newConfig)
            
            Note over GrpcServer, ActiveConns: Graceful Certificate Transition
            
            GrpcServer->>GrpcServer: Update Server Certificate
            
            Note right of GrpcServer: New connections use new cert<br/>Old connections continue with old cert<br/>until natural termination
            
            GrpcServer->>+NewConns: Use New Certificate
            NewConns-->>-GrpcServer: TLS with New Cert
            
            Note over ActiveConns: Existing connections remain active<br/>with old certificate until they close naturally
            
            GrpcServer-->>-TransService: Certificate Updated
            TransService-->>-IdentWatcher: Update Complete
            
            Note over IdentWatcher: Log Certificate Rotation Success
            IdentWatcher->>IdentWatcher: Log: Certificate rotated successfully
            
        else Certificate Still Valid
            SpireAgent-->>SpiffeAdapter: No Update Needed
            SpiffeAdapter-->>-IdentWatcher: Identity Current
        end
        
        Note over IdentWatcher, SpireAgent: Wait for next check interval
    end
    
    Note over App, NewConns: Zero-Downtime Certificate Management
    
    rect rgb(240, 255, 240)
        Note over ActiveConns, NewConns: Benefits of This Approach:<br/>• No connection drops during rotation<br/>• No service interruption<br/>• Automatic certificate lifecycle management<br/>• Configurable rotation timing<br/>• Observability through logging
    end
```

### Certificate Rotation Error Handling

```mermaid
sequenceDiagram
    participant IdentWatcher as Identity Watcher  
    participant SpiffeAdapter as SPIFFE Adapter
    participant SpireAgent as SPIRE Agent
    participant TransService as Transport Service
    participant AlertSvc as Alert Service
    participant HealthCheck as Health Check
    
    Note over IdentWatcher, HealthCheck: Certificate Rotation Failure Scenarios
    
    IdentWatcher->>+SpiffeAdapter: WatchIdentity()
    SpiffeAdapter->>+SpireAgent: StreamSVIDs()
    
    alt SPIRE Agent Unavailable
        SpireAgent-->>SpiffeAdapter: Connection Error
        SpiffeAdapter-->>IdentWatcher: SPIRE Agent Unreachable
        
        IdentWatcher->>+AlertSvc: Alert(SPIRE_AGENT_DOWN)
        AlertSvc-->>-IdentWatcher: Alert Sent
        
        IdentWatcher->>+HealthCheck: SetUnhealthy("SPIRE connection lost")
        HealthCheck-->>-IdentWatcher: Health Status Updated
        
        Note over IdentWatcher: Continue using current certificate<br/>until SPIRE agent recovers
        
    else Certificate Fetch Failure
        SpireAgent->>SpireAgent: Certificate Generation Failed
        SpireAgent-->>SpiffeAdapter: SVID Fetch Error
        SpiffeAdapter-->>IdentWatcher: Certificate Update Failed
        
        IdentWatcher->>IdentWatcher: Retry with Exponential Backoff
        
        loop Max 5 retries
            IdentWatcher->>SpiffeAdapter: RetryFetchSVID()
            SpiffeAdapter->>SpireAgent: FetchX509SVID()
            
            alt Retry Successful
                SpireAgent-->>SpiffeAdapter: New SVID
                SpiffeAdapter-->>IdentWatcher: Certificate Updated
                IdentWatcher->>IdentWatcher: Reset Retry Counter
                
            else Retry Failed
                SpireAgent-->>SpiffeAdapter: Still Failing
                SpiffeAdapter-->>IdentWatcher: Retry Failed
                IdentWatcher->>IdentWatcher: Increment Retry Counter
            end
        end
        
        alt All Retries Exhausted
            IdentWatcher->>+AlertSvc: Alert(CERTIFICATE_ROTATION_FAILED)
            AlertSvc-->>-IdentWatcher: Critical Alert Sent
            
            IdentWatcher->>+HealthCheck: SetDegraded("Certificate rotation failing")
            HealthCheck-->>-IdentWatcher: Health Status Updated
        end
        
    else TLS Config Update Failure  
        SpireAgent-->>SpiffeAdapter: New SVID Available
        SpiffeAdapter-->>IdentWatcher: New Certificate
        IdentWatcher->>+TransService: UpdateIdentity(newSVID)
        
        TransService->>TransService: TLS Config Creation Failed
        TransService-->>IdentWatcher: Config Update Error
        
        IdentWatcher->>+AlertSvc: Alert(TLS_CONFIG_UPDATE_FAILED)
        AlertSvc-->>-IdentWatcher: Alert Sent
        
        Note over IdentWatcher: Keep using previous working certificate<br/>Do not disrupt service
        
    else Certificate Near Expiry With No Rotation
        IdentWatcher->>IdentWatcher: Monitor Certificate Expiry Time
        
        alt Certificate < 10% lifetime remaining
            IdentWatcher->>+AlertSvc: Alert(CERTIFICATE_EXPIRY_WARNING)
            AlertSvc-->>-IdentWatcher: Warning Alert Sent
            
        else Certificate < 1% lifetime remaining  
            IdentWatcher->>+AlertSvc: Alert(CERTIFICATE_EXPIRY_CRITICAL)
            AlertSvc-->>-IdentWatcher: Critical Alert Sent
            
            IdentWatcher->>+HealthCheck: SetUnhealthy("Certificate expiring soon")
            HealthCheck-->>-IdentWatcher: Health Status Updated
        end
    end
    
    Note over IdentWatcher, HealthCheck: Rotation Monitoring & Recovery
```

---

*Last Updated: December 2024*
*Architecture Version: 2.0*