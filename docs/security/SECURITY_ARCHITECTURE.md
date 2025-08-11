# Ephemos Security Architecture

This document provides detailed security architecture guidance for implementing and deploying Ephemos in production environments.

## Security Design Principles

### 1. Zero Trust Architecture

```mermaid
graph TB
    subgraph "Zero Trust Model"
        ZT1[Never Trust, Always Verify]
        ZT2[Least Privilege Access]
        ZT3[Assume Breach]
        ZT4[Continuous Verification]
    end
    
    subgraph "Implementation in Ephemos"
        I1[mTLS for All Communication]
        I2[Identity-based Authorization]
        I3[Short-lived Certificates]
        I4[Continuous Identity Validation]
    end
    
    ZT1 --> I1
    ZT2 --> I2
    ZT3 --> I3
    ZT4 --> I4
    
    classDef principle fill:#e3f2fd
    classDef implementation fill:#e8f5e8
    
    class ZT1,ZT2,ZT3,ZT4 principle
    class I1,I2,I3,I4 implementation
```

### 2. Defense in Depth

```mermaid
graph TB
    subgraph "Layer 1: Network Security"
        L1[Network Segmentation]
        L2[Firewall Rules]
        L3[VPC/VNET Isolation]
    end
    
    subgraph "Layer 2: Host Security"
        L4[OS Hardening]
        L5[Container Security]
        L6[Process Isolation]
    end
    
    subgraph "Layer 3: Identity Security"
        L7[SPIFFE Identity]
        L8[Certificate Validation]
        L9[Attestation Policies]
    end
    
    subgraph "Layer 4: Application Security"
        L10[Input Validation]
        L11[Secure Coding]
        L12[Runtime Protection]
    end
    
    subgraph "Layer 5: Data Security"
        L13[Encryption at Rest]
        L14[Encryption in Transit]
        L15[Key Management]
    end
    
    L1 --> L4
    L4 --> L7
    L7 --> L10
    L10 --> L13
    
    classDef network fill:#e1f5fe
    classDef host fill:#f3e5f5
    classDef identity fill:#e8f5e8
    classDef app fill:#fff3e0
    classDef data fill:#fce4ec
    
    class L1,L2,L3 network
    class L4,L5,L6 host
    class L7,L8,L9 identity
    class L10,L11,L12 app
    class L13,L14,L15 data
```

## Identity Security Architecture

### SPIFFE Trust Domain Design

```mermaid
graph TB
    subgraph "Production Trust Domain: prod.company.com"
        subgraph "Production SPIRE"
            PS[SPIRE Server Cluster]
            PDB[(Production Registry)]
        end
        
        subgraph "Production Services"
            PA1[SPIRE Agent]
            PA2[SPIRE Agent] 
            PA3[SPIRE Agent]
            
            PS1[Payment Service]
            PS2[User Service]
            PS3[Order Service]
        end
    end
    
    subgraph "Staging Trust Domain: staging.company.com"
        subgraph "Staging SPIRE"
            SS[SPIRE Server]
            SDB[(Staging Registry)]
        end
        
        subgraph "Staging Services"
            SA1[SPIRE Agent]
            ST1[Test Services]
        end
    end
    
    PS --> PA1
    PS --> PA2
    PS --> PA3
    PS --- PDB
    
    PA1 --- PS1
    PA2 --- PS2
    PA3 --- PS3
    
    SS --> SA1
    SS --- SDB
    SA1 --- ST1
    
    PS1 ---|mTLS| PS2
    PS2 ---|mTLS| PS3
    
    classDef production fill:#e8f5e8
    classDef staging fill:#fff3e0
    classDef spire fill:#e1f5fe
    
    class PS,PDB,PA1,PA2,PA3,PS1,PS2,PS3 production
    class SS,SDB,SA1,ST1 staging
    class PS,SS spire
```

### Identity Lifecycle Security

