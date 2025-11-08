package walker

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestWalk_AllFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test directory structure
	files := []string{
		"file1.txt",
		"file2.go",
		"subdir/file3.txt",
		"subdir/nested/file4.md",
	}

	for _, f := range files {
		fullPath := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// Walk with no exclusions
	result, err := Walk(tmpDir, []string{})
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if len(result.Files) != len(files) {
		t.Errorf("Expected %d files, got %d", len(files), len(result.Files))
	}
}

func TestWalk_WithExclusions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test directory structure
	files := map[string]bool{
		"file1.txt":           false, // should be included
		"file2.tmp":           true,  // should be excluded (*.tmp)
		"file3.log":           true,  // should be excluded (*.log)
		"node_modules/lib.js": true,  // should be excluded (node_modules/)
		"src/main.go":         false, // should be included
		"dist/output.js":      true,  // should be excluded (dist/)
		".git/config":         true,  // should be excluded (.git/)
	}

	for f := range files {
		fullPath := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	exclusions := []string{
		"*.tmp",
		"*.log",
		"node_modules/",
		"dist/",
		".git/",
	}

	result, err := Walk(tmpDir, exclusions)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	// Count expected files
	expectedCount := 0
	for _, shouldExclude := range files {
		if !shouldExclude {
			expectedCount++
		}
	}

	if len(result.Files) != expectedCount {
		t.Errorf("Expected %d files, got %d", expectedCount, len(result.Files))
	}

	// Verify excluded files are not in results
	for _, fileInfo := range result.Files {
		relPath, _ := filepath.Rel(tmpDir, fileInfo.Path)
		if shouldExclude, exists := files[relPath]; exists && shouldExclude {
			t.Errorf("File %s should have been excluded", relPath)
		}
	}
}

func TestWalk_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := Walk(tmpDir, []string{})
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if len(result.Files) != 0 {
		t.Errorf("Expected 0 files in empty directory, got %d", len(result.Files))
	}
}

func TestWalk_NonExistentDirectory(t *testing.T) {
	_, err := Walk("/nonexistent/directory", []string{})
	if err == nil {
		t.Error("Walk should return error for nonexistent directory")
	}
}

func TestWalk_FileInfoMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("Hello, World!")

	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result, err := Walk(tmpDir, []string{})
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if len(result.Files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(result.Files))
	}

	fileInfo := result.Files[0]

	// Check path is absolute
	if !filepath.IsAbs(fileInfo.Path) {
		t.Error("File path should be absolute")
	}

	// Check size
	if fileInfo.Size != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), fileInfo.Size)
	}

	// Check that ModTime is set
	if fileInfo.ModTime.IsZero() {
		t.Error("ModTime should be set")
	}
}

func TestWalk_GlobPatternExclusion(t *testing.T) {
	tmpDir := t.TempDir()

	files := map[string]bool{
		"test.go":      false, // should be included
		"test_test.go": true,  // should be excluded (*_test.go)
		"main_test.go": true,  // should be excluded (*_test.go)
		"main.go":      false, // should be included
	}

	for f := range files {
		fullPath := filepath.Join(tmpDir, f)
		if err := os.WriteFile(fullPath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	exclusions := []string{"*_test.go"}

	result, err := Walk(tmpDir, exclusions)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	// Should only include test.go and main.go
	if len(result.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(result.Files))
	}
}

func TestHashFiles_AllFilesProcessed(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 10 test files
	fileCount := 10
	files := make([]FileInfo, 0, fileCount)

	for i := 0; i < fileCount; i++ {
		filename := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		content := []byte(fmt.Sprintf("content-%d", i))
		if err := os.WriteFile(filename, content, 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		info, _ := os.Stat(filename)
		files = append(files, FileInfo{
			Path:    filename,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}

	// Hash files with 4 workers
	result, err := HashFiles(files, 4, nil)
	if err != nil {
		t.Fatalf("HashFiles failed: %v", err)
	}

	// All files should be hashed
	if len(result.Hashes) != fileCount {
		t.Errorf("Expected %d hashes, got %d", fileCount, len(result.Hashes))
	}

	// All hashes should be non-empty
	for path, hash := range result.Hashes {
		if hash == "" {
			t.Errorf("Hash for %s is empty", path)
		}
	}
}

func TestHashFiles_ErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()

	// Create one valid file and reference one nonexistent file
	validFile := filepath.Join(tmpDir, "valid.txt")
	if err := os.WriteFile(validFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	files := []FileInfo{
		{Path: validFile, Size: 7},
		{Path: "/nonexistent/file.txt", Size: 0},
	}

	result, err := HashFiles(files, 2, nil)
	if err != nil {
		t.Fatalf("HashFiles should not fail completely: %v", err)
	}

	// Valid file should be hashed
	if _, ok := result.Hashes[validFile]; !ok {
		t.Error("Valid file should be hashed")
	}

	// Should have one error
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}
}

func TestHashFiles_Concurrency(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 100 files
	fileCount := 100
	files := make([]FileInfo, 0, fileCount)

	for i := 0; i < fileCount; i++ {
		filename := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		content := []byte(fmt.Sprintf("content-%d", i))
		if err := os.WriteFile(filename, content, 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		info, _ := os.Stat(filename)
		files = append(files, FileInfo{
			Path:    filename,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}

	// Hash with different worker counts
	for _, workers := range []int{1, 2, 4, 8} {
		result, err := HashFiles(files, workers, nil)
		if err != nil {
			t.Fatalf("HashFiles with %d workers failed: %v", workers, err)
		}

		if len(result.Hashes) != fileCount {
			t.Errorf("Workers=%d: Expected %d hashes, got %d", workers, fileCount, len(result.Hashes))
		}
	}
}
