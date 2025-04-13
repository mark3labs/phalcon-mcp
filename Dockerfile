FROM golang:1.23 AS builder

WORKDIR /app

# Copy go.mod and go.sum files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o phalcon-mcp -a -ldflags '-extldflags "-static"' .

# Use scratch as the final base image
FROM scratch

# Copy the binary from the builder stage
COPY --from=builder /app/phalcon-mcp /phalcon-mcp

# Set the entrypoint to the binary
ENTRYPOINT ["/phalcon-mcp"]
