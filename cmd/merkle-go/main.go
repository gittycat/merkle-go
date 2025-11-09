package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"merkle-go/internal/compare"
	"merkle-go/internal/config"
	"merkle-go/internal/progress"
	"merkle-go/internal/tree"
	"merkle-go/internal/walker"

	"github.com/spf13/cobra"
)

var (
	configPath string
	outputPath string
	workers    int
)

var rootCmd = &cobra.Command{
	Use:   "merkle-go <directory> [output-json-filename]",
	Short: "Generate merkle tree from directory",
	Long:  `Generate a merkle tree from a directory tree and save it to a JSON file.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		directory := args[0]

		// Load config
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Set output path - from args, config, or default
		outputPath := cfg.OutputFile
		if len(args) == 2 {
			outputPath = args[1]
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
		hashResult, err := walker.HashFiles(walkResult.Files, workers, bar)
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
	},
}

var compareCmd = &cobra.Command{
	Use:   "compare <tree.json> <directory>",
	Short: "Compare saved merkle tree against current directory",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		treePath := args[0]
		directory := args[1]

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
		cfg, err := config.LoadConfig(configPath)
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
		hashResult, err := walker.HashFiles(walkResult.Files, workers, bar)
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
	},
}

func init() {
	defaultWorkers := runtime.NumCPU() * 2

	// Root command flags
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "config.toml", "Config file path")
	rootCmd.Flags().IntVarP(&workers, "workers", "w", defaultWorkers, "Number of worker goroutines")

	// Compare command flags
	compareCmd.Flags().StringVarP(&configPath, "config", "c", "config.toml", "Config file path")
	compareCmd.Flags().IntVarP(&workers, "workers", "w", defaultWorkers, "Number of worker goroutines")

	rootCmd.AddCommand(compareCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
