# Implementation Plan: merkle-go

## Project Overview
A Go CLI application that generates merkle trees of file hashes from directory trees and compares them to detect changes. Optimized for speed using xxHash and concurrent file processing.

## Project Structure
```
merkle-go/
├── cmd/merkle-go/
│   └── main.go                 # CLI entry point
├── internal/
│   ├── hash/
│   │   ├── hasher.go          # xxHash file hashing
│   │   └── hasher_test.go
│   ├── tree/
│   │   ├── builder.go         # Merkle tree construction
│   │   ├── builder_test.go
│   │   ├── node.go            # Tree node structure
│   │   └── serializer.go      # JSON serialization
│   ├── walker/
│   │   ├── walker.go          # Concurrent file walking
│   │   └── walker_test.go
│   ├── compare/
│   │   ├── comparer.go        # Tree comparison logic
│   │   └── comparer_test.go
│   └── config/
│       ├── config.go          # config.yml parsing
│       └── config_test.go
├── go.mod
├── go.sum
├── config.yml                  # Exclusion patterns & settings
└── README.md
```

## Dependencies
- **`github.com/spf13/cobra`** - Industry-standard CLI framework (used by kubectl, hugo, gh)
- **`github.com/cespare/xxhash/v2`** - Fast xxHash implementation (non-cryptographic)
- **`github.com/txaty/go-merkletree`** - High-performance merkle tree with parallel processing support
- **Standard library**:
  - `slog` - Structured logging (Go 1.21+)
  - `encoding/json` - JSON serialization
  - `sync/errgroup` - Error propagation in goroutines
  - `filepath` - File tree walking

## CLI Commands

### Command 1: `merkle-go generate <directory> -o <output.json>`
**Purpose**: Generate merkle tree from directory and save to JSON file

**Process**:
1. Parse config.yml for exclusion patterns and settings
2. Walk directory tree using `filepath.WalkDir` (respecting exclusions)
3. Hash files concurrently using worker pool with xxHash
4. Build merkle tree with custom xxHash function
5. Save full tree structure + metadata to JSON
6. Report skipped files (permission errors, I/O errors) at end if any

**Output**: JSON file containing merkle tree with file paths, hashes, sizes, modtimes

**Flags**:
- `-o, --output` - Output file path (required)
- `-c, --config` - Config file path (default: config.yml)
- `-w, --workers` - Number of worker goroutines (default: 2×CPU cores)
- `-v, --verbose` - Enable debug logging
- `--json-log` - Use JSON log format instead of text

### Command 2: `merkle-go compare <output.json> <directory>`
**Purpose**: Compare saved merkle tree against current directory state

**Process**:
1. Load saved merkle tree from JSON file
2. Regenerate merkle tree from current directory (using same exclusions)
3. Compare trees and identify changes
4. Output detailed change report

**Output Format** (detailed change report):
```
Changes detected:

ADDED (2 files):
  + path/to/new-file.txt (hash: abc123..., size: 1024 bytes)
  + another/new-file.go (hash: def456..., size: 2048 bytes)

MODIFIED (1 file):
  ~ config.yml
    Old: hash=789ghi..., size=512 bytes, modified=2025-01-15
    New: hash=012jkl..., size=600 bytes, modified=2025-01-16

DELETED (1 file):
  - old/removed-file.py (hash: mno345..., size: 256 bytes)

Summary: 2 added, 1 modified, 1 deleted
Skipped: 0 files
```

**Exit Codes**:
- `0` - No changes detected
- `1` - Changes detected
- `2` - Errors occurred (with skip-and-warn summary)

**Flags**:
- `-c, --config` - Config file path (default: config.yml)
- `-w, --workers` - Number of worker goroutines (default: 2×CPU cores)
- `-v, --verbose` - Enable debug logging
- `--json-log` - Use JSON log format instead of text

## Configuration File (config.yml)

```yaml
# Exclusion patterns (gitignore-style syntax)
exclude:
  - "*.tmp"
  - "*.log"
  - ".git/"
  - "node_modules/"
  - "__pycache__/"
  - "*.pyc"
  - ".DS_Store"

# Future settings can be added here
# worker_count: 8
# hash_algorithm: xxhash  # for future extensibility
```

## Implementation Phases (TDD Approach)

