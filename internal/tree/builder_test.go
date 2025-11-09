package tree

import (
	"testing"
)

func TestBuild_EmptyFiles(t *testing.T) {
	files := make(map[string]FileData)

	tree, err := Build(files, "/test")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if tree.Root.Hash == "" {
		t.Error("Root hash should not be empty even for empty tree")
	}
}

func TestBuild_SingleFile(t *testing.T) {
	files := map[string]FileData{
		"/test/file1.txt": {
			Hash: "abc123",
			Size: 100,
		},
	}

	tree, err := Build(files, "/test")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if tree.Root.Hash == "" {
		t.Error("Root hash should not be empty")
	}

	if len(tree.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(tree.Files))
	}
}

func TestBuild_MultipleFiles(t *testing.T) {
	files := map[string]FileData{
		"/test/file1.txt": {Hash: "hash1", Size: 100},
		"/test/file2.txt": {Hash: "hash2", Size: 200},
		"/test/file3.txt": {Hash: "hash3", Size: 300},
	}

	tree, err := Build(files, "/test")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if tree.Root.Hash == "" {
		t.Error("Root hash should not be empty")
	}

	if len(tree.Files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(tree.Files))
	}
}

func TestBuild_Deterministic(t *testing.T) {
	files := map[string]FileData{
		"/test/file1.txt": {Hash: "hash1", Size: 100},
		"/test/file2.txt": {Hash: "hash2", Size: 200},
	}

	tree1, err := Build(files, "/test")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	tree2, err := Build(files, "/test")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if tree1.Root.Hash != tree2.Root.Hash {
		t.Error("Same input should produce same root hash")
	}
}

func TestBuild_DifferentInputsDifferentHash(t *testing.T) {
	files1 := map[string]FileData{
		"/test/file1.txt": {Hash: "hash1", Size: 100},
	}

	files2 := map[string]FileData{
		"/test/file2.txt": {Hash: "hash2", Size: 200},
	}

	tree1, err := Build(files1, "/test")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	tree2, err := Build(files2, "/test")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if tree1.Root.Hash == tree2.Root.Hash {
		t.Error("Different inputs should produce different root hashes")
	}
}
