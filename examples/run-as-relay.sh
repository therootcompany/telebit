#!/bin/bash

set -e
set -u

#go generate -mod=vendor ./...
go build -mod=vendor -o telebit cmd/telebit/*.go

source .env

ADMIN_HOSTNAME=${ADMIN_HOSTNAME:-"devices.example.com"}
AUTH_BASEURL=${AUTH_BASEURL:-"https://devices.example.com"}
AUTH_URL=${AUTH_URL:-"$AUTH_BASEURL/api"}
SECRET=${SECRET:-"xxxxxxxxxxxxxxxx"}
ACME_EMAIL="${ACME_EMAIL:-}"

./telebit --acme-agree=true \
    --admin-hostname $ADMIN_HOSTNAME \
    --auth-url $AUTH_URL/api \
    --acme-email "$ACME_EMAIL" \
    --secret "$SECRET" \
    --listen 3020,3030
