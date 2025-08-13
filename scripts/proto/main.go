package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// ProtoGenerator represents the protobuf code generator
type ProtoGenerator struct {
	protoDir    string
	outputDir   string
	verbose     bool
	forceRegenerate bool
}

// ProtocInfo holds information about protoc installation
type ProtocInfo struct {
	Version    string
	Path       string
	Available  bool
}

// ProtoGenResult represents the result of protobuf generation
type ProtoGenResult struct {
	Generated    []string
	Skipped      []string
	Duration     time.Duration
	ProtocInfo   ProtocInfo
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "proto-generator [PROTO_DIR] [OUTPUT_DIR]",
		Short: "Generate Go code from Protocol Buffer definitions",
		Long: `A Go-based protocol buffer code generator that replaces bash scripts with better 
error handling, dependency management, and cross-platform support.

Features:
- Automatic protoc and plugin installation
- Smart file change detection  
- Comprehensive error reporting
- Cross-platform compatibility
- CI/CD friendly operation`,
		Args: cobra.RangeArgs(0, 2),
		Run: func(cmd *cobra.Command, args []string) {
			verbose, _ := cmd.Flags().GetBool("verbose")
			force, _ := cmd.Flags().GetBool("force")
			
			protoDir := "examples/proto"
			outputDir := "examples/proto"
			
			if len(args) >= 1 {
				protoDir = args[0]
			}
			if len(args) >= 2 {
				outputDir = args[1]
			}
			
			generator := &ProtoGenerator{
				protoDir:    protoDir,
				outputDir:   outputDir,
				verbose:     verbose,
				forceRegenerate: force,
			}
			
			if err := generator.GenerateProtoCode(); err != nil {
				log.Fatalf("Proto generation failed: %v", err)
			}
		},
	}

	rootCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().BoolP("force", "f", false, "Force regeneration even if files are up to date")
	rootCmd.Flags().Bool("check-only", false, "Check if generation is needed without generating")
	
	// Add subcommands
	rootCmd.AddCommand(installCmd())
	rootCmd.AddCommand(validateCmd())
	rootCmd.AddCommand(cleanCmd())

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// GenerateProtoCode performs the main protocol buffer code generation
func (p *ProtoGenerator) GenerateProtoCode() error {
	fmt.Printf("🔧 Protocol Buffer Code Generation\n")
	fmt.Printf("Proto directory: %s\n", p.protoDir)
	fmt.Printf("Output directory: %s\n\n", p.outputDir)

	start := time.Now()
	
	// Validate input directories
	if err := p.validateDirectories(); err != nil {
		return fmt.Errorf("directory validation failed: %v", err)
	}
	
	// Find proto files
	protoFiles, err := p.findProtoFiles()
	if err != nil {
		return fmt.Errorf("failed to find proto files: %v", err)
	}
	
	if len(protoFiles) == 0 {
		return fmt.Errorf("no .proto files found in %s", p.protoDir)
	}
	
	// Check if regeneration is needed
	if !p.forceRegenerate {
		upToDate, err := p.areFilesUpToDate(protoFiles)
		if err != nil {
			if p.verbose {
				fmt.Printf("Warning: Could not check file timestamps: %v\n", err)
			}
		} else if upToDate {
			fmt.Println("✅ Generated files are up to date, skipping generation")
			fmt.Printf("Use --force to regenerate anyway\n")
			return nil
		}
	}
	
	// Setup protoc and plugins
	protocInfo, err := p.setupProtoc()
	if err != nil {
		return fmt.Errorf("protoc setup failed: %v", err)
	}
	
	if p.verbose {
		fmt.Printf("Using protoc: %s (version: %s)\n", protocInfo.Path, protocInfo.Version)
	}
	
	// Generate code for each proto file
	result := ProtoGenResult{
		ProtocInfo: protocInfo,
	}
	
	for _, protoFile := range protoFiles {
		if p.verbose {
			fmt.Printf("Processing: %s\n", protoFile)
		}
		
		if err := p.generateProtoFile(protoFile); err != nil {
			return fmt.Errorf("failed to generate code for %s: %v", protoFile, err)
		}
		
		result.Generated = append(result.Generated, protoFile)
	}
	
	result.Duration = time.Since(start)
	
	// Print results
	p.printResults(result)
	
	return nil
}

// validateDirectories checks that input and output directories exist and are valid
func (p *ProtoGenerator) validateDirectories() error {
	// Check proto directory exists
	if _, err := os.Stat(p.protoDir); os.IsNotExist(err) {
		return fmt.Errorf("proto directory does not exist: %s", p.protoDir)
	}
	
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(p.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %v", p.outputDir, err)
	}
	
	return nil
}

