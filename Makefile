.PHONY: all build build-all clean fmt lint vet test tidy

all: build

# ── Build Go server only (no frontend embedded) ──
build:
	CGO_ENABLED=0 go build -o bin/go-database ./cmd/server/

# ── Cross-compile Go backend ──
build-all:
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o bin/go-database-linux-amd64   ./cmd/server/
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -o bin/go-database-linux-arm64   ./cmd/server/
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/go-database-windows-amd64 ./cmd/server/
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -o bin/go-database-darwin-amd64  ./cmd/server/

# ── Format all Go files ──
fmt:
	gofmt -w .

# ── Vet (static analysis) ──
vet:
	go vet ./...

# ── Lint (requires golangci-lint) ──
lint:
	golangci-lint run ./...

# ── Test ──
test:
	go test -count=1 ./...

# ── Tidy go.mod ──
tidy:
	go mod tidy

# ── Clean build artifacts ──
clean:
	rm -rf bin/
