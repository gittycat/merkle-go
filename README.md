# merkle-go

Fast directory integrity verification using Merkle trees with xxHash and concurrent file processing.

## Installation

```bash
make build
```

Or manually:
```bash
go build -o bin/merkle-go ./cmd/merkle-go
```

## Usage

### Generate merkle tree

```bash
# Default output: ./output/<root-hash>.json
go run ./cmd/merkle-go <directory>

# Custom output file
go run ./cmd/merkle-go <directory> <output.json>
```

**Example output:**
```
Scanning directory: /Users/name/project
Found 150 files
Hashing files...
[████████████████████████████░░░░░░░░░░░░░░░░░░░░░░]  56% (84/150) | internal, cmd, pkg
✓ Merkle tree generated successfully
  Root hash: a1b2c3d4e5f6a7b8
  Files: 150
  Output: output/a1b2c3d4e5f6a7b8.json
```

### Compare trees

```bash
go run ./cmd/merkle-go compare <tree.json> <directory>
```

**Example output:**
```
Loaded saved tree (root: a1b2c3d4e5f6...)
Scanning directory: /Users/name/project
Found 150 files
Hashing files...
[██████████████████████████████████████████████████] 100% (150/150)
Changes detected:

ADDED (2 files):
  + src/new-feature.go (hash: abc123..., size: 1024 bytes)
  + tests/new-test.go (hash: def456..., size: 512 bytes)

MODIFIED (1 file):
  ~ README.md
    Old: hash=789ghi..., size=2048 bytes, modified=2025-01-15
    New: hash=012jkl..., size=2100 bytes, modified=2025-01-16

Summary: 2 added, 1 modified, 0 deleted
```

**Exit codes:**
- `0` - No changes detected
- `1` - Changes detected
- `2` - Errors occurred during processing

## Configuration

Create `config.toml` to specify skip patterns and output file:

```toml
# List of files or directories to skip
skip = [
  ".git/",
  "node_modules/",
  "*.tmp",
  "*.log",
]

# Output file path (optional - defaults to ./output/<root-hash>.json)
output_file = ""
```

## Flags

Both commands support:
- `-c, --config` - Config file path (default: `config.toml`)
- `-w, --workers` - Worker goroutines (default: 2×CPU cores)

## Dependencies

- [github.com/cespare/xxhash/v2](https://github.com/cespare/xxhash) - Fast hashing
- [github.com/pelletier/go-toml/v2](https://github.com/pelletier/go-toml) - TOML parsing

## License

MIT
