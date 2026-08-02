# Stage 1: Build Stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod ./
# COPY go.sum ./ (Uncomment when you add dependencies)
RUN go mod download

# Copy source code and build static binary for Linux
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/mc-daemon ./cmd/daemon

# Stage 2: Minimal Runtime Stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/mc-daemon /app/mc-daemon

EXPOSE 8080

ENTRYPOINT ["/app/mc-daemon"]