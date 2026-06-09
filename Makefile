# go-database Makefile

.PHONY: all build build-ui build-server run test clean

all: build

# Build UI (React)
build-ui:
	cd web && npm install && npm run build

# Copy UI to embed directory
copy-ui:
	cp -r web/dist internal/dashboard/dist

# Build Go server
build-server: copy-ui
	go build -buildvcs=false -o bin/go-database ./cmd/server/

# Quick build (Go only, no UI)
build-server-quick:
	go build -buildvcs=false -o bin/go-database ./cmd/server/

# Build both
build: build-ui build-server

# Run server (requires config/config.yaml)
run:
	./bin/go-database

# Test Go code
test:
	go test -buildvcs=false -v -count=1 ./internal/...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf web/dist/
	rm -rf internal/dashboard/dist/
	rm -rf database/internal/*.db
