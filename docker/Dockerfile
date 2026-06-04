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

RUN apk add --no-cache ca-certificates sqlite

WORKDIR /app

COPY --from=builder /app/forum .
COPY --from=builder /app/web ./web
COPY --from=builder /app/migrations ./migrations

RUN mkdir -p /app/data /app/uploads /app/certs

EXPOSE 8080 8443

CMD ["./forum"]