### Phase 1: Configuration & Setup
**Test-Driven Development**:
1. **Test**: Create test config.yml, verify parsing loads exclude patterns correctly
2. **Implement**: Config struct with YAML unmarshaling using `gopkg.in/yaml.v3`
3. **Test**: Verify default config when file doesn't exist
4. **Implement**: Default config fallback logic
5. **Test**: Invalid YAML returns proper error
6. **Implement**: Error handling with descriptive messages

**Deliverable**: `internal/config/config.go` with full test coverage

---

### Phase 2: File Hashing
**Test-Driven Development**:
1. **Test**: Hash small known file, verify xxHash output matches expected value
2. **Implement**: Basic file hasher using `xxhash.New()`
3. **Test**: Hash large file (>10MB) using streaming, verify correctness
4. **Implement**: Streaming hash for large files (read in chunks)
5. **Test**: Hash non-existent file returns error
6. **Implement**: File opening with error handling
7. **Test**: Thread-safe wrapper for merkle tree integration
8. **Implement**: xxHash adapter function for `go-merkletree` custom hash

**Key Implementation Details**:
```go
// Custom hash function for merkle tree
func xxHashFunc(data []byte) ([]byte, error) {
    h := xxhash.New()
    h.Write(data)
    sum := h.Sum64()
    // Convert uint64 to []byte
    buf := make([]byte, 8)
    binary.BigEndian.PutUint64(buf, sum)
    return buf, nil
}
```

**Deliverable**: `internal/hash/hasher.go` with streaming support and test coverage

---

### Phase 3: File Walking with Exclusions
**Test-Driven Development**:
1. **Test**: Create test directory tree, verify walker returns all files
2. **Implement**: Basic walker using `filepath.WalkDir`
3. **Test**: Walker respects exclusion patterns (*.tmp, node_modules/, etc.)
4. **Implement**: Pattern matching using `filepath.Match` and custom logic
5. **Test**: Walker handles symlinks correctly (follow or skip)
6. **Implement**: Symlink handling logic
7. **Test**: Walker skips permission-denied directories, logs error
8. **Implement**: Error collection and logging with `slog`

**Key Implementation Details**:
- Use `filepath.WalkDir` (not `Walk`) for efficiency
- Only call `d.Info()` when needed (modtime, size for comparison)
- Collect file paths in slice, process in worker pool
- Return both files list and errors list

**Deliverable**: `internal/walker/walker.go` with exclusion support

---

### Phase 4: Concurrent Worker Pool
**Test-Driven Development**:
1. **Test**: Worker pool processes all jobs, no jobs lost
2. **Implement**: Basic worker pool with job/result channels
3. **Test**: Worker pool with 4 workers hashes 100 files correctly
4. **Implement**: Configurable worker count
5. **Test**: Worker pool handles errors without crashing (one file fails)
6. **Implement**: Error collection using `errgroup` or custom error channel
7. **Test**: Worker pool performance benchmark (concurrent vs sequential)
8. **Implement**: Optimization based on benchmark results

**Architecture**:
```
filepath.WalkDir → File Paths → Job Channel → Worker Pool → Result Channel → Results
                                      ↓
                                  xxHash File
```

**Key Implementation Details**:
- Default worker count: `2 × runtime.NumCPU()` (I/O-bound workload)
- Buffered channels (size: 100-1000) to prevent blocking
- Use `sync.WaitGroup` or `errgroup` for synchronization
- Each worker reuses xxHash instance (reset between uses)

**Deliverable**: `internal/walker/walker.go` (concurrent version) with benchmarks

---

### Phase 5: Merkle Tree Construction
**Test-Driven Development**:
1. **Test**: Build merkle tree from small list of hashes, verify root hash
2. **Implement**: Integration with `txaty/go-merkletree` library
3. **Test**: Custom xxHash function produces consistent tree
4. **Implement**: xxHash adapter configured in tree builder
5. **Test**: Parallel tree building enabled, verify thread-safety
6. **Implement**: Enable parallel mode with `RunInParallel: true`
7. **Test**: Tree preserves file metadata (path, size, modtime)
8. **Implement**: Custom tree node structure with metadata

