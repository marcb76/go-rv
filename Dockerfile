# Build stage
FROM golang:1.27-alpine AS builder

# Copy dependencies
WORKDIR /app
COPY go.mod ./
# If you add go.sum in the future, uncomment the following line:
# COPY go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Compile a static binary for Linux (CGO disabled)
RUN CGO_ENABLED=0 GOOS=linux go build -o go-rv .




# --- Runtime stage (ultra-light final image) ---
FROM alpine:latest

# Copy the compiled binary from the previous stage
WORKDIR /app
COPY --from=builder /app/go-rv .

# Expose the ports your application will use (e.g., HTTP and HTTPS)
EXPOSE 80 443 8080

# Command to run the binary
CMD ["./go-rv"]
