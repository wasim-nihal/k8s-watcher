# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Build arguments
ARG VERSION
ARG COMMIT
ARG DATE

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build VERSION=${VERSION} COMMIT=${COMMIT} DATE=${DATE}

# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy the binary from builder
COPY --from=builder /app/build/k8s-watcher .

# Set environment variables
ENV PATH="/app:${PATH}"

# Default command
ENTRYPOINT ["/app/k8s-watcher"]
