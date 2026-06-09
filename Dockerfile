# Stage 1: Build UI
FROM node:20-alpine AS ui-builder
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# Stage 2: Build Go server
FROM golang:1.26-alpine AS server-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui-builder /app/web/dist ./internal/dashboard/dist
RUN CGO_ENABLED=0 go build -buildvcs=false -o /go-database ./cmd/server/

# Stage 3: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=server-builder /go-database .
COPY config/config.yaml ./config/config.yaml
COPY database/ ./database/

EXPOSE 8080
CMD ["./go-database"]