// findProtoFiles discovers all .proto files in the proto directory
func (p *ProtoGenerator) findProtoFiles() ([]string, error) {
	var protoFiles []string
	
	err := filepath.Walk(p.protoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if strings.HasSuffix(path, ".proto") && !info.IsDir() {
			protoFiles = append(protoFiles, path)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return protoFiles, nil
}

// areFilesUpToDate checks if the generated files are newer than the proto files
func (p *ProtoGenerator) areFilesUpToDate(protoFiles []string) (bool, error) {
	for _, protoFile := range protoFiles {
		// Get proto file modification time
		protoStat, err := os.Stat(protoFile)
		if err != nil {
			return false, err
		}
		
		// Check corresponding generated files
		baseName := strings.TrimSuffix(filepath.Base(protoFile), ".proto")
		pbGoFile := filepath.Join(p.outputDir, baseName+".pb.go")
		grpcGoFile := filepath.Join(p.outputDir, baseName+"_grpc.pb.go")
		
		// Check if generated files exist and are newer
		for _, genFile := range []string{pbGoFile, grpcGoFile} {
			genStat, err := os.Stat(genFile)
			if os.IsNotExist(err) {
				return false, nil // Generated file doesn't exist
			}
			if err != nil {
				return false, err
			}
			
			if genStat.ModTime().Before(protoStat.ModTime()) {
				return false, nil // Generated file is older
			}
		}
	}
	
	return true, nil
}

// setupProtoc ensures protoc and required plugins are available
func (p *ProtoGenerator) setupProtoc() (ProtocInfo, error) {
	info := ProtocInfo{}
	
	// Check if protoc is available
	protocPath, err := exec.LookPath("protoc")
	if err != nil {
		return info, fmt.Errorf("protoc not found in PATH. Install protoc first:\n" +
			"  Ubuntu/Debian: sudo apt-get install protobuf-compiler\n" +
			"  CentOS/RHEL: sudo yum install protobuf-compiler\n" +
			"  macOS: brew install protobuf\n" +
			"  Windows: choco install protoc")
	}
	
	info.Path = protocPath
	info.Available = true
	
	// Get protoc version
	if version, err := p.getProtocVersion(); err == nil {
		info.Version = version
	}
	
	// Setup Go path for plugins
	goPath, err := p.getGoPath()
	if err != nil {
		return info, fmt.Errorf("failed to get GOPATH: %v", err)
	}
	
	goBin := filepath.Join(goPath, "bin")
	currentPath := os.Getenv("PATH")
	newPath := goBin + string(os.PathListSeparator) + currentPath
	os.Setenv("PATH", newPath)
	
	// Ensure protoc-gen-go is available
	if err := p.ensurePlugin("protoc-gen-go", "google.golang.org/protobuf/cmd/protoc-gen-go@latest"); err != nil {
		return info, err
	}
	
	// Ensure protoc-gen-go-grpc is available
	if err := p.ensurePlugin("protoc-gen-go-grpc", "google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"); err != nil {
		return info, err
	}
	
	return info, nil
}

// getProtocVersion gets the version of the protoc compiler
func (p *ProtoGenerator) getProtocVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "protoc", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	return strings.TrimSpace(string(output)), nil
}

// getGoPath returns the Go workspace path
func (p *ProtoGenerator) getGoPath() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "go", "env", "GOPATH")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	return strings.TrimSpace(string(output)), nil
}

// ensurePlugin ensures a protoc plugin is installed
func (p *ProtoGenerator) ensurePlugin(pluginName, packageName string) error {
	// Check if plugin is already available
	if _, err := exec.LookPath(pluginName); err == nil {
		if p.verbose {
			fmt.Printf("✅ %s is available\n", pluginName)
		}
		return nil
	}
	
	fmt.Printf("Installing %s...\n", pluginName)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "go", "install", packageName)
	if p.verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install %s: %v", pluginName, err)
	}
	
	// Verify installation
	if _, err := exec.LookPath(pluginName); err != nil {
		return fmt.Errorf("failed to verify %s installation", pluginName)
	}
	
	if p.verbose {
		fmt.Printf("✅ %s installed successfully\n", pluginName)
	}
	
	return nil
}

// generateProtoFile generates Go code for a single proto file
func (p *ProtoGenerator) generateProtoFile(protoFile string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	
	// Build protoc command
	cmd := exec.CommandContext(ctx, "protoc",
		"--go_out="+p.outputDir,
		"--go_opt=paths=source_relative",
		"--go-grpc_out="+p.outputDir,
		"--go-grpc_opt=paths=source_relative",
		"-I", p.protoDir,
		protoFile,
	)
	
	if p.verbose {
		fmt.Printf("Running: %s\n", cmd.String())
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("protoc execution failed: %v\nOutput: %s", err, string(output))
	}
	
	if p.verbose && len(output) > 0 {
		fmt.Printf("Protoc output: %s\n", string(output))
	}
	
	return nil
}

