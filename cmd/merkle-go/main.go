package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"merkle-go/internal/compare"
	"merkle-go/internal/config"
	"merkle-go/internal/progress"
	"merkle-go/internal/tree"
	"merkle-go/internal/walker"
)

func generateTree(args []string) error {
	fs := flag.NewFlagSet("merkle-go", flag.ExitOnError)
	configPath := fs.String("config", "config.toml", "Config file path")
	configPathShort := fs.String("c", "config.toml", "Config file path (shorthand)")
	workers := fs.Int("workers", runtime.NumCPU()*2, "Number of worker goroutines")
	workersShort := fs.Int("w", runtime.NumCPU()*2, "Number of worker goroutines (shorthand)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: merkle-go [options] <directory> [output-json-filename]\n\n")
		fmt.Fprintf(os.Stderr, "Generate a merkle tree from a directory tree and save it to a JSON file.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Merge short and long flag values
	if *configPathShort != "config.toml" {
		*configPath = *configPathShort
	}
	if *workersShort != runtime.NumCPU()*2 {
		*workers = *workersShort
	}

	if fs.NArg() < 1 || fs.NArg() > 2 {
		fs.Usage()
		os.Exit(1)
	}

	directory := fs.Arg(0)
	var outputPath string
	if fs.NArg() == 2 {
		outputPath = fs.Arg(1)
	}

	// Load config
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Set output path - from args, config, or default
	if outputPath == "" {
		outputPath = cfg.OutputFile
	}

	// Convert to absolute path
	absDirectory, err := filepath.Abs(directory)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	fmt.Printf("Scanning directory: %s\n", absDirectory)

	// Walk directory
	walkResult, err := walker.Walk(absDirectory, cfg.Skip)
	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	fmt.Printf("Found %d files\n", len(walkResult.Files))
	fmt.Println("Hashing files...")

	// Create progress bar
	bar := progress.New(int64(len(walkResult.Files)))

	// Hash files concurrently
	hashResult, err := walker.HashFiles(walkResult.Files, *workers, bar)
	if err != nil {
		return fmt.Errorf("failed to hash files: %w", err)
	}

	bar.Finish()

	// Build file data map
	fileDataMap := make(map[string]tree.FileData)
	for _, fileInfo := range walkResult.Files {
		if hash, ok := hashResult.Hashes[fileInfo.Path]; ok {
			fileDataMap[fileInfo.Path] = tree.FileData{
				Hash:    hash,
				Size:    fileInfo.Size,
				ModTime: fileInfo.ModTime,
			}
		}
	}

	// Build merkle tree
	merkleTree, err := tree.Build(fileDataMap, absDirectory)
	if err != nil {
		return fmt.Errorf("failed to build merkle tree: %w", err)
	}

	// If no output path specified, use root hash as filename in ./output/
	if outputPath == "" {
		outputPath = filepath.Join("output", merkleTree.Root.Hash+".json")
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Save to file
	if err := tree.Save(merkleTree, outputPath); err != nil {
		return fmt.Errorf("failed to save tree: %w", err)
	}

	fmt.Printf("✓ Merkle tree generated successfully\n")
	fmt.Printf("  Root hash: %s\n", merkleTree.Root.Hash)
	fmt.Printf("  Files: %d\n", len(merkleTree.Files))
	fmt.Printf("  Output: %s\n", outputPath)

	if len(hashResult.Errors) > 0 {
		fmt.Printf("\n⚠ Skipped %d files due to errors\n", len(hashResult.Errors))
	}

	return nil
}

func compareTree(args []string) error {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	configPath := fs.String("config", "config.toml", "Config file path")
	configPathShort := fs.String("c", "config.toml", "Config file path (shorthand)")
	workers := fs.Int("workers", runtime.NumCPU()*2, "Number of worker goroutines")
	workersShort := fs.Int("w", runtime.NumCPU()*2, "Number of worker goroutines (shorthand)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: merkle-go compare [options] <tree.json> <directory>\n\n")
		fmt.Fprintf(os.Stderr, "Compare saved merkle tree against current directory.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Merge short and long flag values
	if *configPathShort != "config.toml" {
		*configPath = *configPathShort
	}
	if *workersShort != runtime.NumCPU()*2 {
		*workers = *workersShort
	}

	if fs.NArg() != 2 {
		fs.Usage()
		os.Exit(1)
	}

	treePath := fs.Arg(0)
	directory := fs.Arg(1)

	// Convert to absolute path
	absDirectory, err := filepath.Abs(directory)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Load saved tree
	oldTree, err := tree.Load(treePath)
	if err != nil {
		return fmt.Errorf("failed to load tree: %w", err)
	}

	fmt.Printf("Loaded saved tree (root: %s)\n", oldTree.Root.Hash[:16]+"...")

	// Load config
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("Scanning directory: %s\n", absDirectory)

	// Walk directory
	walkResult, err := walker.Walk(absDirectory, cfg.Skip)
	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	fmt.Printf("Found %d files\n", len(walkResult.Files))
	fmt.Println("Hashing files...")

	// Create progress bar
	bar := progress.New(int64(len(walkResult.Files)))

	// Hash files
	hashResult, err := walker.HashFiles(walkResult.Files, *workers, bar)
	if err != nil {
		return fmt.Errorf("failed to hash files: %w", err)
	}

	bar.Finish()

	// Build file data map
	fileDataMap := make(map[string]tree.FileData)
	for _, fileInfo := range walkResult.Files {
		if hash, ok := hashResult.Hashes[fileInfo.Path]; ok {
			fileDataMap[fileInfo.Path] = tree.FileData{
				Hash:    hash,
				Size:    fileInfo.Size,
				ModTime: fileInfo.ModTime,
			}
		}
	}

	// Build new tree
	newTree, err := tree.Build(fileDataMap, absDirectory)
	if err != nil {
		return fmt.Errorf("failed to build merkle tree: %w", err)
	}

	// Compare trees
	result := compare.Compare(oldTree, newTree)

	// Print report
	fmt.Println(compare.FormatReport(result))

	if len(hashResult.Errors) > 0 {
		fmt.Printf("Skipped: %d files\n", len(hashResult.Errors))
	}

	// Exit with appropriate code
	if len(hashResult.Errors) > 0 {
		os.Exit(2) // Errors occurred
	}
	if result.HasChanges() {
		os.Exit(1) // Changes detected
	}
	os.Exit(0) // No changes

	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: merkle-go [options] <directory> [output-json-filename]\n")
		fmt.Fprintf(os.Stderr, "       merkle-go compare [options] <tree.json> <directory>\n")
		os.Exit(1)
	}

	var err error
	if os.Args[1] == "compare" {
		err = compareTree(os.Args[2:])
	} else {
		err = generateTree(os.Args[1:])
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
