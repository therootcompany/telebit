source .env
TOKEN=$(go run -mod=vendor cmd/signjwt/*.go $SECRET)
MGMT_BASEURL=${MGMT_BASEURL:-"http://mgmt.example.com:3010"}

CLIENT_SUBJECT=${CLIENT_SUBJECT:-"newbie"}
curl -X POST $MGMT_BASEURL/api/devices \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{ "slug": "'$CLIENT_SUBJECT'" }'
