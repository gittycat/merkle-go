package hash

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/cespare/xxhash/v2"
)

const bufferSize = 32 * 1024 // 32KB buffer for streaming

// HashFile computes the xxHash of a file using streaming for large files
func HashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	h := xxhash.New()
	buf := make([]byte, bufferSize)

	for {
		n, err := file.Read(buf)
		if n > 0 {
			h.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// XXHashFunc is a custom hash function adapter for go-merkletree
// It converts []byte input to xxHash []byte output
func XXHashFunc(data []byte) ([]byte, error) {
	h := xxhash.New()
	h.Write(data)
	sum := h.Sum64()

	// Convert uint64 to []byte in big-endian format
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, sum)
	return buf, nil
}
