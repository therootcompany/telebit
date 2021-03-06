#!/bin/bash

set -e
set -u

source .env

#go generate -mod=vendor ./...
VENDOR_ID="${VENDOR_ID:-"${VENDOR_ID:-"test-id"}"}"
CLIENT_SECRET="${CLIENT_SECRET:-}"
#go build -mod=vendor -o ./telebit \
#    -ldflags="-X 'main.VendorID=$VENDOR_ID' -X 'main.ClientSecret=$CLIENT_SECRET' -X 'main.serviceName=telebit' -X 'main.serviceDesc=securely tunnel through telebit.io'" \
#    cmd/telebit/*.go
pushd cmd/telebit
    go build -mod=vendor -o telebit .
popd

# For Device Authorization across services
#AUTH_URL=${AUTH_URL:-"https://devices.examples.com/api"}
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
    --vendor-id "$VENDOR_ID" \
    --secret "$CLIENT_SECRET" \
    --tunnel-relay-url $TUNNEL_RELAY_URL \
    --listen "$LISTEN" \
    --tls-locals "$TLS_LOCALS" \
    --locals "$LOCALS" \
    --acme-agree=${ACME_AGREE} \
    --acme-email "$ACME_EMAIL" \
    --verbose=$VERBOSE

#    --auth-url $AUTH_URL \
#    --acme-relay-url $ACME_RELAY_URL \
#    --subject "$CLIENT_SUBJECT" \

#PORT_FORWARDS=3443:3001,8443:3002
