#!/bin/bash

set -e
set -u

go generate -mod=vendor ./...
go build -mod=vendor -o telebit cmd/telebit/*.go

source .env

ADMIN_HOSTNAME=${ADMIN_HOSTNAME:-"devices.example.com"}
AUTH_BASEURL=${AUTH_BASEURL:-"https://devices.example.com"}
SECRET=${SECRET:-"xxxxxxxxxxxxxxxx"}

./telebit --acme-agree=true \
    --admin-hostname $ADMIN_HOSTNAME \
    --auth-url $AUTH_BASEURL/api \
    --secret "$SECRET" \
    --listen 3020,3030
