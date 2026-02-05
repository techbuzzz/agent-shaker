# Stage 1: Build Frontend (Vue.js)
FROM node:18-alpine AS frontend-builder

WORKDIR /app

# Copy frontend package files
COPY web/package*.json ./
RUN npm install

# Copy frontend source
COPY web/ ./

# Build frontend for production
RUN npm run build

# Verify build output
RUN ls -la /app/dist || echo "ERROR: dist not found!"

# Stage 2: Build Backend (Go)
FROM golang:1.24-alpine AS backend-builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /mcp-server ./cmd/server

# Stage 3: Final production image
FROM alpine:latest

WORKDIR /app

# Copy backend binary
COPY --from=backend-builder /mcp-server .

# Copy migrations
COPY migrations ./migrations

# Copy documentation files
COPY *.md ./
COPY docs ./docs

# Copy frontend build (static files)
COPY --from=frontend-builder /app/dist ./web/dist

EXPOSE 8080

CMD ["./mcp-server"]
