# Build Go server (API-only, no embedded frontend)
FROM golang:1.26-alpine AS server-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -buildvcs=false -o /go-database ./cmd/server/

# Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=server-builder /go-database .
COPY config/config.yaml ./config/config.yaml
COPY database/ ./database/

EXPOSE 8080
CMD ["./go-database"]
