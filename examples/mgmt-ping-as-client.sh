#!/bin/bash

set -e
set -u

source .env
MGMT_URL="${MGMT_URL:-"http://localhost:3000/api"}"

TOKEN=$(go run cmd/signjwt/*.go \
    --expires-in 1m \
    --vendor-id "$VENDOR_ID" \
    --secret "$CLIENT_SECRET"
)

echo "$MGMT_URL"
curl -X POST "$MGMT_URL/ping" -H "Authorization: Bearer ${TOKEN}"
echo ""
curl "$MGMT_URL/inspect" -H "Authorization: Bearer ${TOKEN}"
echo ""
