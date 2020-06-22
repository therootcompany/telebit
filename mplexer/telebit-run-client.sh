#!/bin/bash

go run cmd/telebit/*.go --acme-agree=true \
    --acme-relay http://devices.rootprojects.org:3010/api/dns \
    --auth-url http://devices.rootprojects.org:3010/api \
    --relay wss://devices.rootprojects.org:8443/api/ws \
    --app-id test-id \
    --secret k7nsLSwNKbOeBhDFpbhwGHv \
    --listen 3443
