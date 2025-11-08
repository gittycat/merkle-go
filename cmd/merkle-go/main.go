package main

import (
	"fmt"
	"os"
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
	Use:   "merkle-go",
	Short: "Generate and compare merkle trees of file hashes",
	Long:  `A CLI tool that generates merkle trees from directory trees and compares them to detect changes.`,
}

var generateCmd = &cobra.Command{
	Use:   "generate <directory>",
	Short: "Generate merkle tree from directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		directory := args[0]

		// Load config
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Printf("Scanning directory: %s\n", directory)

		// Walk directory
		walkResult, err := walker.Walk(directory, cfg.Exclude)
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
		merkleTree, err := tree.Build(fileDataMap)
		if err != nil {
			return fmt.Errorf("failed to build merkle tree: %w", err)
		}

		// Save to file
		if err := tree.Save(merkleTree, outputPath); err != nil {
			return fmt.Errorf("failed to save tree: %w", err)
		}

		fmt.Printf("✓ Merkle tree generated successfully\n")
		fmt.Printf("  Root hash: %s\n", merkleTree.RootHash)
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

		// Load saved tree
		oldTree, err := tree.Load(treePath)
		if err != nil {
			return fmt.Errorf("failed to load tree: %w", err)
		}

		fmt.Printf("Loaded saved tree (root: %s)\n", oldTree.RootHash[:16]+"...")

		// Load config
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Printf("Scanning directory: %s\n", directory)

		// Walk directory
		walkResult, err := walker.Walk(directory, cfg.Exclude)
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
		newTree, err := tree.Build(fileDataMap)
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

	// Generate command flags
	generateCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (required)")
	generateCmd.Flags().StringVarP(&configPath, "config", "c", "config.yml", "Config file path")
	generateCmd.Flags().IntVarP(&workers, "workers", "w", defaultWorkers, "Number of worker goroutines")
	generateCmd.MarkFlagRequired("output")

	// Compare command flags
	compareCmd.Flags().StringVarP(&configPath, "config", "c", "config.yml", "Config file path")
	compareCmd.Flags().IntVarP(&workers, "workers", "w", defaultWorkers, "Number of worker goroutines")

	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(compareCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