```mermaid
sequenceDiagram
    participant Admin
    participant SPIRE as SPIRE Server
    participant Agent as SPIRE Agent
    participant App as Application
    participant Monitor as Security Monitor
    
    Note over Admin,Monitor: Secure Identity Provisioning
    
    Admin->>+SPIRE: Create Registration Entry
    Note right of SPIRE: ✅ Admin Authentication
    Note right of SPIRE: ✅ Authorization Check
    Note right of SPIRE: ✅ Policy Validation
    
    App->>+Agent: Request Identity
    Agent->>+SPIRE: Attest Workload
    Note right of SPIRE: ✅ Workload Attestation
    Note right of SPIRE: ✅ Selector Validation
    
    SPIRE->>Agent: Issue SVID
    Agent->>App: Deliver SVID
    Agent->>Monitor: Log Identity Event
    
    Note over Admin,Monitor: Continuous Rotation
    
    loop Every 30 minutes
        Agent->>SPIRE: Rotate SVID
        SPIRE->>Agent: New SVID
        Agent->>App: Update Identity
        Agent->>Monitor: Log Rotation
    end
    
    Note over Admin,Monitor: Emergency Revocation
    
    Monitor->>Admin: Security Alert
    Admin->>SPIRE: Revoke Identity
    SPIRE->>Agent: Revocation Notice
    Agent->>App: Identity Revoked
    Agent->>Monitor: Log Revocation
```

## Deployment Security Patterns

### 1. High Availability SPIRE Deployment

```mermaid
graph TB
    subgraph "Load Balancer Layer"
        LB[Load Balancer]
        LB2[Backup Load Balancer]
    end
    
    subgraph "SPIRE Server Cluster"
        SS1[SPIRE Server 1]
        SS2[SPIRE Server 2]
        SS3[SPIRE Server 3]
    end
    
    subgraph "Database Layer"
        DB1[(Primary DB)]
        DB2[(Replica DB)]
        DB3[(Replica DB)]
    end
    
    subgraph "Compute Nodes"
        subgraph "Node 1"
            SA1[SPIRE Agent 1]
            W1[Workloads]
        end
        
        subgraph "Node 2"
            SA2[SPIRE Agent 2]
            W2[Workloads]
        end
        
        subgraph "Node 3"
            SA3[SPIRE Agent 3]
            W3[Workloads]
        end
    end
    
    LB --> SS1
    LB --> SS2
    LB --> SS3
    
    LB2 -.-> SS1
    LB2 -.-> SS2
    LB2 -.-> SS3
    
    SS1 --> DB1
    SS2 --> DB1
    SS3 --> DB1
    
    DB1 --> DB2
    DB1 --> DB3
    
    SA1 --> LB
    SA2 --> LB
    SA3 --> LB
    
    SA1 --- W1
    SA2 --- W2
    SA3 --- W3
    
    classDef lb fill:#e1f5fe
    classDef spire fill:#e8f5e8
    classDef db fill:#fce4ec
    classDef node fill:#fff3e0
    
    class LB,LB2 lb
    class SS1,SS2,SS3,SA1,SA2,SA3 spire
    class DB1,DB2,DB3 db
    class W1,W2,W3 node
```

### 2. Network Security Zones

```mermaid
graph TB
    subgraph "DMZ Zone"
        DMZ1[Web Gateway]
        DMZ2[API Gateway]
    end
    
    subgraph "Application Zone"
        APP1[Frontend Services]
        APP2[Business Logic]
        APP3[API Services]
    end
    
    subgraph "Data Zone"
        DATA1[Database Services]
        DATA2[Cache Layer]
        DATA3[Message Queue]
    end
    
    subgraph "Management Zone"
        MGMT1[SPIRE Server]
        MGMT2[Monitoring]
        MGMT3[Logging]
    end
    
    subgraph "Security Controls"
        FW1[Firewall]
        IDS[Intrusion Detection]
        WAF[Web Application Firewall]
    end
    
    Internet ---|HTTPS| WAF
    WAF --> DMZ1
    WAF --> DMZ2
    
    DMZ1 ---|mTLS| FW1
    DMZ2 ---|mTLS| FW1
    
    FW1 --> APP1
    FW1 --> APP2
    FW1 --> APP3
    
    APP1 ---|mTLS| DATA1
    APP2 ---|mTLS| DATA2
    APP3 ---|mTLS| DATA3
    
    MGMT1 --> APP1
    MGMT1 --> APP2
    MGMT1 --> APP3
    MGMT1 --> DATA1
    MGMT1 --> DATA2
    MGMT1 --> DATA3
    
    IDS --> MGMT2
    
    classDef dmz fill:#ffebee
    classDef app fill:#e8f5e8
    classDef data fill:#e1f5fe
    classDef mgmt fill:#fff3e0
    classDef security fill:#f3e5f5
    
    class DMZ1,DMZ2 dmz
    class APP1,APP2,APP3 app
    class DATA1,DATA2,DATA3 data
    class MGMT1,MGMT2,MGMT3 mgmt
    class FW1,IDS,WAF security
```

