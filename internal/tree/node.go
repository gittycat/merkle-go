package tree

import "time"

type FileData struct {
	Hash    string
	Size    int64
	ModTime time.Time
}

type Node struct {
	Hash  string `json:"hash"`
	Left  *Node  `json:"left,omitempty"`
	Right *Node  `json:"right,omitempty"`
	Path  string `json:"path,omitempty"` // Only set for leaf nodes
	Size  int64  `json:"size,omitempty"` // Only set for leaf nodes
	MTime int64  `json:"mtime,omitempty"` // Only set for leaf nodes (Unix timestamp)
}

type MerkleTree struct {
	Root      *Node
	RootPath  string              // Absolute path of scanned directory
	TotalSize int64               // Total size in bytes
	Files     map[string]FileData // path -> FileData (kept for compatibility)
}
