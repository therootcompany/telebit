#!/bin/bash

set -e
set -u

source .env
AUTH_URL="${AUTH_URL:-"http://localhost:3000/api"}"

# 1. (srv) create a new shared key for a given slug
# 2. (dev) try to update via ping
# 3. (dev) use key to exchange machine id
# 4. (dev) use key to connect to remote
# 5. (dev) ping occasionally

echo "CLIENT_SECRET: $CLIENT_SECRET"
TOKEN=$(go run cmd/signjwt/*.go --vendor-id "$VENDOR_ID" --secret "$CLIENT_SECRET")
echo "TOKEN 1: '$TOKEN'"

my_parts=$(go run cmd/signjwt/*.go --vendor-id "$VENDOR_ID" --secret $CLIENT_SECRET --machine-ppid-only)
my_ppid=$(echo $my_parts | cut -d' ' -f1)
my_keyid=$(echo $my_parts | cut -d' ' -f2)
echo "PPID: $my_ppid KeyID: $my_keyid"

echo "$AUTH_URL"
curl -X POST "$AUTH_URL/ping" -H "Authorization: Bearer ${TOKEN}"
echo ""
curl "$AUTH_URL/inspect" -H "Authorization: Bearer ${TOKEN}"
echo ""