## Security Configuration Hardening

### SPIRE Server Security Configuration

```yaml
# Secure SPIRE Server configuration
server:
  bind_address: "127.0.0.1"
  bind_port: "8081"
  socket_path: "/tmp/spire-server/private/api.sock"
  trust_domain: "production.company.com"
  data_dir: "/opt/spire/data"
  log_level: "WARN"
  log_format: "json"
  
  # Enable audit logging
  audit:
    enabled: true
    log_file: "/var/log/spire/audit.log"
    max_size: 100  # MB
    max_backups: 10
    
  # TLS configuration
  tls:
    cert_file: "/opt/spire/certs/server.crt"
    key_file: "/opt/spire/certs/server.key"
    ca_cert_file: "/opt/spire/certs/ca.crt"
    min_version: "TLS1.2"
    cipher_suites: [
      "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
      "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
      "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
    ]

plugins:
  DataStore:
    sql:
      plugin_data:
        database_type: "postgres"
        connection_string: "postgresql://spire:$PASSWORD@localhost/spire?sslmode=require"
        max_open_conns: 20
        max_idle_conns: 10
        conn_max_lifetime: "1h"
        
  KeyManager:
    disk:
      plugin_data:
        keys_path: "/opt/spire/data/keys"
        # In production, use HSM:
        # aws_kms:
        #   plugin_data:
        #     region: "us-west-2"
        #     key_metadata_file: "/opt/spire/aws_kms_keys.json"
        
  NodeAttestor:
    join_token:
      plugin_data:
        trust_domain: "production.company.com"
        
  UpstreamAuthority:
    disk:
      plugin_data:
        cert_file_path: "/opt/spire/certs/upstream_ca.crt"
        key_file_path: "/opt/spire/certs/upstream_ca.key"
```

### SPIRE Agent Security Configuration

```yaml
# Secure SPIRE Agent configuration
agent:
  data_dir: "/opt/spire/data"
  log_level: "WARN"
  log_format: "json"
  server_address: "spire-server.internal"
  server_port: "8081"
  socket_path: "/tmp/spire-agent/public/api.sock"
  trust_domain: "production.company.com"
  trust_bundle_path: "/opt/spire/conf/bootstrap.crt"
  
  # Workload API security
  workload_api:
    socket_path: "/tmp/spire-agent/public/api.sock"
    # Restrict socket permissions
    socket_mode: "0600"
    socket_uid: 1000  # spire user
    socket_gid: 1000  # spire group

plugins:
  NodeAttestor:
    join_token:
      plugin_data:
        trust_domain: "production.company.com"
        
  KeyManager:
    memory:
      plugin_data: {}
      
  WorkloadAttestor:
    unix:
      plugin_data:
        discover_workload_path: true
    k8s:
      plugin_data:
        kubelet_read_only_port: 10255
        max_poll_attempts: 5
        poll_retry_interval: "5s"
```

## Runtime Security Monitoring

