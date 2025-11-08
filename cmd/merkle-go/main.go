package main

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"merkle-go/internal/compare"
	"merkle-go/internal/config"
	"merkle-go/internal/tree"
	"merkle-go/internal/walker"

	"github.com/spf13/cobra"
)

var (
	configPath string
	outputPath string
	workers    int
	verbose    bool
	jsonLog    bool
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

		logger := setupLogger()

		// Load config
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		logger.Info("Starting directory walk", slog.String("directory", directory))

		// Walk directory
		walkResult, err := walker.Walk(directory, cfg.Exclude)
		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}

		logger.Info("Files discovered", slog.Int("count", len(walkResult.Files)))

		// Hash files concurrently
		hashResult, err := walker.HashFiles(walkResult.Files, workers)
		if err != nil {
			return fmt.Errorf("failed to hash files: %w", err)
		}

		logger.Info("Files hashed", slog.Int("count", len(hashResult.Hashes)))

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

		logger.Info("Merkle tree built", slog.String("root_hash", merkleTree.RootHash))

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

		logger := setupLogger()

		// Load saved tree
		oldTree, err := tree.Load(treePath)
		if err != nil {
			return fmt.Errorf("failed to load tree: %w", err)
		}

		logger.Info("Loaded saved tree", slog.String("root_hash", oldTree.RootHash))

		// Load config
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Walk directory
		walkResult, err := walker.Walk(directory, cfg.Exclude)
		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}

		// Hash files
		hashResult, err := walker.HashFiles(walkResult.Files, workers)
		if err != nil {
			return fmt.Errorf("failed to hash files: %w", err)
		}

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

func setupLogger() *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if jsonLog {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

func init() {
	defaultWorkers := runtime.NumCPU() * 2

	// Generate command flags
	generateCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (required)")
	generateCmd.Flags().StringVarP(&configPath, "config", "c", "config.yml", "Config file path")
	generateCmd.Flags().IntVarP(&workers, "workers", "w", defaultWorkers, "Number of worker goroutines")
	generateCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug logging")
	generateCmd.Flags().BoolVar(&jsonLog, "json-log", false, "Use JSON log format")
	generateCmd.MarkFlagRequired("output")

	// Compare command flags
	compareCmd.Flags().StringVarP(&configPath, "config", "c", "config.yml", "Config file path")
	compareCmd.Flags().IntVarP(&workers, "workers", "w", defaultWorkers, "Number of worker goroutines")
	compareCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug logging")
	compareCmd.Flags().BoolVar(&jsonLog, "json-log", false, "Use JSON log format")

	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(compareCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
