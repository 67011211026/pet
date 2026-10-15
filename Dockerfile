# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Install ca-certificates and git
RUN apk add --no-cache git ca-certificates

# Copy go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build pure static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/server

# Final runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

# Copy compiled binary and web assets
COPY --from=builder /app/server .
COPY --from=builder /app/templates ./templates

ENV PORT=8080
ENV GIN_MODE=release

EXPOSE 8080

CMD ["./server"]