### Security Event Schema

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "event_type": "identity_provisioned",
  "severity": "INFO",
  "source": "spire-server",
  "trust_domain": "production.company.com",
  "spiffe_id": "spiffe://production.company.com/workload/payment-service",
  "details": {
    "workload_pid": 12345,
    "workload_uid": 1000,
    "attestation_type": "unix",
    "certificate_serial": "1a2b3c4d5e6f",
    "expiry_time": "2024-01-15T11:30:00Z"
  },
  "metadata": {
    "node_id": "node-1",
    "agent_version": "1.8.0",
    "server_version": "1.8.0"
  }
}
```

### Alert Rules Configuration

```yaml
# Security monitoring alert rules
alerts:
  - name: "Failed Workload Attestation"
    query: "event_type:attestation_failed"
    threshold: 5
    window: "5m"
    severity: "HIGH"
    actions:
      - "notify_security_team"
      - "quarantine_workload"
      
  - name: "Certificate Validation Failure"
    query: "event_type:cert_validation_failed"
    threshold: 10
    window: "1m" 
    severity: "MEDIUM"
    actions:
      - "alert"
      - "log_investigation"
      
  - name: "Unusual Identity Requests"
    query: "event_type:identity_requested AND source:unknown"
    threshold: 1
    window: "1m"
    severity: "HIGH"
    actions:
      - "block_request"
      - "notify_security_team"
      
  - name: "Mass Certificate Revocation"
    query: "event_type:certificate_revoked"
    threshold: 100
    window: "5m"
    severity: "CRITICAL"
    actions:
      - "emergency_alert"
      - "investigate_potential_breach"
```

## Incident Response Procedures

### Security Incident Classification

```mermaid
graph TB
    subgraph "Incident Types"
        IT1[Identity Compromise]
        IT2[Certificate Authority Breach]
        IT3[Mass Service Disruption] 
        IT4[Unauthorized Access]
        IT5[Data Exfiltration]
    end
    
    subgraph "Response Severity"
        RS1[P0 - Critical]
        RS2[P1 - High]
        RS3[P2 - Medium]
        RS4[P3 - Low]
    end
    
    subgraph "Response Actions"
        RA1[Immediate Revocation]
        RA2[Service Isolation]
        RA3[Forensic Analysis]
        RA4[Communication Plan]
        RA5[Recovery Procedures]
    end
    
    IT2 --> RS1
    IT1 --> RS2
    IT3 --> RS2
    IT4 --> RS3
    IT5 --> RS1
    
    RS1 --> RA1
    RS1 --> RA2
    RS2 --> RA1
    RS3 --> RA3
    RS1 --> RA4
    RS1 --> RA5
    
    classDef incident fill:#ffebee
    classDef severity fill:#fff3e0
    classDef action fill:#e8f5e8
    
    class IT1,IT2,IT3,IT4,IT5 incident
    class RS1,RS2,RS3,RS4 severity
    class RA1,RA2,RA3,RA4,RA5 action
```

### Automated Response Playbook

```yaml
# Automated incident response playbook
playbooks:
  identity_compromise:
    triggers:
      - "failed_attestation_threshold_exceeded"
      - "certificate_misuse_detected"
    
    actions:
      - step: "isolate_workload"
        timeout: "30s"
        
      - step: "revoke_identity"
        timeout: "60s"
        
      - step: "notify_security_team"
        timeout: "5s"
        
      - step: "collect_forensics"
        timeout: "300s"
        
      - step: "generate_incident_report"
        timeout: "60s"
        
  certificate_authority_breach:
    triggers:
      - "ca_key_access_unauthorized"
      - "mass_certificate_issuance"
      
    actions:
      - step: "emergency_ca_rotation"
        timeout: "300s"
        
      - step: "revoke_all_certificates"  
        timeout: "600s"
        
      - step: "activate_backup_ca"
        timeout: "120s"
        
      - step: "notify_executive_team"
        timeout: "5s"
```

## Compliance and Auditing

### Audit Trail Requirements

```mermaid
graph TB
    subgraph "Audit Events"
        AE1[Identity Creation]
        AE2[Certificate Issuance]
        AE3[Authentication Events]
        AE4[Authorization Decisions]
        AE5[Configuration Changes]
        AE6[Administrative Actions]
    end
    
    subgraph "Audit Storage"
        AS1[Immutable Logs]
        AS2[Encrypted Storage]
        AS3[Off-site Backup]
        AS4[Retention Policy]
    end
    
    subgraph "Compliance Frameworks"
        CF1[SOC 2 Type II]
        CF2[ISO 27001]
        CF3[PCI DSS]
        CF4[GDPR]
        CF5[HIPAA]
    end
    
    AE1 --> AS1
    AE2 --> AS1
    AE3 --> AS1
    AE4 --> AS1
    AE5 --> AS1
    AE6 --> AS1
    
    AS1 --> AS2
    AS2 --> AS3
    AS3 --> AS4
    
    AS4 --> CF1
    AS4 --> CF2
    AS4 --> CF3
    AS4 --> CF4
    AS4 --> CF5
    
    classDef event fill:#e3f2fd
    classDef storage fill:#e8f5e8
    classDef compliance fill:#fff3e0
    
    class AE1,AE2,AE3,AE4,AE5,AE6 event
    class AS1,AS2,AS3,AS4 storage
    class CF1,CF2,CF3,CF4,CF5 compliance
