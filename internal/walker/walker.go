package walker

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"merkle-go/internal/hash"
	"merkle-go/internal/progress"
)

type FileInfo struct {
	Path    string
	Size    int64
	ModTime time.Time
}

type WalkResult struct {
	Files  []FileInfo
	Errors []error
}

func Walk(rootPath string, exclusions []string) (*WalkResult, error) {
	result := &WalkResult{
		Files:  make([]FileInfo, 0),
		Errors: make([]error, 0),
	}

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// If error is on the root path, return it (don't continue walking)
			if path == rootPath {
				return err
			}
			// Skip permission errors and continue walking
			result.Errors = append(result.Errors, err)
			return nil
		}

		// Get relative path for matching
		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			result.Errors = append(result.Errors, err)
			return nil
		}

		// Check if path should be excluded
		if shouldExclude(relPath, d, exclusions) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Only add files, not directories
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				result.Errors = append(result.Errors, err)
				return nil
			}

			result.Files = append(result.Files, FileInfo{
				Path:    path,
				Size:    info.Size(),
				ModTime: info.ModTime(),
			})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return result, nil
}

func shouldExclude(relPath string, d fs.DirEntry, exclusions []string) bool {
	for _, pattern := range exclusions {
		// Handle directory exclusions (patterns ending with /)
		if strings.HasSuffix(pattern, "/") {
			dirPattern := strings.TrimSuffix(pattern, "/")
			// Check if the current path or any parent matches the directory pattern
			parts := strings.Split(relPath, string(filepath.Separator))
			for _, part := range parts {
				if matched, _ := filepath.Match(dirPattern, part); matched {
					return true
				}
				// Also check exact match
				if part == dirPattern {
					return true
				}
			}
		} else {
			// Handle file pattern exclusions
			matched, err := filepath.Match(pattern, filepath.Base(relPath))
			if err == nil && matched {
				return true
			}
			// Also try matching against the full relative path for patterns with /
			if strings.Contains(pattern, "/") {
				matched, err := filepath.Match(pattern, relPath)
				if err == nil && matched {
					return true
				}
			}
		}
	}
	return false
}

type HashResult struct {
	Hashes map[string]string // path -> hash
	Errors []error
}

type hashJob struct {
	fileInfo FileInfo
}

type hashJobResult struct {
	path string
	hash string
	err  error
}

func HashFiles(files []FileInfo, numWorkers int, progressBar *progress.Bar) (*HashResult, error) {
	if numWorkers <= 0 {
		numWorkers = 1
	}

	result := &HashResult{
		Hashes: make(map[string]string),
		Errors: make([]error, 0),
	}

	if len(files) == 0 {
		return result, nil
	}

	// Create channels
	jobs := make(chan hashJob, len(files))
	results := make(chan hashJobResult, len(files))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				hashStr, err := hash.HashFile(job.fileInfo.Path)
				results <- hashJobResult{
					path: job.fileInfo.Path,
					hash: hashStr,
					err:  err,
				}
			}
		}()
	}

	// Send jobs
	go func() {
		for _, fileInfo := range files {
			jobs <- hashJob{fileInfo: fileInfo}
		}
		close(jobs)
	}()

	// Wait for workers to finish and close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for jobResult := range results {
		if jobResult.err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", jobResult.path, jobResult.err))
		} else {
			result.Hashes[jobResult.path] = jobResult.hash

			// Update progress bar
			if progressBar != nil {
				dir := filepath.Dir(jobResult.path)
				progressBar.SetDirectory(dir)
				progressBar.Increment()
			}
		}
	}

	return result, nil
}
