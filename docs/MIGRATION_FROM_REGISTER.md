# Migration from `ephemos register` Command

The `ephemos register` command has been removed to keep the core Ephemos library focused on identity-based authentication. Service registration is now handled directly using official SPIRE tooling.

## Quick Migration

### Before (Ephemos CLI)
```bash
ephemos register --name my-service --domain company.com --selector unix:uid:1000
```

### After (SPIRE CLI)
```bash
spire-server entry create \
  -spiffeID spiffe://company.com/my-service \
  -parentID spiffe://company.com/spire-agent \
  -selector unix:uid:1000
```

## Why This Change?

- **Separation of Concerns**: Registration is a control-plane operation that belongs in SPIRE tooling
- **Reduced Dependencies**: Keeps Ephemos focused on authentication, not infrastructure management  
- **Official Support**: Uses SPIRE's native tooling with full feature support and documentation
- **Better Integration**: Works seamlessly with existing SPIRE workflows and automation

## Command Mapping

| Old Ephemos Command | New SPIRE Command |
|---|---|
| `ephemos register --name SERVICE --domain DOMAIN` | `spire-server entry create -spiffeID spiffe://DOMAIN/SERVICE -parentID spiffe://DOMAIN/spire-agent -selector unix:uid:$(id -u)` |
| `ephemos register --config service.yaml` | Extract values from YAML and use `spire-server entry create` with those values |
| `--selector unix:uid:1000` | `-selector unix:uid:1000` |
| `--selector k8s:ns:production` | `-selector k8s:ns:production` |

## Advanced Registration

For production environments, consider:

1. **SPIRE Server API**: Use the gRPC API for programmatic registration
2. **Kubernetes Operator**: Use SPIFFE CSI driver or operators for automated registration
3. **CI/CD Integration**: Add `spire-server entry create` to your deployment pipelines
4. **Configuration Management**: Use tools like Ansible/Terraform with SPIRE provider

## Getting Help

- **SPIRE Documentation**: https://spiffe.io/docs/latest/deploying/spire_server/
- **SPIRE Registration Guide**: https://spiffe.io/docs/latest/deploying/registering/
- **SPIRE CLI Reference**: Run `spire-server entry create --help`

## What Remains in Ephemos

Ephemos still provides:
- `ephemos health` - Check SPIRE infrastructure health
- `ephemos verify` - Identity verification commands  
- `ephemos diagnose` - SPIRE diagnostics
- `ephemos inspect` - Certificate and trust bundle inspection

The core Go library remains unchanged - only the registration command has been removed.