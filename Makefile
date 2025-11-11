.PHONY: build clean test all

all: build

build:
	go build -o bin/merkle-go ./cmd/merkle-go

clean:
	rm -rf bin/

test:
	go test ./...
