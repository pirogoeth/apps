# Multi-stage Dockerfile for docker-hermes
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /src

# Copy source code
COPY . .

WORKDIR /src/docker-hermes

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o docker-hermes .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates docker-cli

# Create non-root user
RUN addgroup -g 1001 -S docker-hermes && \
    adduser -u 1001 -S docker-hermes -G docker-hermes

# Copy binary from builder
COPY --from=builder /src/docker-hermes/docker-hermes /usr/local/bin/docker-hermes

# Create config directory
RUN mkdir -p /etc/docker-hermes && \
    chown docker-hermes:docker-hermes /etc/docker-hermes

# Switch to non-root user
USER docker-hermes

# Expose metrics port
EXPOSE 9090

# Default command
ENTRYPOINT ["docker-hermes"]
