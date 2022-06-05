#!/bin/bash

set -e
set -u

source .env
MGMT_PORT="${MGMT_PORT:-3000}"
MGMT_URL="${MGMT_URL:-"http://localhost:${MGMT_PORT}/api"}"

TOKEN=$(
    go run cmd/signjwt/*.go \
        --expires-in 1m \
        --vendor-id "$VENDOR_ID" \
        --secret "$RELAY_SECRET" \
        --machine-ppid "$RELAY_SECRET"
)

echo "MGMT URL: $MGMT_URL"
curl -X POST "$MGMT_URL/ping" -H "Authorization: Bearer ${TOKEN}"
echo ""
curl "$MGMT_URL/inspect" -H "Authorization: Bearer ${TOKEN}"
echo ""
