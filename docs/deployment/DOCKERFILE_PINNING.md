# Dockerfile Dependency Pinning

This document describes the Dockerfile dependency pinning strategy and fixes applied to ensure reproducible and secure container builds.

## Issue Fixed

### Original Problem
```dockerfile
FROM alpine@sha256:bc41182d7ef5ffc53a40b044e725193bc10142a1243f395ee852a8d9730fc2ad # alpine:latest
```

**Issues:**
1. Invalid syntax - missing tag before digest
2. Outdated SHA256 digest
3. Comment suggests intent but doesn't match implementation

### Correct Syntax for Pinned Dependencies

**Option 1: Tag with Digest (Recommended)**
```dockerfile
FROM alpine:3.20@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1
```

**Option 2: Digest Only**
```dockerfile
FROM alpine@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1
```

## Files Created/Modified

### 1. Dockerfile (Fixed)
- Corrected base image syntax
- Updated to current Alpine 3.20 digest
- Added descriptive comment

### 2. Dockerfile.multistage (New)
- Multi-stage build for optimization
- Separate build and runtime stages
- Non-root user for security
- Minimal final image size

### 3. .dockerignore (New)
- Excludes unnecessary files from build context
- Reduces build time and image size
- Improves security by excluding sensitive files

## Security Benefits

### 1. Pinned Dependencies
- **Reproducible Builds**: Same digest always produces same image
- **Supply Chain Security**: Prevents unexpected base image changes
- **Vulnerability Management**: Known base image version for CVE tracking

### 2. Multi-Stage Build
- **Reduced Attack Surface**: Final image only contains runtime dependencies
- **No Build Tools**: Compilers and development tools not in production image
- **Smaller Image Size**: Typically 50-80% smaller than single-stage builds

### 3. Non-Root User
- **Privilege Separation**: Application runs without root privileges
- **Container Escape Mitigation**: Limits impact of potential vulnerabilities
- **Best Practice Compliance**: Follows container security guidelines

## How to Build

### Simple Dockerfile
```bash
docker build -t ephemos:latest .
```

### Multi-Stage Dockerfile
```bash
docker build -f Dockerfile.multistage -t ephemos:latest .
```

## Updating Pinned Digests

To update the pinned digest to a newer version:

1. **Find the latest digest:**
```bash
docker pull alpine:3.20
docker inspect alpine:3.20 | grep -i digest
```

2. **Update Dockerfile:**
```dockerfile
FROM alpine:3.20@sha256:<new-digest-here>
```

3. **Test the build:**
```bash
docker build --no-cache -t test-build .
```

## Verification

### Check Image Digest
```bash
docker images --digests ephemos
```

### Scan for Vulnerabilities
```bash
# Using Trivy
trivy image ephemos:latest

# Using Docker Scout
docker scout cves ephemos:latest
```

## Best Practices

1. **Regular Updates**: Update base image digests monthly or when security patches are released
2. **Version Pinning**: Pin both tag AND digest for clarity
3. **Documentation**: Document why specific versions are chosen
4. **Automation**: Use Renovate or Dependabot to automate updates
5. **Testing**: Always test builds after updating digests

## References

- [Docker Official Images](https://hub.docker.com/_/alpine)
- [Alpine Linux Security](https://alpinelinux.org/releases/)
- [Docker Security Best Practices](https://docs.docker.com/develop/security-best-practices/)
- [OCI Image Specification](https://github.com/opencontainers/image-spec)