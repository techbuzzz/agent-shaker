FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o mcp-tracker .

# Final stage
FROM alpine:latest

# Update ca-certificates first
RUN apk update && apk --no-cache add ca-certificates sqlite-libs || \
    (apk update && apk --no-cache add ca-certificates sqlite-libs)

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/mcp-tracker .

# Create data directory
RUN mkdir -p /root/data

# Expose port
EXPOSE 8080

# Run the application
CMD ["./mcp-tracker"]
