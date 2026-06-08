#!/bin/sh
set -e

CERTS_DIR="$(dirname "$0")/../certs"
mkdir -p "$CERTS_DIR"

openssl req -x509 \
  -newkey rsa:4096 \
  -keyout "$CERTS_DIR/key.pem" \
  -out "$CERTS_DIR/cert.pem" \
  -days 365 \
  -nodes \
  -subj "/C=FR/ST=Paris/L=Paris/O=Easy/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

echo "Certificates generated in $CERTS_DIR"
echo "cert.pem and key.pem are valid for 365 days"
