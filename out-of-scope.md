## Scope Definition: Core vs Contrib

> 📊 **See [mvp.md](mvp.md) for comprehensive authentication method comparison tables and framework positioning.**

### ✅ **Core Scope** (this repository)
- **SPIFFE identity management**: X.509 SVID certificates and trust bundles
- **HTTP over mTLS**: Authenticated HTTP connections using X.509 SVIDs
- **Core primitives**: Certificate fetching, trust bundle management, TLS configuration
- **Configuration**: Service identity and trust domain management
- **Domain logic**: Core business logic for service identity and HTTP authentication

### 🚫 **Out of Core Scope** (moved to contrib)
- **HTTP middleware implementations**: Chi, Gin, custom HTTP framework integrations
- **Framework-specific code**: Web framework dependencies and routing patterns
- **gRPC transport**: gRPC client/server and interceptors (future release)
- **JWT SVIDs**: JWT-based authentication (future release)
- **Non-auth features**: Logging, metrics, and other cross-cutting concerns

### 🔌 **Plugin Points** (interfaces for contrib)
Core exposes these interfaces for contrib extensions:
- `IdentityService.GetCertificate()` - Access to X.509 SVID certificates
- `IdentityService.GetTrustBundle()` - Access to trust bundles  
- `HTTPClient()` - HTTP client with automatic mTLS configuration
- Authorization policy builders - For peer validation with X.509 SVIDs
- TLS configuration helpers - For custom mTLS setup

**Note**: Framework middleware implementations are explicitly contrib-only. Core provides HTTP + mTLS primitives; contrib provides Chi/Gin/framework integrations.

### Review of the Updated Architecture Plan
The architecture has been updated to focus on HTTP + X.509 SVIDs for MVP, which is focused, incremental, and reversible, minimizing risk in the initial release. This correctly identifies framework-specific middleware (e.g., Chi/Gin handlers, routing patterns) as removable from the core while preserving HTTP + mTLS and SPIFFE identity primitives. This aligns with best practices for Go libraries: keeping the core lightweight and HTTP-focused (e.g., identity + mTLS logic), with framework-specific extensions (like Chi/Gin middleware) in a separate contrib module or repo. The plan follows patterns seen in projects like OpenTelemetry-go, where core APIs are transport-specific but framework-agnostic, and integrations (e.g., for Gin/Chi) live in contrib for modularity and independent versioning.

**Strengths**:
- **Granularity**: Small PRs (e.g., docs first, then code removals) make reviews easy and allow quick reverts if issues arise.
- **Safety Checks**: Using `rg` (ripgrep) for code searches and `go test` ensures no lingering references or broken builds.
- **Completeness**: Covers code, tests, docs, and build files. Retains essential core components (domain/ports/services, gRPC interceptors).
- **Rationale**: Emphasizes contrib for HTTP adoption, which keeps core bloat-free while enabling easy extensions—common in Go ecosystems where stdlib + routers like Chi are preferred for minimalism over full frameworks like Gin.

**Potential Issues/Gaps**:
- **Over-Removal Risk**: The SPIFFE adapters (e.g., `svidSourceAdapter`, `bundleSourceAdapter`) and `extractTLSConfig` might be reusable for gRPC mTLS creds (e.g., via `grpc.WithTransportCredentials(spiffetls.MTLSClientCredentials(...))`). If gRPC transport uses similar go-spiffe configs, removing them could break gRPC indirectly. Verify if they're HTTP-only.
- **Export Gaps**: Contrib middleware needs access to core primitives (e.g., `IdentityService.GetCertificate()`, `GetTrustBundle()`, authorizers). If not already exported (e.g., via `pkg/ephemos`), add an export PR.
- **Gin Symmetry**: Plan mentions adding Gin later but doesn't include it—add a PR stub for contrib/middleware/gin to mirror Chi.
- **Testing Coverage**: Post-removal, ensure gRPC tests cover mTLS fully; add integration tests in contrib for HTTP.
- **Dependencies**: No checks for import cycles or external deps (e.g., ensure core has no `net/http` imports after removal).
- **Best Practices Alignment**: Draw from Go community: Favor composable middleware (as in Chi/Gin), keep core interfaces extensible for contrib. Use multi-module setup in monorepo for contrib (as suggested earlier).

### Improved PR Plan
I've refined the plan: Added a PR for exports and Gin stub; enhanced checks (e.g., `go mod graph` for deps); clarified keepers/removals; added a final verification PR; incorporated best practices (e.g., ensure contrib examples demonstrate zero core bloat). Still assumes .sh ignored; focuses on file-level changes.

#### PR0: Scope & Plugin Points (Docs Only)
- **Changes**:
  - README.md: Update to "Core = SPIFFE identity + mTLS (gRPC-focused). HTTP support via contrib middleware for Chi/Gin."
  - Add contrib/README.md: "Extensions for frameworks; consumes core primitives like certs, bundles, and authorizers."
  - out-of-scope.md: Explicitly list "HTTP middleware implementations" as contrib-only; note plugin points (e.g., exposed interfaces for auth policies).
- **Accept Criteria**: Docs only; no code changes. Run `git diff --stat` to confirm.

#### PR1: Remove Framework-Specific Middleware from Core
- **Goal**: Eliminate framework coupling; core exposes only HTTP + mTLS primitives and identity management.
- **Changes** (Surgical Edits):
  - internal/adapters/primary/api/client.go:
    - Keep `HTTPClient()`, `extractTLSConfig()`, `createSVIDSource()`, `createBundleSource()` as core HTTP primitives
    - Remove any Chi/Gin-specific middleware or routing code
    - Keep HTTP client creation, mTLS configuration, identity management
  - Move framework middleware to contrib/middleware/chi/ and contrib/middleware/gin/
  - Remove any framework imports from core (chi, gin, etc.)