**Key Implementation Details**:
```go
config := &merkletree.Config{
    HashFunc:         xxHashFunc,
    RunInParallel:    true,
    NumRoutines:      0, // 0 = use runtime.NumCPU()
    SortSiblingPairs: true,
    Mode:             merkletree.ModeTreeBuild,
}
```

**Deliverable**: `internal/tree/builder.go` with parallel tree construction

---

### Phase 6: JSON Serialization
**Test-Driven Development**:
1. **Test**: Serialize merkle tree to JSON, verify structure
2. **Implement**: JSON marshaling with proper structure
3. **Test**: Deserialize JSON back to tree, verify equality
4. **Implement**: JSON unmarshaling
5. **Test**: Large tree serialization (1000+ files) performance
6. **Implement**: Optimization if needed (streaming, compression consideration)
7. **Test**: Pretty-printed JSON is human-readable
8. **Implement**: JSON indent formatting

**JSON Structure**:
```json
{
  "version": "1.0",
  "generated_at": "2025-01-16T10:30:00Z",
  "root_hash": "abc123...",
  "config": {
    "hash_algorithm": "xxhash",
    "exclusions": ["*.tmp", ".git/"]
  },
  "files": [
    {
      "path": "relative/path/to/file.txt",
      "hash": "def456...",
      "size": 1024,
      "modified": "2025-01-15T14:20:00Z"
    }
  ],
  "tree": {
    "root": "abc123...",
    "nodes": [...]
  },
  "stats": {
    "total_files": 150,
    "total_size": 1048576,
    "skipped_files": 2
  }
}
```

**Deliverable**: `internal/tree/serializer.go` with round-trip tests

---

### Phase 7: Comparison Logic
**Test-Driven Development**:
1. **Test**: Compare identical trees, verify no changes reported
2. **Implement**: Basic tree comparison
3. **Test**: Detect added file (file in new tree, not in old)
4. **Implement**: Added files detection
5. **Test**: Detect modified file (same path, different hash)
6. **Implement**: Modified files detection
7. **Test**: Detect deleted file (file in old tree, not in new)
8. **Implement**: Deleted files detection
9. **Test**: Detailed change report formatting
10. **Implement**: Pretty-printed output with colors (optional)

**Algorithm**:
- Build map of old tree: `path → FileInfo`
- Build map of new tree: `path → FileInfo`
- Iterate new tree: if path not in old OR hash differs → added/modified
- Iterate old tree: if path not in new → deleted

**Deliverable**: `internal/compare/comparer.go` with comprehensive tests

---

### Phase 8: CLI Integration with Cobra
**Test-Driven Development**:
1. **Test**: Parse `generate` command with required flags
2. **Implement**: Cobra `generate` command structure
3. **Test**: Parse `compare` command with arguments
4. **Implement**: Cobra `compare` command structure
5. **Test**: Verify error on missing required arguments
6. **Implement**: Argument validation
7. **Test**: Verify --help output is clear
8. **Implement**: Command descriptions and examples
9. **Test**: Integration test: full generate → compare workflow
10. **Implement**: Wire all components together in commands

**Cobra Setup**:
```bash
# Initialize cobra
cobra-cli init
cobra-cli add generate
cobra-cli add compare
```

**Deliverable**: `cmd/merkle-go/main.go` with full CLI integration

---

### Phase 9: Error Handling & Logging
**Test-Driven Development**:
1. **Test**: Permission-denied file is skipped, logged, and reported in summary
2. **Implement**: Error collection during file walking and hashing
3. **Test**: Summary shows count and list of skipped files
4. **Implement**: Skipped files summary formatting
5. **Test**: Exit code 2 when errors occurred
6. **Implement**: Exit code logic based on results
7. **Test**: Verbose flag enables debug logging
8. **Implement**: slog level configuration

**Logging Strategy**:
```go
// Development (text, colored)
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: logLevel,
}))

// Production (JSON, structured)
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: logLevel,
}))

// Structured logging examples
logger.Info("file hashed",
    slog.String("path", filePath),
    slog.Int64("size", fileSize),
    slog.String("hash", hashHex),
    slog.Duration("elapsed", elapsed),
)
```

**Deliverable**: Comprehensive logging throughout application

---

