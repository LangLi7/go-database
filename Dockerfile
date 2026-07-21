# Build Go server (API-only, no embedded frontend)
FROM golang:1.26-alpine AS server-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -buildvcs=false -o /go-database ./cmd/server/

# Build MCP stdio server
RUN CGO_ENABLED=0 go build -buildvcs=false -o /go-database-mcp ./cmd/mcp/

# Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl
WORKDIR /app
COPY --from=server-builder /go-database .
COPY --from=server-builder /go-database-mcp .
COPY config/config.yaml ./config/config.yaml
COPY database/ ./database/
# models/ NOT copied into image (multi-GB); mount via volume for llamacpp — see docker-compose mcp service

EXPOSE 8080

# Healthcheck (requires server to be running)
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD curl -sf http://localhost:8080/health || exit 1

CMD ["./go-database"]
