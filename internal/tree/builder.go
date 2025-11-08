package tree

import (
	"encoding/hex"
	"fmt"
	"sort"

	"merkle-go/internal/hash"

	merkletree "github.com/txaty/go-merkletree"
)

type dataBlock struct {
	path string
	data []byte
}

func (d *dataBlock) Serialize() ([]byte, error) {
	return d.data, nil
}

func Build(files map[string]FileData) (*MerkleTree, error) {
	// Handle empty files case
	if len(files) == 0 {
		// Create an empty tree with a deterministic root hash
		emptyData := []byte("empty-tree")
		rootHash, err := hash.XXHashFunc(emptyData)
		if err != nil {
			return nil, fmt.Errorf("failed to create empty tree hash: %w", err)
		}
		return &MerkleTree{
			RootHash: hex.EncodeToString(rootHash),
			Files:    make(map[string]FileData),
		}, nil
	}

	// Handle single file case (merkle tree library requires at least 2 blocks)
	if len(files) == 1 {
		for path, fileData := range files {
			data := []byte(path + ":" + fileData.Hash)
			rootHash, err := hash.XXHashFunc(data)
			if err != nil {
				return nil, fmt.Errorf("failed to hash single file: %w", err)
			}
			return &MerkleTree{
				RootHash: hex.EncodeToString(rootHash),
				Files:    files,
			}, nil
		}
	}

	// Sort paths for deterministic ordering
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	// Create data blocks
	blocks := make([]merkletree.DataBlock, 0, len(paths))
	for _, path := range paths {
		fileData := files[path]
		// Combine path and hash for the merkle tree leaf
		data := []byte(path + ":" + fileData.Hash)
		blocks = append(blocks, &dataBlock{
			path: path,
			data: data,
		})
	}

	// Configure merkle tree
	config := &merkletree.Config{
		HashFunc:         hash.XXHashFunc,
		RunInParallel:    true,
		NumRoutines:      0, // 0 = use runtime.NumCPU()
		SortSiblingPairs: true,
		Mode:             merkletree.ModeTreeBuild,
	}

	// Build merkle tree
	mt, err := merkletree.New(config, blocks)
	if err != nil {
		return nil, fmt.Errorf("failed to build merkle tree: %w", err)
	}

	// Get root hash
	root := mt.Root
	rootHashStr := hex.EncodeToString(root)

	return &MerkleTree{
		RootHash: rootHashStr,
		Files:    files,
	}, nil
}
