# Stage 1: Build static Go binary
FROM golang:1.25-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /src/bin/beaverish ./cmd/agent

# Stage 2: Minimal runtime image
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -u 10001 beaverish

COPY --from=builder /src/bin/beaverish /usr/local/bin/beaverish

USER beaverish
WORKDIR /home/beaverish

ENTRYPOINT ["/usr/local/bin/beaverish"]
CMD ["--mcp"]
