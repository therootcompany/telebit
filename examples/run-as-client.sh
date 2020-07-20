#!/bin/bash

set -e
set -u

source .env

#go generate -mod=vendor ./...
VENDOR_ID="${VENDOR_ID:-"${VENDOR_ID:-"test-id"}"}"
CLIENT_SECRET="${CLIENT_SECRET:-}"
go build -mod=vendor -o ./telebit \
    -ldflags="-X 'main.VendorID=$VENDOR_ID' -X 'main.ClientSecret=$CLIENT_SECRET'" \
    cmd/telebit/*.go
#go build -mod=vendor -o telebit \
#    cmd/telebit/*.go

# For Device Authorization across services
AUTH_URL=${AUTH_URL:-"https://devices.examples.com/api"}
VENDOR_ID="$VENDOR_ID"
SECRET="${CLIENT_SECRET:-"xxxxxxxxxxxxxxxx"}"
#CLIENT_SECRET=${CLIENT_SECRET:-"yyyyyyyyyyyyyyyy"}
LOCALS="${LOCALS:-"https:newbie.devices.examples.com:3000,http:newbie.devices.examples.com:3000"}"

# For the Remote Server (Tunnel Client)
TUNNEL_RELAY_URL=${TUNNEL_RELAY_URL:-"wss://devices.example.com"}
LISTEN=":3080"

# For Let's Encrypt / ACME registration
ACME_AGREE=${ACME_AGREE:-}
ACME_EMAIL=${ACME_EMAIL:-"me@example.com"}

# For Let's Encrypt / ACME challenges
ACME_RELAY_URL=${ACME_RELAY_URL:-"https://devices.examples.com/api/dns"}

VERBOSE=${VERBOSE:-}
VERBOSE_BYTES=${VERBOSE_BYTES:-}
VERBOSE_RAW=${VERBOSE_RAW:-}


./telebit \
    --auth-url $AUTH_URL \
    --vendor-id "$VENDOR_ID" \
    --secret "$CLIENT_SECRET" \
    --tunnel-relay-url $TUNNEL_RELAY_URL \
    --listen "$LISTEN" \
    --locals "$LOCALS" \
    --acme-agree=${ACME_AGREE} \
    --acme-email "$ACME_EMAIL" \
    --acme-relay-url $ACME_RELAY_URL \
    --verbose=$VERBOSE

#    --subject "$CLIENT_SUBJECT" \

#PORT_FORWARDS=3443:3001,8443:3002
