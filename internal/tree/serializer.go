package tree

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type SerializedTree struct {
	Generator string    `json:"generator"`
	Created   time.Time `json:"created"`
	Root      string    `json:"root"`
	Size      string    `json:"size"`
	Tree      *Node     `json:"tree"`
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func Save(tree *MerkleTree, path string) error {
	serialized := SerializedTree{
		Generator: "merkle-go",
		Created:   time.Now(),
		Root:      tree.RootPath,
		Size:      formatSize(tree.TotalSize),
		Tree:      tree.Root,
	}

	data, err := json.MarshalIndent(serialized, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tree: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func Load(path string) (*MerkleTree, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var serialized SerializedTree
	if err := json.Unmarshal(data, &serialized); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tree: %w", err)
	}

	// Calculate total size from the tree and rebuild Files map with absolute paths
	var totalSize int64
	var collectLeaves func(*Node)
	files := make(map[string]FileData)

	collectLeaves = func(node *Node) {
		if node == nil {
			return
		}
		if node.Path != "" {
			// This is a leaf node
			totalSize += node.Size
			// Convert relative path to absolute path
			absolutePath := filepath.Join(serialized.Root, node.Path)
			if node.MTime != 0 {
				files[absolutePath] = FileData{
					Hash:    node.Hash,
					Size:    node.Size,
					ModTime: time.Unix(node.MTime, 0),
				}
			}
		}
		collectLeaves(node.Left)
		collectLeaves(node.Right)
	}
	collectLeaves(serialized.Tree)

	return &MerkleTree{
		Root:      serialized.Tree,
		RootPath:  serialized.Root,
		TotalSize: totalSize,
		Files:     files,
	}, nil
}
