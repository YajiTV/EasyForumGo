FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev openssl

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN ./scripts/gen_certs.sh
RUN CGO_ENABLED=1 GOOS=linux go build -a -o forum ./cmd/server

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/forum .
COPY --from=builder /app/web ./web
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/certs ./certs

RUN apk add --no-cache ca-certificates sqlite \
    && mkdir -p /app/data /app/uploads

EXPOSE 8080 8443

CMD ["./forum"]
