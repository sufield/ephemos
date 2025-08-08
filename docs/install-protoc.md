The error shows that `protoc-gen-go` and `protoc-gen-go-grpc` are not found in your PATH, even though the installation script claimed they were installed. Here's how to fix this:

## 1. First, check if the Go tools were actually installed:

```bash
ls -la ~/go/bin/
```

You should see `protoc-gen-go` and `protoc-gen-go-grpc` in there.

## 2. Check if your Go bin directory is in your PATH:

```bash
echo $PATH | grep -o ~/go/bin
```

If this returns nothing, your Go bin directory isn't in your PATH.

## 3. Add Go bin to your PATH:

```bash
# Add to your shell profile
echo 'export PATH=$PATH:~/go/bin' >> ~/.bashrc

# Reload your shell configuration
source ~/.bashrc

# Or alternatively, add it to ~/.profile for system-wide effect
echo 'export PATH=$PATH:~/go/bin' >> ~/.profile
```

## 4. If the tools aren't installed, install them manually:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## 5. Verify the installation:

```bash
which protoc-gen-go
which protoc-gen-go-grpc
protoc-gen-go --version
```

## 6. Try your make command again:

```bash
make proto
```

## Alternative Quick Fix:

If you're still having issues, you can specify the full path in your Makefile temporarily:

```bash
protoc --go_out=. --go_opt=paths=source_relative \
  --plugin=protoc-gen-go=$HOME/go/bin/protoc-gen-go \
  your_proto_files
```

The most common cause of this issue is that `~/go/bin` (where Go installs binaries) isn't in your system's PATH environment variable.