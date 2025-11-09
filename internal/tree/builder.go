package tree

import (
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"merkle-go/internal/hash"
)

// Build creates a true Merkle tree from file hashes
// Following the classic algorithm:
// 1. Sort files alphabetically by path
// 2. Create leaf nodes (hash each file)
// 3. Pair adjacent nodes and hash them to create parent level
// 4. Repeat until single root hash
func Build(files map[string]FileData, rootPath string) (*MerkleTree, error) {
	// Handle empty files case
	if len(files) == 0 {
		emptyData := []byte("empty-tree")
		rootHash, err := hash.XXHashFunc(emptyData)
		if err != nil {
			return nil, fmt.Errorf("failed to create empty tree hash: %w", err)
		}
		return &MerkleTree{
			Root: &Node{
				Hash: hex.EncodeToString(rootHash),
			},
			RootPath:  rootPath,
			TotalSize: 0,
			Files:     make(map[string]FileData),
		}, nil
	}

	// Sort paths alphabetically for deterministic ordering
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	// Calculate total size
	var totalSize int64
	for _, fileData := range files {
		totalSize += fileData.Size
	}

	// Clean the root path for comparison
	cleanRoot := filepath.Clean(rootPath)

	// Build leaf level: create Node objects with path, size, mtime
	currentLevel := make([]*Node, 0, len(paths))
	for _, path := range paths {
		fileData := files[path]

		// Compute relative path
		relativePath := path
		cleanPath := filepath.Clean(path)
		if strings.HasPrefix(cleanPath, cleanRoot+string(filepath.Separator)) {
			relativePath = strings.TrimPrefix(cleanPath, cleanRoot+string(filepath.Separator))
		} else if cleanPath == cleanRoot {
			relativePath = filepath.Base(cleanPath)
		}

		// Use file content hash directly as the leaf node hash
		node := &Node{
			Hash:  fileData.Hash,
			Path:  relativePath,
			Size:  fileData.Size,
			MTime: fileData.ModTime.Unix(),
		}
		currentLevel = append(currentLevel, node)
	}

	// Build tree by repeatedly pairing and hashing adjacent nodes
	for len(currentLevel) > 1 {
		nextLevel := make([]*Node, 0, (len(currentLevel)+1)/2)

		// Process pairs of nodes
		for i := 0; i < len(currentLevel); i += 2 {
			var parentNode *Node

			if i+1 < len(currentLevel) {
				// Create parent from pair of nodes
				leftNode := currentLevel[i]
				rightNode := currentLevel[i+1]

				// Hash the pair
				leftHashBytes, _ := hex.DecodeString(leftNode.Hash)
				rightHashBytes, _ := hex.DecodeString(rightNode.Hash)
				combined := append(leftHashBytes, rightHashBytes...)
				parentHash, err := hash.XXHashFunc(combined)
				if err != nil {
					return nil, fmt.Errorf("failed to hash parent node: %w", err)
				}

				parentNode = &Node{
					Hash:  hex.EncodeToString(parentHash),
					Left:  leftNode,
					Right: rightNode,
				}
			} else {
				// Odd node: duplicate it
				node := currentLevel[i]
				hashBytes, _ := hex.DecodeString(node.Hash)
				combined := append(hashBytes, hashBytes...)
				parentHash, err := hash.XXHashFunc(combined)
				if err != nil {
					return nil, fmt.Errorf("failed to hash parent node: %w", err)
				}

				parentNode = &Node{
					Hash:  hex.EncodeToString(parentHash),
					Left:  node,
					Right: node,
				}
			}

			nextLevel = append(nextLevel, parentNode)
		}

		currentLevel = nextLevel
	}

	// The last remaining node is the root
	return &MerkleTree{
		Root:      currentLevel[0],
		RootPath:  rootPath,
		TotalSize: totalSize,
		Files:     files,
	}, nil
}
