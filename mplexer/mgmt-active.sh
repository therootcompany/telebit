TOKEN=$(go run cmd/signjwt/*.go)
echo "TOKEN: $TOKEN"

curl -L http://localhost:3000/api/devices -H "Authorization: Bearer ${TOKEN}"