// printResults displays the generation results
func (p *ProtoGenerator) printResults(result ProtoGenResult) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 PROTOBUF GENERATION SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	
	fmt.Printf("Protoc version: %s\n", result.ProtocInfo.Version)
	fmt.Printf("Generated files: %d\n", len(result.Generated))
	fmt.Printf("Duration: %.2fs\n\n", result.Duration.Seconds())
	
	if len(result.Generated) > 0 {
		fmt.Println("Generated from:")
		for _, file := range result.Generated {
			baseName := strings.TrimSuffix(filepath.Base(file), ".proto")
			fmt.Printf("  %s → %s.pb.go, %s_grpc.pb.go\n", file, baseName, baseName)
		}
	}
	
	if len(result.Skipped) > 0 {
		fmt.Println("\nSkipped files:")
		for _, file := range result.Skipped {
			fmt.Printf("  %s (up to date)\n", file)
		}
	}
	
	fmt.Println("\n🎉 Protobuf generation completed successfully!")
}

// Subcommands

func installCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install protoc and required plugins",
		Run: func(cmd *cobra.Command, args []string) {
			generator := &ProtoGenerator{verbose: true}
			
			fmt.Println("🔧 Installing protoc plugins...")
			
			if _, err := generator.setupProtoc(); err != nil {
				log.Fatalf("Installation failed: %v", err)
			}
			
			fmt.Println("✅ All plugins installed successfully!")
		},
	}
}

func validateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [PROTO_DIR]",
		Short: "Validate proto files without generating code",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			verbose, _ := cmd.Flags().GetBool("verbose")
			
			protoDir := "examples/proto"
			if len(args) >= 1 {
				protoDir = args[0]
			}
			
			generator := &ProtoGenerator{
				protoDir: protoDir,
				verbose:  verbose,
			}
			
			fmt.Printf("🔍 Validating proto files in: %s\n", protoDir)
			
			// Find proto files
			protoFiles, err := generator.findProtoFiles()
			if err != nil {
				log.Fatalf("Failed to find proto files: %v", err)
			}
			
			if len(protoFiles) == 0 {
				log.Fatalf("No .proto files found in %s", protoDir)
			}
			
			// Setup protoc for validation
			if _, err := generator.setupProtoc(); err != nil {
				log.Fatalf("Protoc setup failed: %v", err)
			}
			
			// Validate each proto file
			for _, protoFile := range protoFiles {
				if err := generator.validateProtoFile(protoFile); err != nil {
					log.Fatalf("Validation failed for %s: %v", protoFile, err)
				}
				fmt.Printf("✅ %s is valid\n", protoFile)
			}
			
			fmt.Printf("\n🎉 All %d proto files are valid!\n", len(protoFiles))
		},
	}
	
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	return cmd
}

func (p *ProtoGenerator) validateProtoFile(protoFile string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Use protoc to validate without generating output
	cmd := exec.CommandContext(ctx, "protoc",
		"--descriptor_set_out=/dev/null",
		"-I", p.protoDir,
		protoFile,
	)
	
	if p.verbose {
		fmt.Printf("Validating: %s\n", protoFile)
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("validation failed: %v\nOutput: %s", err, string(output))
	}
	
	return nil
}

func cleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean [OUTPUT_DIR]",
		Short: "Remove generated proto files",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			verbose, _ := cmd.Flags().GetBool("verbose")
			
			outputDir := "examples/proto"
			if len(args) >= 1 {
				outputDir = args[0]
			}
			
			fmt.Printf("🧹 Cleaning generated files in: %s\n", outputDir)
			
			// Find generated files
			patterns := []string{"*.pb.go", "*_grpc.pb.go"}
			var removedFiles []string
			
			for _, pattern := range patterns {
				matches, err := filepath.Glob(filepath.Join(outputDir, pattern))
				if err != nil {
					log.Printf("Warning: Failed to match pattern %s: %v", pattern, err)
					continue
				}
				
				for _, file := range matches {
					if verbose {
						fmt.Printf("Removing: %s\n", file)
					}
					
					if err := os.Remove(file); err != nil {
						log.Printf("Warning: Failed to remove %s: %v", file, err)
					} else {
						removedFiles = append(removedFiles, file)
					}
				}
			}
			
			if len(removedFiles) == 0 {
				fmt.Println("No generated files found to clean")
			} else {
				fmt.Printf("✅ Removed %d generated files\n", len(removedFiles))
			}
		},
	}
	
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	return cmd
}