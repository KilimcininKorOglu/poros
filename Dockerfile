# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o poros ./cmd/poros

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Create non-root user (note: raw sockets still need capabilities)
RUN adduser -D -u 1000 poros

# Copy binary from builder
COPY --from=builder /build/poros /usr/local/bin/poros

# Set capabilities for raw socket access
# Note: This requires --cap-add=NET_RAW when running the container
RUN chmod +x /usr/local/bin/poros

# Use non-root user
USER poros

# Default command
ENTRYPOINT ["poros"]
CMD ["--help"]
