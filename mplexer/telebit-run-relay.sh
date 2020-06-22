#!/bin/bash

go run cmd/telebit/*.go --acme-agree=true \
    --acme-relay http://devices.rootprojects.org:3010/api/dns \
    --auth-url http://devices.rootprojects.org:3010/api \
    --relay wss://devices.rootprojects.org:8443/ws \
    --app-id test-id \
    --secret xxxxyyyyssss8347 \
    --listen 3443
