BINARY    := esquisse-mcp
GOFLAGS   :=

.PHONY: all build test test-race lint fmt tidy install clean

all: build

build:
	go build $(GOFLAGS) -o $(BINARY) .

test:
	go test -count=1 -timeout 60s ./...

test-race:
	go test -count=1 -race -timeout 120s ./...

lint:
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, falling back to go vet"; \
		go vet ./...; \
	fi

fmt:
	gofmt -w .

tidy:
	go mod tidy

install:
	go install .

clean:
	rm -f $(BINARY)
