.PHONY: build test lint docs man clean

BINARY := witr
CMD    := ./cmd/witr

build:
	CGO_ENABLED=0 go build -o $(BINARY) $(CMD)

test:
	go test ./...

test-race:
	go test -race ./...

lint:
	@test -z "$$(find . -name '*.go' -not -path './vendor/*' | xargs gofmt -l)" || { echo 'Unformatted Go files (run gofmt -w):'; find . -name '*.go' -not -path './vendor/*' | xargs gofmt -l; exit 1; }
	go vet ./...

docs: man markdown

man:
	go run ./internal/tools/docgen -format man -out docs/cli

markdown:
	go run ./internal/tools/docgen -format markdown -out docs/cli

clean:
	rm -f $(BINARY)
