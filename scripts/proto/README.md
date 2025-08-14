# Go Protocol Buffer Generator

A comprehensive Go-based protocol buffer code generator that replaces bash scripts with better error handling, dependency management, and cross-platform support.

## Features

- **Automatic Setup**: Installs protoc plugins automatically if missing
- **Smart Generation**: Only regenerates when proto files have changed
- **Validation**: Validates proto files without generating code
- **Cross-platform**: Works on Windows, macOS, and Linux
- **CLI Interface**: Cobra-based CLI with subcommands and options
- **CI/CD Friendly**: Structured output and proper exit codes

## Usage

### Generate Proto Code
```bash
# Using wrapper script (recommended)
scripts/proto/run-proto-generator.sh

# With specific directories
scripts/proto/run-proto-generator.sh proto_input proto_output

# With options
scripts/proto/run-proto-generator.sh --verbose --force

# Using Bazel
bazel run //scripts:proto_generator

# Direct Go execution
go run scripts/proto/main.go proto_input proto_output
```

### Subcommands
```bash
# Install protoc plugins
scripts/proto/run-proto-generator.sh install

# Validate proto files only
scripts/proto/run-proto-generator.sh validate proto_input --verbose

# Clean generated files
scripts/proto/run-proto-generator.sh clean proto_output --verbose
```

## Command Line Options

### Main Command
- `--verbose, -v`: Enable verbose output
- `--force, -f`: Force regeneration even if files are up to date
- `--check-only`: Check if generation is needed without generating

### Subcommands
- `install`: Install required protoc plugins
- `validate [PROTO_DIR]`: Validate proto files without generating code
- `clean [OUTPUT_DIR]`: Remove generated proto files

## Dependencies

The generator automatically installs these tools if missing:
- `protoc`: Protocol buffer compiler
- `protoc-gen-go`: Go protobuf plugin
- `protoc-gen-go-grpc`: Go gRPC plugin

## Installation Requirements

### Protocol Buffer Compiler (protoc)
```bash
# Ubuntu/Debian
sudo apt-get update && sudo apt-get install -y protobuf-compiler

# CentOS/RHEL
sudo yum install -y protobuf-compiler

# macOS
brew install protobuf

# Windows
choco install protoc
```

### Go Plugins (Auto-installed)
The generator automatically installs these if missing:
- `google.golang.org/protobuf/cmd/protoc-gen-go@latest`
- `google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest`

## Output Format

The generator provides structured output with:
- Generation summary with timing
- List of processed files and generated outputs
- Validation results
- Installation status

Example output:
```
🔧 Protocol Buffer Code Generation
Proto directory: proto_input
Output directory: proto_output

✅ protoc-gen-go is available
✅ protoc-gen-go-grpc is available
Using protoc: /usr/bin/protoc (version: libprotoc 3.21.12)
Processing: proto_input/service.proto

============================================================
📊 PROTOBUF GENERATION SUMMARY
============================================================
Protoc version: libprotoc 3.21.12
Generated files: 1
Duration: 0.11s

Generated from:
  proto_input/service.proto → service.pb.go, service_grpc.pb.go

🎉 Protobuf generation completed successfully!
```

## Smart File Detection

The generator automatically:
- Finds all `.proto` files in the specified directory
- Checks if generated files are up to date
- Skips generation if files haven't changed (unless `--force`)
- Provides detailed feedback about what was processed

## Error Handling

Comprehensive error handling for:
- Missing protoc installation with helpful install instructions
- Proto file validation errors with clear messages
- Plugin installation failures with retry suggestions
- Directory permission issues
- Timeout management for all operations

## CI/CD Integration

Features designed for CI/CD:
- Exit codes: 0 for success, 1 for failure
- Structured output for parsing
- Automatic dependency installation
- Timeout protection for all operations
- Detailed logging for debugging

## Advantages over Bash Scripts

1. **Better Error Handling**: Structured error handling with timeouts
2. **Cross-platform**: Works on Windows, macOS, and Linux
3. **Dependency Management**: Automatic plugin installation
4. **Smart Generation**: Only regenerates when needed
5. **Validation**: Can validate proto files without generation
6. **Consistent Interface**: Unified CLI across all operations
7. **Type Safety**: Compile-time error checking
8. **Better Testing**: Unit testable Go code

## Migration from Bash Scripts

This Go implementation replaces:

- `scripts/build/generate-proto.sh` → `proto-generator [dirs]`
- `scripts/generate-proto-ci.sh` → `proto-generator --check-only`
- Manual protoc commands → `proto-generator validate`

The Go version is recommended for all new usage.

## Development

To modify the generator:

1. Edit `scripts/proto/main.go`
2. Update BUILD.bazel if adding dependencies
3. Test with: `go run scripts/proto/main.go`
4. Build with Bazel: `bazel build //scripts:proto_generator`

## Example Workflows

### Development Workflow
```bash
# Validate proto files
scripts/proto/run-proto-generator.sh validate

# Generate code with verbose output
scripts/proto/run-proto-generator.sh --verbose

# Force regeneration
scripts/proto/run-proto-generator.sh --force
```

### CI/CD Workflow
```bash
# Install dependencies
scripts/proto/run-proto-generator.sh install

# Validate proto files
scripts/proto/run-proto-generator.sh validate

# Generate code
scripts/proto/run-proto-generator.sh
```

### Clean Rebuild
```bash
# Clean generated files
scripts/proto/run-proto-generator.sh clean

# Regenerate everything
scripts/proto/run-proto-generator.sh --force
```