FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /mcp-server ./cmd/server

FROM alpine:latest

WORKDIR /app

# Copy binary
COPY --from=builder /mcp-server .

# Copy migrations and static files
COPY migrations ./migrations
COPY web ./web

EXPOSE 8080

CMD ["./mcp-server"]
