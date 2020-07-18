source .env

TOKEN=$(go run -mod=vendor cmd/signjwt/*.go $SECRET)
AUTH_URL=${AUTH_URL:-"http://mgmt.example.com:3010/api"}

CLIENT_SUBJECT=${CLIENT_SUBJECT:-"newbie"}
curl -X POST $AUTH_URL/devices \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{ "slug": "'$CLIENT_SUBJECT'" }'
