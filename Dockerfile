# Use the official Go image as the base image for building
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Use a minimal base image for the final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create a non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Copy the templates directory
COPY --from=builder /app/templates ./templates

# Copy the site directory
COPY --from=builder /app/site ./site

# Change ownership to the non-root user
RUN chown -R appuser:appgroup /root

# Switch to non-root user
USER appuser

# Expose the port that the application will run on
EXPOSE 8080

# Set environment variable for port (Cloud Run uses PORT env var)
ENV PORT=8080
ENV SITE_DIR=./site

# Command to run the application
CMD ["./main"]
