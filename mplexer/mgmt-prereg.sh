TOKEN=$(go run cmd/signjwt/*.go)
echo "TOKEN: $TOKEN"

my_shared="k7nsLSwNKbOeBhDFpbhwGHv"
my_domain="duckdns.org"
my_client="rooted"
curl -X POST http://roottest.duckdns.org:3010/api/devices \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{ "slug": "'$my_client'", "shared_key": "'$my_shared'" }'
