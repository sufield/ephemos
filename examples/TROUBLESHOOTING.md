# Ephemos Troubleshooting Guide

This guide helps you diagnose and fix common issues when building services with the Ephemos library.

## Table of Contents
- [Quick Diagnostics](#quick-diagnostics)
- [SPIRE Connection Issues](#spire-connection-issues)
- [Certificate Problems](#certificate-problems)
- [Service Registration Issues](#service-registration-issues)
- [mTLS Handshake Failures](#mtls-handshake-failures)
- [Authorization Problems](#authorization-problems)
- [Performance Issues](#performance-issues)
- [Configuration Errors](#configuration-errors)
- [Development Environment](#development-environment)
- [Production Issues](#production-issues)

## Quick Diagnostics

### Health Check Commands

```bash
# Check SPIRE server status
docker exec spire-server /opt/spire/bin/spire-server healthcheck

# Check SPIRE agent status  
docker exec spire-agent /opt/spire/bin/spire-agent healthcheck

# Check if agent socket exists
ls -la /tmp/spire-agent/public/api.sock

# Test service connectivity
./bin/ephemos health --config config/echo-server.yaml

# Check registered entries
docker exec spire-server /opt/spire/bin/spire-server entry show
```

### Common Log Locations

```bash
# Docker container logs
docker logs spire-server
docker logs spire-agent

# Application logs (if using structured logging)
tail -f /var/log/ephemos/service.log

# System journal (on systemd systems)
journalctl -u spire-agent -f
```

## SPIRE Connection Issues

### Problem: "connection refused" or "no such file or directory"

**Symptoms:**
```
Failed to create SPIFFE provider: connection refused
Could not connect to SPIRE agent: dial unix /tmp/spire-agent/public/api.sock: no such file
```

**Root Causes & Solutions:**

1. **SPIRE Agent Not Running**
   ```bash
   # Check if agent is running
   docker ps | grep spire-agent
   
   # Start agent if stopped
   cd scripts/demo && ./start-spire.sh
   ```

2. **Wrong Socket Path**
   ```yaml
   # Check config.yaml - common mistake
   spiffe:
     socket_path: /tmp/spire-agent/public/api.sock  # Correct
     # not: /var/run/spire/sockets/agent.sock        # Wrong for demo
   ```

3. **Permission Issues**
   ```bash
   # Check socket permissions
   ls -la /tmp/spire-agent/public/api.sock
   # Should be readable by your user
   
   # Fix permissions if needed
   sudo chown $(whoami):$(whoami) /tmp/spire-agent/public/api.sock
   ```

4. **Docker Network Issues**
   ```bash
   # Check if Docker containers are on same network
   docker network ls
   docker inspect spire-agent | grep NetworkMode
   
   # Recreate containers if network is wrong
   cd scripts/demo && ./cleanup.sh && ./start-spire.sh
   ```

### Problem: "SPIRE server unreachable"

**Symptoms:**
```
Failed to fetch bundle: rpc error: code = Unavailable desc = connection error
```

**Solutions:**
```bash
# Check SPIRE server is listening
docker exec spire-server netstat -tlnp | grep 8081

# Check agent configuration
docker exec spire-agent cat /opt/spire/conf/agent.conf
# Verify server_address and server_port are correct

# Test connectivity from agent
docker exec spire-agent ping spire-server
```

## Certificate Problems

### Problem: Certificate expired or invalid

**Symptoms:**
```
x509: certificate has expired or is not yet valid
tls: bad certificate
```

**Diagnosis:**
```bash
# Check certificate validity
docker exec spire-agent /opt/spire/bin/spire-agent api fetch x509
# Look for NotAfter field

# Check server time
date
docker exec spire-server date
docker exec spire-agent date
# Times should be synchronized
```

**Solutions:**
```bash
# Force certificate renewal
docker restart spire-agent

# Check SPIRE server logs for CA issues
docker logs spire-server | grep -i certificate

# Verify system clock synchronization
sudo timedatectl status
```

### Problem: Certificate chain validation fails

**Symptoms:**
```
x509: certificate signed by unknown authority
tls: failed to verify certificate: x509: certificate signed by unknown authority
```

**Solutions:**
```bash
# Check if trust bundle is available
docker exec spire-server /opt/spire/bin/spire-server bundle show

# Force bundle refresh on agent
docker restart spire-agent

# Verify bundle propagation
docker exec spire-agent /opt/spire/bin/spire-agent api fetch x509 -write /tmp/
openssl x509 -in /tmp/svid.0.pem -noout -issuer
openssl x509 -in /tmp/bundle.0.pem -noout -subject
```

## Service Registration Issues

### Problem: Service not registered with SPIRE

**Symptoms:**
```
no identity SVID available
could not get identity: rpc error: code = PermissionDenied
```

**Diagnosis:**
```bash
# List all registered entries
docker exec spire-server /opt/spire/bin/spire-server entry show

# Check for your service SPIFFE ID
docker exec spire-server /opt/spire/bin/spire-server entry show -spiffeID spiffe://example.org/echo-server
```

**Solutions:**
```bash
# Register service manually
docker exec spire-server /opt/spire/bin/spire-server entry create \
    -spiffeID spiffe://example.org/echo-server \
    -parentID spiffe://example.org/spire-agent \
    -selector unix:user:$(id -u)

# Or use Ephemos CLI
./bin/ephemos register --config config/echo-server.yaml --selector unix:user:$(id -u)
```

### Problem: Wrong selectors or parent ID

**Symptoms:**
```
no identity SVID available
could not get identity: rpc error: code = PermissionDenied
```

**Diagnosis:**
```bash
# Check current user ID and selectors
id -u  # Note the user ID
ps aux | grep echo-server  # Check what user process runs as

# Check registered selectors
docker exec spire-server /opt/spire/bin/spire-server entry show -spiffeID spiffe://example.org/echo-server
```

**Solutions:**
```bash
# Delete wrong entry
docker exec spire-server /opt/spire/bin/spire-server entry delete \
    -spiffeID spiffe://example.org/echo-server

# Re-register with correct selectors
docker exec spire-server /opt/spire/bin/spire-server entry create \
    -spiffeID spiffe://example.org/echo-server \
    -parentID spiffe://example.org/spire-agent \
    -selector unix:user:$(id -u)
```

## mTLS Handshake Failures

### Problem: TLS handshake timeout

**Symptoms:**
```
context deadline exceeded
connection reset by peer
tls: handshake timeout
```

**Diagnosis:**
```bash
# Test basic connectivity
telnet localhost 8080

# Check if service is listening
netstat -tlnp | grep 8080

# Test TLS handshake manually
echo | openssl s_client -connect localhost:8080 -servername echo-server
```

**Solutions:**
```bash
# Increase timeout in client code
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

# Check firewall rules
sudo iptables -L | grep 8080
sudo ufw status

# Verify certificates are valid
./bin/ephemos health --config config/echo-server.yaml --verbose
```

### Problem: Certificate verification failed during handshake

**Symptoms:**
```
tls: failed to verify certificate: x509: certificate is valid for spiffe://example.org/wrong-service
remote error: tls: bad certificate
```

**Solutions:**
```bash
# Verify service identities match
docker exec spire-agent /opt/spire/bin/spire-agent api fetch x509 -write /tmp/
openssl x509 -in /tmp/svid.0.pem -noout -ext subjectAltName

# Check service configuration
cat config/echo-server.yaml
# Verify service.name matches SPIFFE ID

# Re-register with correct SPIFFE ID
./bin/ephemos register --config config/echo-server.yaml --force
```

## Authorization Problems

### Problem: "permission denied" after successful mTLS

**Symptoms:**
```
rpc error: code = PermissionDenied desc = client spiffe://example.org/echo-client not authorized
```

**Diagnosis:**
```bash
# Check server authorization configuration
cat config/echo-server.yaml
# Look at authorized_clients section

# Check actual client identity
# Look in server logs for the connecting client SPIFFE ID
```

**Solutions:**
```bash
# Update server config to include client
# config/echo-server.yaml
authorized_clients:
  - spiffe://example.org/echo-client
  - spiffe://example.org/other-client

# Or remove authorization check temporarily for testing
# Comment out authorization logic in server code
```

### Problem: Client connecting with unexpected identity

**Symptoms:**
```
Server log: "Unauthorized client: spiffe://example.org/unexpected-service"
```

**Solutions:**
```bash
# Check what identity the client is actually using
# Add logging to client:
identity, err := client.GetIdentity(context.Background())
log.Printf("Client identity: %s", identity.URI)

# Verify client's SPIRE registration
docker exec spire-server /opt/spire/bin/spire-server entry show -spiffeID spiffe://example.org/echo-client

# Check client configuration
cat config/echo-client.yaml
# Verify service.name matches expected SPIFFE ID
```

## Performance Issues

### Problem: Slow mTLS handshakes

**Symptoms:**
- High latency on first requests
- Timeout errors under load

**Diagnosis:**
```bash
# Profile handshake time
time echo | openssl s_client -connect localhost:8080 -servername echo-server

# Check certificate size
docker exec spire-agent /opt/spire/bin/spire-agent api fetch x509 -write /tmp/
ls -la /tmp/svid.0.pem /tmp/bundle.0.pem
```

**Solutions:**
```bash
# Enable connection reuse in client
# Use grpc.WithKeepaliveParams()

# Reduce certificate chain length
# Check SPIRE server configuration for intermediate CAs

# Pre-warm connections
# Implement connection pooling
```

### Problem: High memory usage

**Symptoms:**
- Services consuming excessive memory
- Out of memory errors

**Solutions:**
```go
// Implement certificate caching with TTL
type CertCache struct {
    mu    sync.RWMutex
    certs map[string]*CachedCert
}

// Set appropriate GOMAXPROCS
runtime.GOMAXPROCS(runtime.NumCPU())

// Monitor certificate rotation frequency
// Reduce rotation if too frequent
```

## Configuration Errors

### Problem: "configuration validation failed"

**Symptoms:**
```
invalid service configuration: service name is required
invalid SPIFFE configuration: socket path must be absolute
```

**Solutions:**
```yaml
# Fix common configuration mistakes
service:
  name: "my-service"        # Must not be empty
  domain: "example.org"     # Valid domain format

spiffe:
  socket_path: "/tmp/spire-agent/public/api.sock"  # Must be absolute path

authorized_clients:
  - "spiffe://example.org/client"  # Must start with spiffe://

# Validate configuration
./bin/ephemos validate --config config.yaml
```

### Problem: Environment variable substitution not working

**Symptoms:**
Configuration values not being replaced from environment

**Solutions:**
```bash
# Check environment variables are set
env | grep EPHEMOS

# Use correct syntax in config
service:
  name: "${SERVICE_NAME:-default-service}"

# Export variables before running
export SERVICE_NAME=my-service
./my-service
```

## Development Environment

### Problem: Changes not taking effect

**Solutions:**
```bash
# Rebuild binaries after code changes
make clean && make all

# Restart SPIRE infrastructure
cd scripts/demo && ./cleanup.sh && ./start-spire.sh

# Clear Go module cache if needed
go clean -modcache

# Re-register services after config changes
./bin/ephemos register --config config/echo-server.yaml --force
```

### Problem: Tests failing in CI/CD

**Common Issues:**
```bash
# Port conflicts in parallel tests
# Use random ports or test isolation

# SPIRE infrastructure not available
# Use mocked SPIFFE provider for unit tests

# Time synchronization issues
# Use relative time comparisons in tests

# Resource cleanup
# Implement proper test teardown
```

## Production Issues

### Problem: Services failing after SPIRE restart

**Symptoms:**
```
All services stop working after SPIRE maintenance
```

**Solutions:**
```go
// Implement retry logic with exponential backoff
func connectWithRetry(ctx context.Context) error {
    backoff := 1 * time.Second
    maxBackoff := 30 * time.Second
    
    for {
        err := connect()
        if err == nil {
            return nil
        }
        
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(backoff):
            backoff = min(backoff*2, maxBackoff)
        }
    }
}

// Monitor SPIRE agent connection health
go func() {
    ticker := time.NewTicker(30 * time.Second)
    for range ticker.C {
        if err := client.HealthCheck(); err != nil {
            log.Warnf("SPIRE connection unhealthy: %v", err)
            // Trigger reconnection
        }
    }
}()
```

### Problem: Certificate rotation causing brief outages

**Solutions:**
```go
// Pre-fetch certificates before expiry
func (c *Client) rotateCertificates() {
    // Fetch new certificate when current is 50% through lifetime
    expiresIn := cert.NotAfter.Sub(time.Now())
    renewAt := time.Now().Add(expiresIn / 2)
    
    time.AfterFunc(renewAt.Sub(time.Now()), func() {
        newCert, err := c.fetchCertificate()
        if err == nil {
            c.updateCertificate(newCert)
        }
    })
}

// Implement graceful connection draining
func (s *Server) Shutdown(ctx context.Context) error {
    // Stop accepting new connections
    s.listener.Close()
    
    // Wait for existing connections to finish
    done := make(chan struct{})
    go func() {
        s.grpcServer.GracefulStop()
        close(done)
    }()
    
    select {
    case <-done:
        return nil
    case <-ctx.Done():
        s.grpcServer.Stop()
        return ctx.Err()
    }
}
```

## Getting Help

If you're still having issues:

1. **Enable Debug Logging**
   ```yaml
   # Add to configuration
   logging:
     level: debug
     format: json
   ```

2. **Collect Diagnostic Information**
   ```bash
   # Create support bundle
   ./scripts/collect-diagnostics.sh > diagnostics.txt
   ```

3. **Check Known Issues**
   - GitHub Issues: Search for similar problems
   - Release Notes: Check for breaking changes
   - SPIRE Documentation: Verify SPIRE version compatibility

4. **Report Issues**
   - Include complete error messages
   - Provide configuration files (sanitized)
   - Include versions: Go, Ephemos, SPIRE
   - Describe expected vs actual behavior

### Useful Debug Commands

```bash
# Comprehensive health check
./bin/ephemos doctor --config config.yaml

# Network connectivity test
./bin/ephemos test-connection --target localhost:8080

# Certificate inspection
./bin/ephemos inspect-cert --config config.yaml

# SPIRE integration test
./bin/ephemos test-spire --config config.yaml
```

---

Most issues are related to SPIRE configuration, network connectivity, or certificate handling. Start with the health checks and work through the specific problem areas systematically.