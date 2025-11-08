package tree

import "time"

type FileData struct {
	Hash    string
	Size    int64
	ModTime time.Time
}

type MerkleTree struct {
	RootHash string
	Files    map[string]FileData // path -> FileData
}
