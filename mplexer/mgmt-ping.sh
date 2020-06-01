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
my_parts=$(go run cmd/signjwt/*.go $my_shared machineid)
my_ppid=$(echo $my_parts | cut -d' ' -f1)
my_keyid=$(echo $my_parts | cut -d' ' -f2)
echo "PPID: $my_ppid KeyID: $my_keyid"

TOKEN=$(go run cmd/signjwt/*.go $my_ppid)
curl -X POST http://localhost:3000/api/ping  -H "Authorization: Bearer ${TOKEN}"
curl http://localhost:3000/api/inspect  -H "Authorization: Bearer ${TOKEN}"