- **Follow-ups**: Prune framework references in core tests/files.
- **Accept Criteria**:
  ```
  rg -n '"github.com/go-chi/chi|gin-gonic/gin"' internal pkg # No framework imports in core
  rg -n 'router\.|middleware\.' internal pkg # No framework-specific patterns in core
  go test ./... # All pass
  go mod graph | grep -E "chi|gin" # Ensure no framework deps in core module
  ```

#### PR2: Export HTTP Primitives for Contrib Middleware
- **Goal**: Ensure contrib can build HTTP middleware without internal access; core remains thin but extensible.
- **Changes**:
  - In pkg/ephemos (public facade): Export necessary HTTP types/interfaces if not already (e.g., `IdentityService` with `GetCertificate()`, `GetTrustBundle()`; `HTTPClient()` with mTLS; `AuthenticationPolicy`; authorizer builders like `func NewAuthorizerFromConfig(cfg) tlsconfig.Authorizer`).
  - Add pkg/ephemos/authorizer.go: Simple funcs for common HTTP authorizers (e.g., `AuthorizeMemberOf(domain string) tlsconfig.Authorizer` wrapping go-spiffe).
  - Export HTTP client helpers for contrib frameworks to build middleware.
- **Accept Criteria**:
  ```
  rg -n '\".*internal/' pkg/ephemos # No internal imports in public pkg
  go doc pkg/ephemos # Verify exported symbols include HTTP client, cert/bundle access
  ```

#### PR3: Move HTTP Client Guidance to Contrib
- **Goal**: Guide users on HTTP adoption without core docs pollution.
- **Changes**:
  - Move docs/HTTP_CLIENT.md → contrib/docs/HTTP_CLIENT.md.
  - Add contrib/examples/http_client.go: Minimal example building `http.Client` with core primitives (e.g., fetch cert/bundle, use `tlsconfig.MTLSClientConfig(source, bundle, authorizer)`).
  - Update contrib/README.md: Link to example; emphasize "Core provides certs/bundles; contrib glues to http.Transport".
- **Accept Criteria**: No core changes; contrib builds independently (`go build ./contrib/...`).

#### PR4: Remove Non-Auth Interceptors
- **Goal**: Core interceptors focus on auth/identity only (aligns with zero-trust SPIFFE).
- **Changes** (If not already clean):
  - Delete internal/adapters/interceptors/logging.go, logging_test.go, metrics.go, metrics_test.go.
  - Keep auth.go, identity_propagation.go (gRPC-specific for SPIFFE ID propagation).
- **Accept Criteria**:
  ```
  rg -n 'Logging.*|Metrics.*|AuthMetrics' internal/adapters/interceptors # Only auth/identity files
  go test ./internal/adapters/interceptors/... # Pass
  ```

#### PR5: Remove Non-Auth HTTP Scaffolding from Core
- **Goal**: No HTTP middleware or config in core; frameworks in contrib.
- **Changes**:
  - Delete internal/middleware/config.go (assuming it's HTTP-only; verify it's not used for gRPC).
  - Confirm no Chi/Gin imports in core (already planned).
- **Accept Criteria**:
  ```
  rg -n 'package .*middleware' internal # No matches
  rg -n '"github.com/go-chi/chi|gin-gonic/gin"' . # No matches outside contrib
  ```

#### PR6: Add Gin Middleware Stub in Contrib
- **Goal**: Symmetry with Chi; prepare for Gin extensions without core changes.
- **Changes**:
  - Create contrib/middleware/gin/ (mirror chi/): Add gin.IdentityMiddleware(config) using core cert/bundle/authorizer.
  - Include basic example/test: e.g., Gin handler with SPIFFE auth.
- **Accept Criteria**:
  ```
  go test ./contrib/middleware/gin/... # Pass
  # Ensure gin imports only in this subdir
  ```

#### PR7: Build and Final Verification
- **Goal**: Tidy up; confirm core is HTTP-free.
- **Changes**:
  - Remove Bazel/Make targets for deleted files.
  - `go mod tidy` in root and contrib modules.
  - Add arch test: e.g., internal/arch/no_http_in_core_test.go checking imports.
- **Accept Criteria**:
  ```
  go build ./...
  go test ./...
  go vet ./...
  # Optional: golangci-lint run for style/deadcode
  ```

#### What Remains (Core Essentials)
- **Unchanged**: Domain/ports/services (identity logic); secondary adapters (spiffe, transport/gRPC, config); primary API (gRPC client/server); auth interceptors; factory.
- **Post-Refactor**: Core is gRPC-first but extensible (e.g., via exported authorizers). Contrib handles HTTP/Chi/Gin, consuming only public APIs—no internals.

#### Why Improved Plan is Safer/Better
- **Added Exports/PR2**: Ensures contrib viability without hacks; follows "interface-based extension" best practices.
- **Gin Inclusion**: Balances Chi; common in Go web projects to support multiple routers.
- **Enhanced Checks**: Dep graphs prevent sneaky imports; arch tests enforce boundaries.
- **Risk Mitigation**: Verify adapters' reusability before full removal; add contrib tests early.
- **Alignment**: Matches community patterns—core minimal like stdlib + Chi for services, contrib for frameworks. If breaking changes, semver bump core to v2.