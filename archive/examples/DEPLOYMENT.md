# Ephemos Deployment Guide

This document provides comprehensive guidance for deploying Ephemos-based services in production environments.

## Table of Contents
- [Production Checklist](#production-checklist)
- [SPIRE Infrastructure Setup](#spire-infrastructure-setup)
- [Container Deployment](#container-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [High Availability](#high-availability)
- [Monitoring and Observability](#monitoring-and-observability)
- [Security Hardening](#security-hardening)
- [Performance Tuning](#performance-tuning)

## Production Checklist

### Pre-Deployment Requirements

- [ ] **SPIRE Infrastructure**: SPIRE server and agents deployed and operational
- [ ] **Trust Domain**: Production trust domain configured (not `example.org`)
- [ ] **Certificate Authority**: Proper CA hierarchy established
- [ ] **Node Attestation**: Production-grade node attestation configured
- [ ] **Network Security**: Firewall rules and network policies in place
- [ ] **Monitoring**: Observability stack deployed (metrics, logs, traces)
- [ ] **Backup Strategy**: SPIRE server data backup procedures established

### Service-Specific Checklist

- [ ] **Configuration**: Production configurations without test/debug settings
- [ ] **Authorization**: Proper authorization policies implemented
- [ ] **Resource Limits**: CPU/memory limits configured appropriately
- [ ] **Health Checks**: Liveness and readiness probes configured
- [ ] **Graceful Shutdown**: Signal handling for clean service termination
- [ ] **Secret Management**: No hardcoded secrets or test credentials

## SPIRE Infrastructure Setup

### SPIRE Server Configuration

```yaml
# /etc/spire/server.conf
server {
    bind_address = "0.0.0.0"
    bind_port = "8081"
    trust_domain = "production.company.com"
    data_dir = "/opt/spire/data/server"
    log_level = "INFO"
    
    ca_subject = {
        country = ["US"]
        organization = ["Company Inc"]
        common_name = "SPIRE Server CA"
    }
    
    # Production-grade CA configuration
    ca_ttl = "168h"  # 7 days
    default_svid_ttl = "1h"
}

plugins {
    DataStore "sql" {
        plugin_data {
            database_type = "postgres"
            connection_string = "postgresql://spire:password@postgres:5432/spire"
        }
    }
    
    KeyManager "memory" {
        plugin_data = {}
    }
    
    NodeAttestor "k8s_sat" {
        plugin_data {
            clusters = {
                "production" = {
                    service_account_allow_list = ["spire:spire-agent"]
                }
            }
        }
    }
    
    Notifier "k8sbundle" {
        plugin_data {
            namespace = "spire-system"
            config_map = "spire-bundle"
        }
    }
}
```

### SPIRE Agent Configuration

```yaml
# /etc/spire/agent.conf
agent {
    data_dir = "/opt/spire/data/agent"
    log_level = "INFO"
    server_address = "spire-server.spire-system.svc.cluster.local"
    server_port = "8081"
    socket_path = "/tmp/spire-agent/public/api.sock"
    trust_domain = "production.company.com"
    trust_bundle_path = "/opt/spire/conf/bootstrap.crt"
}

plugins {
    NodeAttestor "k8s_sat" {
        plugin_data {
            cluster = "production"
        }
    }
    
    KeyManager "memory" {
        plugin_data = {}
    }
    
    WorkloadAttestor "k8s" {
        plugin_data {
            skip_kubelet_verification = true
        }
    }
}
```

## Container Deployment

### Dockerfile Best Practices

```dockerfile
# Use distroless base image for minimal attack surface
FROM gcr.io/distroless/base-debian11:latest

# Copy binary built with CGO_ENABLED=0
COPY echo-server /usr/local/bin/echo-server

# Non-root user
USER 65534:65534

# Expose service port
EXPOSE 8080

# Health check endpoint
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/usr/local/bin/echo-server", "health"]

ENTRYPOINT ["/usr/local/bin/echo-server"]
```

### Docker Compose Example

```yaml
version: '3.8'

services:
  spire-server:
    image: gcr.io/spiffe-io/spire-server:1.8.0
    command: ["/opt/spire/bin/spire-server", "run"]
    volumes:
      - ./spire-server.conf:/opt/spire/conf/server/server.conf:ro
      - spire-server-data:/opt/spire/data/server
    networks:
      - spire
    ports:
      - "8081:8081"
    
  spire-agent:
    image: gcr.io/spiffe-io/spire-agent:1.8.0
    command: ["/opt/spire/bin/spire-agent", "run"]
    depends_on:
      - spire-server
    volumes:
      - ./spire-agent.conf:/opt/spire/conf/agent/agent.conf:ro
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - spire-agent-socket:/tmp/spire-agent/public
    networks:
      - spire
      
  echo-server:
    build: .
    command: ["serve"]
    depends_on:
      - spire-agent
    volumes:
      - spire-agent-socket:/var/run/spire/sockets:ro
      - ./config/echo-server.yaml:/etc/ephemos/config.yaml:ro
    ports:
      - "8080:8080"
    networks:
      - spire
      - app
    environment:
      - EPHEMOS_CONFIG_PATH=/etc/ephemos/config.yaml

volumes:
  spire-server-data:
  spire-agent-socket:

networks:
  spire:
  app:
```

## Kubernetes Deployment

### SPIRE Server Deployment

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: spire-server
  namespace: spire-system
spec:
  serviceName: spire-server
  replicas: 3  # HA deployment
  selector:
    matchLabels:
      app: spire-server
  template:
    metadata:
      labels:
        app: spire-server
    spec:
      serviceAccountName: spire-server
      containers:
        - name: spire-server
          image: gcr.io/spiffe-io/spire-server:1.8.0
          args:
            - -config
            - /run/spire/config/server.conf
          ports:
            - containerPort: 8081
              name: grpc
          volumeMounts:
            - name: spire-config
              mountPath: /run/spire/config
              readOnly: true
            - name: spire-server-socket
              mountPath: /tmp/spire-server/private
            - name: spire-data
              mountPath: /run/spire/data
          livenessProbe:
            httpGet:
              path: /live
              port: 8080
            initialDelaySeconds: 30
            periodSeconds: 30
          readinessProbe:
            httpGet:
              path: /ready
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 5
          resources:
            requests:
              cpu: 100m
              memory: 256Mi
            limits:
              cpu: 500m
              memory: 1Gi
      volumes:
        - name: spire-config
          configMap:
            name: spire-server
        - name: spire-server-socket
          emptyDir: {}
  volumeClaimTemplates:
    - metadata:
        name: spire-data
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 10Gi
```

### Application Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: echo-server
  namespace: ephemos-demo
spec:
  replicas: 3
  selector:
    matchLabels:
      app: echo-server
  template:
    metadata:
      labels:
        app: echo-server
      annotations:
        # SPIRE workload registration
        spiffe.io/spiffe-id: spiffe://production.company.com/echo-server
    spec:
      serviceAccountName: echo-server
      containers:
        - name: echo-server
          image: echo-server:v1.0.0
          ports:
            - containerPort: 8080
              name: grpc
            - containerPort: 9090
              name: metrics
          env:
            - name: EPHEMOS_CONFIG_PATH
              value: /etc/ephemos/config.yaml
            - name: SPIFFE_ENDPOINT_SOCKET
              value: unix:///var/run/spire/sockets/agent.sock
          volumeMounts:
            - name: ephemos-config
              mountPath: /etc/ephemos
              readOnly: true
            - name: spire-agent-socket
              mountPath: /var/run/spire/sockets
              readOnly: true
          livenessProbe:
            grpc:
              port: 8080
            initialDelaySeconds: 30
            periodSeconds: 30
          readinessProbe:
            grpc:
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 5
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 256Mi
          securityContext:
            runAsNonRoot: true
            runAsUser: 65534
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
            capabilities:
              drop: ["ALL"]
      volumes:
        - name: ephemos-config
          configMap:
            name: echo-server-config
        - name: spire-agent-socket
          hostPath:
            path: /run/spire/sockets
            type: DirectoryOrCreate
      nodeSelector:
        kubernetes.io/os: linux
      tolerations:
        - effect: NoSchedule
          operator: Exists
```

### Service and NetworkPolicy

```yaml
apiVersion: v1
kind: Service
metadata:
  name: echo-server
  namespace: ephemos-demo
spec:
  selector:
    app: echo-server
  ports:
    - name: grpc
      port: 8080
      targetPort: 8080
    - name: metrics
      port: 9090
      targetPort: 9090

---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: echo-server-netpol
  namespace: ephemos-demo
spec:
  podSelector:
    matchLabels:
      app: echo-server
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ephemos-demo
        - podSelector:
            matchLabels:
              app: echo-client
      ports:
        - protocol: TCP
          port: 8080
  egress:
    - to: []  # SPIRE agent communication
      ports:
        - protocol: TCP
          port: 8081
```

## High Availability

### SPIRE Server HA

```yaml
# StatefulSet with 3 replicas behind a load balancer
apiVersion: v1
kind: Service
metadata:
  name: spire-server
  namespace: spire-system
spec:
  clusterIP: None  # Headless service
  selector:
    app: spire-server
  ports:
    - name: grpc
      port: 8081

---
# External load balancer configuration
apiVersion: v1
kind: Service
metadata:
  name: spire-server-lb
  namespace: spire-system
spec:
  type: LoadBalancer
  selector:
    app: spire-server
  ports:
    - port: 8081
      targetPort: 8081
```

### Database High Availability

```yaml
# PostgreSQL cluster for SPIRE server data store
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: spire-postgres
  namespace: spire-system
spec:
  instances: 3
  postgresql:
    parameters:
      max_connections: "200"
      shared_preload_libraries: "pg_stat_statements"
  bootstrap:
    initdb:
      database: spire
      owner: spire
      secret:
        name: spire-postgres-credentials
  storage:
    size: 50Gi
    storageClass: fast-ssd
  monitoring:
    enabled: true
```

### Application HA

```yaml
# Pod Disruption Budget
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: echo-server-pdb
  namespace: ephemos-demo
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: echo-server

---
# Horizontal Pod Autoscaler
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: echo-server-hpa
  namespace: ephemos-demo
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: echo-server
  minReplicas: 3
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

## Monitoring and Observability

### Prometheus Monitoring

```yaml
# ServiceMonitor for Prometheus scraping
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: echo-server-metrics
  namespace: ephemos-demo
spec:
  selector:
    matchLabels:
      app: echo-server
  endpoints:
    - port: metrics
      interval: 30s
      path: /metrics

---
# SPIRE server monitoring
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: spire-server-metrics
  namespace: spire-system
spec:
  selector:
    matchLabels:
      app: spire-server
  endpoints:
    - port: http-monitoring
      interval: 30s
      path: /metrics
```

### Grafana Dashboard

```json
{
  "dashboard": {
    "title": "Ephemos Services",
    "panels": [
      {
        "title": "Certificate Expiry",
        "type": "stat",
        "targets": [
          {
            "expr": "ephemos_certificate_expiry_seconds"
          }
        ]
      },
      {
        "title": "mTLS Handshake Duration",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, ephemos_tls_handshake_duration_seconds_bucket)"
          }
        ]
      },
      {
        "title": "SPIRE Agent Connection Status",
        "type": "stat",
        "targets": [
          {
            "expr": "ephemos_spire_agent_connection_status"
          }
        ]
      }
    ]
  }
}
```

### Alerting Rules

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: ephemos-alerts
  namespace: ephemos-demo
spec:
  groups:
    - name: ephemos
      rules:
        - alert: CertificateExpiringSoon
          expr: ephemos_certificate_expiry_seconds < 300
          for: 1m
          labels:
            severity: warning
          annotations:
            summary: "Certificate expiring soon"
            description: "Certificate for {{ $labels.service }} expires in {{ $value }} seconds"
        
        - alert: SPIREAgentDown
          expr: ephemos_spire_agent_connection_status == 0
          for: 5m
          labels:
            severity: critical
          annotations:
            summary: "SPIRE agent connection lost"
            description: "Service {{ $labels.service }} cannot connect to SPIRE agent"
        
        - alert: HighmTLSHandshakeLatency
          expr: histogram_quantile(0.95, ephemos_tls_handshake_duration_seconds_bucket) > 1
          for: 10m
          labels:
            severity: warning
          annotations:
            summary: "High mTLS handshake latency"
            description: "95th percentile handshake latency is {{ $value }} seconds"
```

## Security Hardening

### Container Security

```dockerfile
# Security-hardened Dockerfile
FROM gcr.io/distroless/static-debian11:nonroot

# Copy statically-linked binary
COPY --from=builder /build/echo-server /usr/local/bin/echo-server

# Run as non-root user
USER 65534:65534

# Read-only root filesystem
# Temporary directories mounted as tmpfs

ENTRYPOINT ["/usr/local/bin/echo-server"]
```

### Pod Security Standards

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: echo-server
spec:
  securityContext:
    # Pod-level security context
    runAsNonRoot: true
    runAsUser: 65534
    runAsGroup: 65534
    fsGroup: 65534
    seccompProfile:
      type: RuntimeDefault
  containers:
    - name: echo-server
      securityContext:
        # Container-level security context
        allowPrivilegeEscalation: false
        readOnlyRootFilesystem: true
        capabilities:
          drop: ["ALL"]
      volumeMounts:
        - name: tmp
          mountPath: /tmp
        - name: var-run
          mountPath: /var/run
  volumes:
    - name: tmp
      emptyDir: {}
    - name: var-run
      emptyDir: {}
```

### Network Security

```yaml
# Istio ServiceEntry for external SPIRE server
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: external-spire-server
spec:
  hosts:
    - spire-server.external.com
  ports:
    - number: 8081
      name: grpc
      protocol: GRPC
  location: MESH_EXTERNAL

---
# Istio VirtualService for traffic management
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: echo-server
spec:
  hosts:
    - echo-server
  http:
    - match:
        - headers:
            spiffe-id:
              regex: "^spiffe://production\\.company\\.com/.*"
      route:
        - destination:
            host: echo-server
```

## Performance Tuning

### Resource Optimization

```yaml
# Optimized resource limits
resources:
  requests:
    cpu: 100m      # Baseline CPU requirement
    memory: 64Mi   # Minimal memory for Go service
  limits:
    cpu: 500m      # Burst capacity
    memory: 256Mi  # Maximum memory usage

# Node affinity for performance-critical services
affinity:
  nodeAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        preference:
          matchExpressions:
            - key: node-type
              operator: In
              values: ["compute-optimized"]
```

### Connection Tuning

```yaml
# Environment variables for Go runtime tuning
env:
  - name: GOMAXPROCS
    valueFrom:
      resourceFieldRef:
        resource: limits.cpu
  - name: GOMEMLIMIT
    valueFrom:
      resourceFieldRef:
        resource: limits.memory

# gRPC keepalive settings
  - name: GRPC_KEEPALIVE_TIME
    value: "30s"
  - name: GRPC_KEEPALIVE_TIMEOUT
    value: "5s"
  - name: GRPC_MAX_CONNECTION_IDLE
    value: "60s"
```

### SPIRE Performance

```yaml
# SPIRE agent configuration for high throughput
agent {
    # Increase cache size for high-frequency certificate requests
    experimental {
        cache_reloaded_svids = true
    }
    
    # Optimize socket permissions and location
    socket_path = "/var/run/spire/sockets/agent.sock"
    
    # Increase connection limits
    server_address = "spire-server.spire-system.svc.cluster.local"
    server_port = "8081"
    
    # Tune log levels for production
    log_level = "WARN"
}
```

---

This deployment guide provides production-ready configurations and best practices for deploying Ephemos-based services at scale with security, reliability, and performance in mind.