.PHONY: build test vet generate clean

build:
	mkdir -p bin
	go build -o bin/cmc-lsp ./cmd/cmc-lsp
	go build -o bin/cmc-fmt ./cmd/cmc-fmt
	go build -o bin/cmc-check ./cmd/cmc-check

test:
	go test ./...

vet:
	go vet ./...

generate:
	go generate ./...

clean:
	rm -rf bin
