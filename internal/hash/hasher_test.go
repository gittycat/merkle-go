package hash

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/cespare/xxhash/v2"
)

func TestHashFile_SmallFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := []byte("Hello, World!")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash, err := HashFile(testFile)
	if err != nil {
		t.Fatalf("HashFile failed: %v", err)
	}

	// Compute expected hash
	h := xxhash.New()
	h.Write(content)
	expected := hex.EncodeToString(h.Sum(nil))

	if hash != expected {
		t.Errorf("Hash mismatch: expected %s, got %s", expected, hash)
	}
}

func TestHashFile_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.bin")

	// Create a 1MB file
	size := 1024 * 1024
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}

	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash, err := HashFile(testFile)
	if err != nil {
		t.Fatalf("HashFile failed: %v", err)
	}

	// Compute expected hash
	h := xxhash.New()
	h.Write(data)
	expected := hex.EncodeToString(h.Sum(nil))

	if hash != expected {
		t.Errorf("Hash mismatch: expected %s, got %s", expected, hash)
	}
}

func TestHashFile_NonExistent(t *testing.T) {
	_, err := HashFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("HashFile should return error for nonexistent file")
	}
}

func TestHashFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash, err := HashFile(testFile)
	if err != nil {
		t.Fatalf("HashFile failed: %v", err)
	}

	// Empty file should still produce a valid hash
	if hash == "" {
		t.Error("Hash should not be empty string")
	}
}

func TestXXHashFunc(t *testing.T) {
	data := []byte("test data")

	hashBytes, err := XXHashFunc(data)
	if err != nil {
		t.Fatalf("XXHashFunc failed: %v", err)
	}

	if len(hashBytes) != 8 {
		t.Errorf("Expected 8 bytes, got %d", len(hashBytes))
	}

	// Test consistency - same input should produce same output
	hashBytes2, err := XXHashFunc(data)
	if err != nil {
		t.Fatalf("XXHashFunc failed on second call: %v", err)
	}

	if hex.EncodeToString(hashBytes) != hex.EncodeToString(hashBytes2) {
		t.Error("XXHashFunc should be deterministic")
	}
}

func TestXXHashFunc_EmptyData(t *testing.T) {
	hashBytes, err := XXHashFunc([]byte{})
	if err != nil {
		t.Fatalf("XXHashFunc failed: %v", err)
	}

	if len(hashBytes) != 8 {
		t.Errorf("Expected 8 bytes, got %d", len(hashBytes))
	}
}
