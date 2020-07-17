#!/bin/bash

set -e
set -u

go generate -mod=vendor ./...
go build -mod=vendor -o telebit cmd/telebit/*.go

source .env

ACME_RELAY_URL=${ACME_RELAY_URL:-"https://devices.examples.com"}
AUTH_URL=${AUTH_URL:-"https://devices.examples.com"}
CLIENT_SECRET=${CLIENT_SECRET:-"yyyyyyyyyyyyyyyy"}

./telebit --acme-agree=true \
    --acme-relay-url $ACME_RELAY_URL/api \
    --auth-url $AUTH_URL/api \
    --app-id test-id --secret "$CLIENT_SECRET"
