# merkle-go

A Go CLI application that generates merkle trees of file hashes from directory trees and compares them to detect changes. Optimized for speed using xxHash and concurrent file processing.

## Features

- Fast file hashing using xxHash (non-cryptographic)
- Concurrent file processing with worker pools
- Merkle tree construction with parallel processing
- Directory change detection
- Configurable exclusion patterns
- JSON output format

## Installation

```bash
go build -o bin/merkle-go ./cmd/merkle-go
```

## Usage

### Generate merkle tree

```bash
./merkle-go generate <directory> -o <output.json>
```

### Compare trees

```bash
./merkle-go compare <output.json> <directory>
```

## Configuration

Create a `config.yml` file to specify exclusion patterns:

```yaml
exclude:
  - ".git/"
  - "node_modules/"
  - "*.tmp"
  - "*.log"
```

## Development Status

🚧 **Work in Progress** - This project is under active development.

See [PLAN.md](PLAN.md) for the detailed implementation plan.

## Dependencies

- [github.com/spf13/cobra](https://github.com/spf13/cobra) - CLI framework
- [github.com/cespare/xxhash/v2](https://github.com/cespare/xxhash) - Fast hashing
- [github.com/txaty/go-merkletree](https://github.com/txaty/go-merkletree) - Merkle tree implementation
- [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) - YAML parsing

## License

MIT
