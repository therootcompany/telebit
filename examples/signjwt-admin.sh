#!/bin/bash

source .env

TOKEN=$(go run cmd/signjwt/*.go \
    --expires-in 1m \
    --vendor-id "$VENDOR_ID" \
    --secret "$MGMT_SECRET" \
    --machine-ppid "$MGMT_SECRET"
)
echo "MGMT_TOKEN: $TOKEN"

my_parts=$(
go run cmd/signjwt/*.go \
    --vendor-id "$VENDOR_ID" \
    --secret "$MGMT_SECRET" \
    --machine-ppid "$MGMT_SECRET" \
    --machine-ppid-only
)
my_ppid=$(echo $my_parts | cut -d' ' -f1)
my_keyid=$(echo $my_parts | cut -d' ' -f2)
echo "PPID (Priv): $my_ppid KeyID (Pub): $my_keyid"
