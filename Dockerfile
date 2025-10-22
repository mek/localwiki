# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_CFLAGS="-D_LARGEFILE64_SOURCE=1" CGO_ENABLED=1 go build -a -installsuffix cgo -o wiki .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache sqlite ca-certificates tzdata tcl

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/wiki .

# dac file
COPY ./dac /usr/bin/dac

# Copy static files
COPY static/ ./static/

# Create data directory
RUN mkdir -p /app/data

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/pages || exit 1

# Run the application
CMD ["./wiki"]
