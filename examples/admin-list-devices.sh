#!/bin/bash

set -e
set -u

source .env
TUNNEL_RELAY_API="${TUNNEL_RELAY_API:-"https://devices.example.com/api"}"

echo "RELAY_SECRET: $RELAY_SECRET"
TOKEN=$(go run cmd/signjwt/*.go \
    --vendor-id "$VENDOR_ID" \
    --secret "$RELAY_SECRET" \
    --machine-ppid "$RELAY_SECRET"
)
echo "ADMIN TOKEN: '$TOKEN'"

echo "Auth URL: $TUNNEL_RELAY_API"
curl "$TUNNEL_RELAY_API/subscribers" -H "Authorization: Bearer ${TOKEN}"
curl "$TUNNEL_RELAY_API/subscribers/$CLIENT_SUBJECT" -H "Authorization: Bearer ${TOKEN}"
curl "$TUNNEL_RELAY_API/subscribers/DOESNT_EXIST" -H "Authorization: Bearer ${TOKEN}"
echo ""
