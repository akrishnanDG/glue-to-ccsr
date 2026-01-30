# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install git for go mod download
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /glue-to-ccsr ./cmd/glue-to-ccsr

# Runtime stage
FROM alpine:3.19

# Add ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /glue-to-ccsr /usr/local/bin/glue-to-ccsr

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

# Set entrypoint
ENTRYPOINT ["glue-to-ccsr"]
CMD ["--help"]
