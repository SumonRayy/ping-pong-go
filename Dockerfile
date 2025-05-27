# Use a multi-stage build to compile the Go application
FROM golang:1.22.9 AS builder

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o ping-pong-go .

# Use a minimal image to run the application
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/ping-pong-go .

# Expose the port the app runs on
EXPOSE 8080

# Command to run the application
CMD ["./ping-pong-go"] 