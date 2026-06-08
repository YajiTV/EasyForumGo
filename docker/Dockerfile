# Stage 1 - Build
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -o forum ./cmd/server

# Stage 2 - Run
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/forum .
COPY --from=builder /app/web ./web
COPY --from=builder /app/migrations ./migrations

RUN apk add --no-cache ca-certificates sqlite openssl \
    && mkdir -p /app/data /app/uploads /app/certs \
    && openssl req -x509 -newkey rsa:2048 -nodes \
        -keyout /app/certs/key.pem \
        -out /app/certs/cert.pem \
        -days 365 \
        -subj "/CN=localhost" \
        -addext "subjectAltName=DNS:localhost,IP:127.0.0.1" \
    && apk del openssl

EXPOSE 8080 8443

CMD ["./forum"]
