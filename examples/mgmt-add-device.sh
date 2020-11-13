source .env

# 1. (srv) create a new shared key for a given slug
# 2. (dev) try to update via ping
# 3. (dev) use key to exchange machine id
# 4. (dev) use key to connect to remote
# 5. (dev) ping occasionally

TOKEN=$(go run cmd/signjwt/*.go \
    --expires-in 1m \
    --vendor-id "$VENDOR_ID" \
    --secret "$RELAY_SECRET" \
    --machine-ppid "$RELAY_SECRET"
)

MGMT_URL=${MGMT_URL:-"http://mgmt.example.com:6468/api"}

CLIENT_SUBJECT=${CLIENT_SUBJECT:-"newbie"}
curl -X POST "$MGMT_URL/devices" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{ "slug": "'$CLIENT_SUBJECT'" }'
