# Build Go server (API-only, no embedded frontend)
FROM golang:1.25-alpine AS server-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -buildvcs=false -o /go-database ./cmd/server/
RUN CGO_ENABLED=0 go build -buildvcs=false -o /go-database-mcp ./cmd/mcp/

# Build llama-server from source (static CPU build, runs on alpine/musl).
# Uses master (supports recent architectures like qwen3.5/ornith). Pin to a
# tag via --build-arg LLAMA_TAG=<tag> for reproducibility. go-database
# (mcp.provider=llamacpp + auto_start=true) launches this itself.
FROM alpine:3.20 AS llama-builder
ARG LLAMA_TAG=master
RUN apk add --no-cache git cmake gcc g++ make linux-headers
WORKDIR /tmp
RUN git clone --depth 1 --branch ${LLAMA_TAG} https://github.com/ggml-org/llama.cpp.git
RUN cd llama.cpp && cmake -B build -DGGML_CPU=ON -DLLAMA_CURL=OFF -DBUILD_SHARED_LIBS=OFF -DCMAKE_BUILD_TYPE=Release && \
    cmake --build build --target llama-server -j"$(nproc)"


# Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl libstdc++ gcc
WORKDIR /app
COPY --from=server-builder /go-database .
COPY --from=server-builder /go-database-mcp .
COPY --from=llama-builder /tmp/llama.cpp/build/bin/llama-server /usr/local/bin/llama-server
RUN chmod +x /usr/local/bin/llama-server
COPY config/config.yaml ./config/config.yaml
COPY database/ ./database/
# models/ NOT copied into image (multi-GB); mount via volume for llamacpp — see docker-compose mcp service

EXPOSE 8080

# Healthcheck (requires server to be running)
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD curl -sf http://localhost:8080/health || exit 1

CMD ["./go-database"]
