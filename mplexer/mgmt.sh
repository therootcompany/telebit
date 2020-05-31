#!/bin/bash

set -e
set -u

# 1. (srv) create a new shared key for a given slug
# 2. (dev) try to update via ping
# 3. (dev) use key to exchange machine id
# 4. (dev) use key to connect to remote
# 5. (dev) ping occasionally

TOKEN=$(go run cmd/signjwt/*.go)
echo "TOKEN: $TOKEN"

my_shared="ZR2rxYmcKJcmtKgmH9D5Qw"
my_domain="example.com"
my_client="1-client-slug"
TOK=$(curl -X POST http://localhost:3000/api/devices -H "Authorization: Bearer ${TOKEN}" -H "Content-Type: application/json" -d '{ "slug": "'$my_client'", "shared_key": "'$my_shared'" }')
echo Response: $TOK

SHARED=$(echo "$TOK" | sed 's/.*shared_key":"//g' | sed 's/".*//')
echo Shared Key: $SHARED
my_parts=$(go run cmd/signjwt/*.go $SHARED machineid)
my_ppid=$(echo $my_parts | cut -d' ' -f1)
my_keyid=$(echo $my_parts | cut -d' ' -f2)
echo "PPID: $my_ppid KeyID: $my_keyid"

TOKEN=$(go run cmd/signjwt/*.go $my_ppid)
echo "PING 1 (should fail)"
curl -X POST http://localhost:3000/api/ping  -H "Authorization: Bearer ${TOKEN}"
echo ""

curl -X POST http://localhost:3000/api/register-device/$SHARED -H "Content-Type: application/json" -d '{ "machine_ppid": "'$my_ppid'", "public_key": "'$my_keyid'" }'
echo ''

echo "PING 2 (should work)"
curl -X POST http://localhost:3000/api/ping  -H "Authorization: Bearer ${TOKEN}"
echo ""

curl -X POST http://localhost:3000/api/dns/"${my_client}.${my_domain}" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{ "token": "xxxx", "key_authorization": "yyyy" }'
