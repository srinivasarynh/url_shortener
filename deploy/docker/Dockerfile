FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install required packages
RUN apk add --no-cache git ca-certificates tzdata && \
    update-ca-certificates

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /go/bin/url-shortener ./cmd/api

# Create a minimal production image
FROM scratch

# Copy timezone info
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder
COPY --from=builder /go/bin/url-shortener /usr/local/bin/url-shortener

# Set the entrypoint
ENTRYPOINT ["/usr/local/bin/url-shortener"]