### Phase 10: Performance Optimization & Benchmarks
**Test-Driven Development**:
1. **Test**: Benchmark file hashing (sequential vs concurrent)
2. **Implement**: Benchmark suite in `hasher_test.go`
3. **Test**: Benchmark worker pool with different sizes (1, 2, 4, 8, 16 workers)
4. **Implement**: Worker pool benchmarks
5. **Test**: Benchmark large directory (1000+ files)
6. **Implement**: End-to-end benchmark
7. **Optimize**: Based on profiling results (pprof)
8. **Test**: Memory usage is reasonable (no memory leaks)

**Benchmark Commands**:
```bash
# Run benchmarks
go test -bench=. -benchmem ./...

# Profile CPU
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Profile memory
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

**Performance Targets**:
- Hash 1000 files (100MB total) in < 1 second on modern hardware
- Memory usage < 100MB for 10,000 files
- Worker pool scaling: near-linear until I/O bottleneck

**Deliverable**: Optimized implementation with benchmark suite

---

## Key Technical Decisions

| Component | Decision | Rationale |
|-----------|----------|-----------|
| **CLI Framework** | Cobra | Industry standard, used by kubectl, hugo, gh. Rich features. |
| **Logging** | slog (stdlib) | Good performance, no external deps, future-proof |
| **Hashing** | cespare/xxhash/v2 | Fast (17GB/s), well-maintained, non-crypto |
| **Merkle Tree** | txaty/go-merkletree | High performance, parallel support, custom hash |
| **File Walking** | filepath.WalkDir | Efficient (uses DirEntry), standard library |
| **Concurrency** | Worker pool | Controlled resource usage, predictable performance |
| **Worker Count** | 2×CPU cores | I/O-bound workload benefits from more workers |
| **Testing** | Table-driven | Go community standard, covers edge cases |
| **Structure** | cmd/internal | Clean separation, testable, scalable |
| **Output Format** | JSON (human-readable) | Easy to debug, inspect, version |
| **Error Handling** | Skip with summary | Resilient, user gets full report at end |
| **Exclusions** | config.yml | Reusable across runs, version-controllable |

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        User (CLI)                            │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                 Cobra Commands                               │
│  ┌──────────────────┐         ┌──────────────────┐         │
│  │  generate        │         │  compare         │         │
│  └──────────────────┘         └──────────────────┘         │
└───────────────────────┬─────────────────┬───────────────────┘
                        │                 │
                        ▼                 ▼
┌─────────────────────────────────────────────────────────────┐
│                  Config Parser (config.yml)                  │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│              File Walker (filepath.WalkDir)                  │
│              with Exclusion Pattern Matching                 │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
              ┌─────────────────┐
              │  File Paths     │
              │  (buffered chan)│
              └────────┬─────────┘
                        │
        ┌───────────────┼───────────────┬─────────────┐
        ▼               ▼               ▼             ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ...
│ Worker 1     │ │ Worker 2     │ │ Worker N     │
│ (xxHash file)│ │ (xxHash file)│ │ (xxHash file)│
└──────┬───────┘ └──────┬───────┘ └──────┬───────┘
        │               │               │
        └───────────────┼───────────────┘
                        │
                        ▼
              ┌─────────────────┐
              │  Hash Results   │
              │  (buffered chan)│
              └────────┬─────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│          Merkle Tree Builder (go-merkletree)                 │
│          with Parallel Processing Enabled                    │
└───────────────────────┬─────────────────────────────────────┘
                        │
        ┌───────────────┴───────────────┐
        ▼                               ▼
┌──────────────────┐          ┌──────────────────┐
│ JSON Serializer  │          │  Tree Comparer   │
│ (save to file)   │          │  (detect changes)│
└──────────────────┘          └────────┬─────────┘
                                        │
                                        ▼
                              ┌──────────────────┐
                              │ Detailed Report  │
                              │ (stdout + exit)  │
                              └──────────────────┘
                                        │
                                        ▼
                              ┌──────────────────┐
                              │  slog Logger     │
                              │  (structured)    │
                              └──────────────────┘
```

## Success Criteria

- ✅ All tests pass before implementation (TDD)
- ✅ Test coverage > 80% for critical paths
- ✅ Concurrent processing demonstrably faster than sequential (benchmarks)
- ✅ Accurate change detection (zero false positives/negatives)
- ✅ Graceful handling of permission errors (skip and report)
- ✅ Clear, actionable output for users
- ✅ Performance: Hash 1000 files in < 1 second
- ✅ Memory efficient: < 100MB for 10,000 files
- ✅ Zero panics or crashes in error scenarios
- ✅ CLI help text is comprehensive and clear

