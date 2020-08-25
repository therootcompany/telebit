#!/bin/bash

set -e
set -u

go mod tidy
go mod vendor
go generate -mod=vendor ./...
go build -mod=vendor -o ./telebit ./cmd/telebit/*.go
if [ -n "$(command -v setcap)" ]; then
    sudo setcap 'cap_net_bind_service=+ep' ./telebit
fi

source .env

SPF_HOSTNAME="${SPF_HOSTNAME:-""}"
#SPF_HOSTNAME="_allowed.example.com"

# For Tunnel Relay Server
API_HOSTNAME=${API_HOSTNAME:-"devices.example.com"}
LISTEN="${LISTEN:-":80 :443"}"

# For Device Management & Authentication
AUTH_URL=${AUTH_URL:-"https://devices.example.com/api"}

# For Let's Encrypt / ACME challenges
ACME_RELAY_URL=${ACME_RELAY_URL:-"http://localhost:4200/api/dns"}
SECRET=${SECRET:-"xxxxxxxxxxxxxxxx"}

# For Let's Encrypt / ACME registration
ACME_AGREE=${ACME_AGREE:-}
ACME_EMAIL="${ACME_EMAIL:-}"

./telebit \
    --spf-domain $SPF_HOSTNAME \
    --api-hostname $API_HOSTNAME \
    --auth-url $AUTH_URL \
    --acme-agree "$ACME_AGREE" \
    --acme-email "$ACME_EMAIL" \
    --acme-relay-url "$ACME_RELAY_URL" \
    --secret "$SECRET" \
    --listen "$LISTEN"
