FROM golang:1.22-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc openssl libssl-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN ./scripts/gen_certs.sh
RUN CGO_ENABLED=1 GOOS=linux go build -a -o forum ./cmd/server

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/forum .
COPY --from=builder /app/web ./web
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/certs ./certs

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates sqlite3 \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /app/data /app/uploads

EXPOSE 8080 8443

CMD ["./forum"]
