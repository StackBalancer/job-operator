# Use a Go base image to build the task-job service
FROM golang:1.23-alpine as go-builder

# Set the working directory in the container
WORKDIR /app

# Copy the source code into the container
COPY . .

# Build the Go binary
RUN apk add --no-cache -t build-tools curl git && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o task-job main.go && \
    apk del build-tools && \
    rm -rf /var/cache/apk/*

# Use a minimal Alpine image to run the service
FROM alpine:latest

# Install CA certificates (needed for HTTPS requests)
RUN apk --no-cache add ca-certificates bash curl

# Set the working directory
WORKDIR /root/

# Copy the Go binary from the builder stage
COPY --from=go-builder /app/task-job .

# Expose the port that the service will run on
EXPOSE 8080

# Start the task-job service when the container starts
CMD ["./task-job"]
