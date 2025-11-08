package compare

import (
	"fmt"
	"sort"

	"merkle-go/internal/tree"
)

type ChangeType string

const (
	Added    ChangeType = "ADDED"
	Modified ChangeType = "MODIFIED"
	Deleted  ChangeType = "DELETED"
)

type Change struct {
	Type    ChangeType
	Path    string
	OldData *tree.FileData
	NewData *tree.FileData
}

type CompareResult struct {
	Added    []Change
	Modified []Change
	Deleted  []Change
}

func (r *CompareResult) HasChanges() bool {
	return len(r.Added) > 0 || len(r.Modified) > 0 || len(r.Deleted) > 0
}

func Compare(oldTree, newTree *tree.MerkleTree) *CompareResult {
	result := &CompareResult{
		Added:    make([]Change, 0),
		Modified: make([]Change, 0),
		Deleted:  make([]Change, 0),
	}

	// Check for added and modified files
	for path, newData := range newTree.Files {
		if oldData, exists := oldTree.Files[path]; exists {
			// File exists in both - check if modified
			if oldData.Hash != newData.Hash {
				oldDataCopy := oldData
				newDataCopy := newData
				result.Modified = append(result.Modified, Change{
					Type:    Modified,
					Path:    path,
					OldData: &oldDataCopy,
					NewData: &newDataCopy,
				})
			}
		} else {
			// File only in new tree - added
			newDataCopy := newData
			result.Added = append(result.Added, Change{
				Type:    Added,
				Path:    path,
				NewData: &newDataCopy,
			})
		}
	}

	// Check for deleted files
	for path, oldData := range oldTree.Files {
		if _, exists := newTree.Files[path]; !exists {
			oldDataCopy := oldData
			result.Deleted = append(result.Deleted, Change{
				Type:    Deleted,
				Path:    path,
				OldData: &oldDataCopy,
			})
		}
	}

	// Sort for deterministic output
	sort.Slice(result.Added, func(i, j int) bool {
		return result.Added[i].Path < result.Added[j].Path
	})
	sort.Slice(result.Modified, func(i, j int) bool {
		return result.Modified[i].Path < result.Modified[j].Path
	})
	sort.Slice(result.Deleted, func(i, j int) bool {
		return result.Deleted[i].Path < result.Deleted[j].Path
	})

	return result
}

func FormatReport(result *CompareResult) string {
	if !result.HasChanges() {
		return "No changes detected."
	}

	report := "Changes detected:\n\n"

	if len(result.Added) > 0 {
		report += fmt.Sprintf("ADDED (%d files):\n", len(result.Added))
		for _, change := range result.Added {
			report += fmt.Sprintf("  + %s (hash: %s, size: %d bytes)\n",
				change.Path, change.NewData.Hash, change.NewData.Size)
		}
		report += "\n"
	}

	if len(result.Modified) > 0 {
		report += fmt.Sprintf("MODIFIED (%d files):\n", len(result.Modified))
		for _, change := range result.Modified {
			report += fmt.Sprintf("  ~ %s\n", change.Path)
			report += fmt.Sprintf("    Old: hash=%s, size=%d bytes, modified=%s\n",
				change.OldData.Hash, change.OldData.Size, change.OldData.ModTime.Format("2006-01-02"))
			report += fmt.Sprintf("    New: hash=%s, size=%d bytes, modified=%s\n",
				change.NewData.Hash, change.NewData.Size, change.NewData.ModTime.Format("2006-01-02"))
		}
		report += "\n"
	}

	if len(result.Deleted) > 0 {
		report += fmt.Sprintf("DELETED (%d files):\n", len(result.Deleted))
		for _, change := range result.Deleted {
			report += fmt.Sprintf("  - %s (hash: %s, size: %d bytes)\n",
				change.Path, change.OldData.Hash, change.OldData.Size)
		}
		report += "\n"
	}

	report += fmt.Sprintf("Summary: %d added, %d modified, %d deleted\n",
		len(result.Added), len(result.Modified), len(result.Deleted))

	return report
}
