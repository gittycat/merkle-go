package tree

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type SerializedTree struct {
	Version     string              `json:"version"`
	GeneratedAt time.Time           `json:"generated_at"`
	RootHash    string              `json:"root_hash"`
	Files       map[string]FileData `json:"files"`
	Stats       Stats               `json:"stats"`
}

type Stats struct {
	TotalFiles int   `json:"total_files"`
	TotalSize  int64 `json:"total_size"`
}

func Save(tree *MerkleTree, path string) error {
	stats := Stats{
		TotalFiles: len(tree.Files),
	}
	for _, fileData := range tree.Files {
		stats.TotalSize += fileData.Size
	}

	serialized := SerializedTree{
		Version:     "1.0",
		GeneratedAt: time.Now(),
		RootHash:    tree.RootHash,
		Files:       tree.Files,
		Stats:       stats,
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

	return &MerkleTree{
		RootHash: serialized.RootHash,
		Files:    serialized.Files,
	}, nil
}