## Development Workflow

### Initial Setup
```bash
# Initialize Go module
go mod init merkle-go

# Install dependencies
go get github.com/spf13/cobra@latest
go get github.com/cespare/xxhash/v2@latest
go get github.com/txaty/go-merkletree@latest
go get gopkg.in/yaml.v3@latest

# Initialize Cobra CLI
go install github.com/spf13/cobra-cli@latest
cobra-cli init
cobra-cli add generate
cobra-cli add compare
```

### TDD Cycle
```bash
# 1. Write test
vim internal/hash/hasher_test.go

# 2. Run test (should fail)
go test ./internal/hash/... -v

# 3. Implement minimum code to pass
vim internal/hash/hasher.go

# 4. Run test (should pass)
go test ./internal/hash/... -v

# 5. Refactor if needed

# 6. Run all tests
go test ./... -v
```

### Continuous Testing
```bash
# Run tests on file change (using entr or similar)
fd -e go | entr -c go test ./... -v

# Run with coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Performance Testing
```bash
# Run benchmarks
go test -bench=. -benchmem ./...

# CPU profiling
go test -cpuprofile=cpu.prof -bench=. ./internal/hash/...
go tool pprof -http=:8080 cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=. ./internal/hash/...
go tool pprof -http=:8080 mem.prof
```

### Building
```bash
# Build binary
go build -o bin/merkle-go ./cmd/merkle-go

# Build with optimizations
go build -ldflags="-s -w" -o bin/merkle-go ./cmd/merkle-go

# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 go build -o bin/merkle-go-linux ./cmd/merkle-go
GOOS=darwin GOARCH=arm64 go build -o bin/merkle-go-macos ./cmd/merkle-go
GOOS=windows GOARCH=amd64 go build -o bin/merkle-go.exe ./cmd/merkle-go
```

## Example Usage

### Generate merkle tree
```bash
# Basic usage
./merkle-go generate /path/to/directory -o tree.json

# With custom config and verbose logging
./merkle-go generate /path/to/directory -o tree.json -c custom-config.yml -v

# With more workers
./merkle-go generate /path/to/directory -o tree.json -w 16

# With JSON logging (for production)
./merkle-go generate /path/to/directory -o tree.json --json-log
```

### Compare trees
```bash
# Basic comparison
./merkle-go compare tree.json /path/to/directory

# Check exit code (in scripts)
./merkle-go compare tree.json /path/to/directory
echo $?  # 0=no changes, 1=changes, 2=errors

# With verbose logging
./merkle-go compare tree.json /path/to/directory -v
```

### Example config.yml
```yaml
exclude:
  # Version control
  - ".git/"
  - ".svn/"

  # Dependencies
  - "node_modules/"
  - "vendor/"
  - "__pycache__/"

  # Build artifacts
  - "*.o"
  - "*.so"
  - "*.exe"
  - "bin/"
  - "dist/"

  # Temporary files
  - "*.tmp"
  - "*.swp"
  - "*.log"
  - ".DS_Store"
  - "Thumbs.db"
```

## Future Enhancements (Post-MVP)

- Support for multiple hash algorithms (SHA256, BLAKE3)
- Incremental updates (update tree instead of full rebuild)
- Streaming JSON for very large trees
- Compression support for output files
- Watch mode (continuous monitoring)
- Merkle proof generation for individual files
- Integration with cloud storage (S3, GCS)
- Web UI for visualizing trees and changes
- Signing/verification of merkle roots
- Multi-root comparison (compare multiple snapshots)

---

## Timeline Estimate

Based on TDD approach with proper testing:

- **Phase 1-2** (Config + Hashing): 4-6 hours
- **Phase 3-4** (Walking + Workers): 6-8 hours
- **Phase 5-6** (Tree + Serialization): 6-8 hours
- **Phase 7-8** (Comparison + CLI): 6-8 hours
- **Phase 9-10** (Error handling + Optimization): 4-6 hours

**Total**: 26-36 hours of focused development

This assumes:
- Familiarity with Go
- Test-first discipline
- No major blockers
- Reasonable test coverage goals
