source .env

TOKEN=$(go run cmd/signjwt/*.go \
    --expires-in 1m \
    --vendor-id "$VENDOR_ID" \
    --secret "$RELAY_SECRET" \
    --machine-ppid "$RELAY_SECRET"
)

MGMT_URL=${MGMT_URL:-"http://mgmt.example.com:6468/api"}

CLIENT_SUBJECT=${CLIENT_SUBJECT:-"newbie"}
curl -X DELETE "$MGMT_URL/devices/$CLIENT_SUBJECT" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{ "slug": "'$CLIENT_SUBJECT'" }'
