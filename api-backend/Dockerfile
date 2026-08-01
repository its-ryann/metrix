# Stage 1: Build static Go binary
FROM golang:alpine AS builder

WORKDIR /app

# Enable Go to download the toolchain version declared in go.mod if needed
ENV GOTOOLCHAIN=auto

# Copy go.mod first for layer caching
COPY go.mod ./
RUN go mod download || true

# Copy all source files
COPY . .

# Build from cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/server/main.go

# Stage 2: Production runtime image
FROM alpine:3.19

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/server .

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

ENTRYPOINT ["./server"]