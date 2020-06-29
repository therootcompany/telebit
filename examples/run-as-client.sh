#!/bin/bash

set -e
set -u

go generate -mod=vendor ./...
go build -mod=vendor -o telebit cmd/telebit/*.go

source .env

ACME_RELAY_BASEURL=${ACME_RELAY_BASEURL:-"https://devices.examples.com"}
AUTH_BASEURL=${AUTH_BASEURL:-"https://devices.examples.com"}
CLIENT_SECRET=${CLIENT_SECRET:-"yyyyyyyyyyyyyyyy"}

./telebit --acme-agree=true \
    --acme-relay $ACME_RELAY_BASEURL/api \
    --auth-url $AUTH_BASEURL/api \
    --app-id test-id --secret "$CLIENT_SECRET"
