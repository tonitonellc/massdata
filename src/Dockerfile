# Copied from toninoble.com project
# https://github.com/dethmasque/toninobledotcom

# Stage 1: Build the Go application
FROM golang:1.25-bookworm AS builder

# Set the working directory to where the Go source code lives
WORKDIR /app

# Copy dependency files first to leverage Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the Go source code (api, models, types, main.go, etc.)
COPY . ./

# Build the binary named 'server'
RUN go build -v -o server ./cmd/server

# Stage 2: Create the minimal runtime image
FROM debian:bookworm-slim

# Install CA certificates for secure API proxying
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/server ./server

# Ensure the binary has execution permissions
RUN chmod +x ./server

# Cloud Run will inject the PORT env var automatically.
EXPOSE 8080

# Run the application
CMD ["./server"]