```

### Supply Chain Security

Ephemos includes comprehensive Software Bill of Materials (SBOM) generation to support supply chain security and compliance requirements.

```mermaid
graph TB
    subgraph "SBOM Generation"
        SG1[Syft Scanner]
        SG2[Dependency Analysis]
        SG3[License Detection]
        SG4[Vulnerability Correlation]
    end
    
    subgraph "SBOM Formats"
        SF1[SPDX 2.3 JSON]
        SF2[CycloneDX JSON]
        SF3[Human Readable Summary]
        SF4[Integrity Checksums]
    end
    
    subgraph "Security Integration"
        SI1[OSV Scanner]
        SI2[Grype Scanner]
        SI3[Trivy Scanner]
        SI4[CI/CD Validation]
    end
    
    subgraph "Compliance Support"
        CS1[NTIA Minimum Elements]
        CS2[Executive Order 14028]
        CS3[ISO/IEC 5962 SPDX]
        CS4[NIST SP 800-161]
    end
    
    SG1 --> SF1
    SG2 --> SF2
    SG3 --> SF3
    SG4 --> SF4
    
    SF1 --> SI1
    SF2 --> SI2
    SF1 --> SI3
    SF4 --> SI4
    
    SI1 --> CS1
    SI2 --> CS2
    SI3 --> CS3
    SI4 --> CS4
    
    classDef generation fill:#e3f2fd
    classDef format fill:#e8f5e8
    classDef security fill:#ffebee
    classDef compliance fill:#fff3e0
    
    class SG1,SG2,SG3,SG4 generation
    class SF1,SF2,SF3,SF4 format
    class SI1,SI2,SI3,SI4 security
    class CS1,CS2,CS3,CS4 compliance
```

**SBOM Capabilities:**
- **Automated Generation**: SBOM files generated in CI/CD pipeline
- **Multiple Formats**: SPDX and CycloneDX for tool compatibility
- **Vulnerability Scanning**: Integration with security scanners
- **Compliance Ready**: Supports regulatory requirements
- **Integrity Verification**: Checksums for tamper detection

### Compliance Checklist

- ✅ **Access Controls**: Role-based access to SPIRE administration
- ✅ **Audit Logging**: Comprehensive audit trail for all identity operations
- ✅ **Encryption**: End-to-end encryption for all communications
- ✅ **Key Management**: Secure key storage and rotation procedures
- ✅ **Incident Response**: Documented procedures and automated responses
- ✅ **Regular Assessment**: Quarterly security reviews and penetration testing
- ✅ **Data Protection**: Privacy controls and data minimization
- ✅ **Backup and Recovery**: Disaster recovery procedures and testing
- ✅ **Supply Chain Security**: SBOM generation and vulnerability scanning
- ✅ **Dependency Tracking**: Complete software bill of materials
- ✅ **License Compliance**: Automated license verification and reporting

## Related Security Documentation

- [SBOM Generation Guide](SBOM_GENERATION.md) - Comprehensive SBOM procedures
- [CI/CD Security](CI_CD_SECURITY.md) - Pipeline security configuration
- [Threat Model](THREAT_MODEL.md) - Security threat analysis
- [Security Runbook](SECURITY_RUNBOOK.md) - Operational procedures
- [Configuration Security](CONFIGURATION_SECURITY.md) - Secure configuration

---

*This security architecture should be reviewed and updated regularly to address evolving threats and compliance requirements.*