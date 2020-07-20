TOKEN=$(go run cmd/signjwt/*.go)
echo "TOKEN: $TOKEN"

echo "Active:"
curl -L http://localhost:3000/api/devices -H "Authorization: Bearer ${TOKEN}"

echo "Inactive:"
curl -L http://localhost:3000/api/devices?inactive=true -H "Authorization: Bearer ${TOKEN}"
