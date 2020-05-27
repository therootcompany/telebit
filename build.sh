#!/bin/bash

set -e
set -u

#CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -o telebit-relay-linux ./cmd/telebit-relay/telebit-relay.go
#CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -mod vendor -o telebit-relay-macos ./cmd/telebit-relay/telebit-relay.go
#CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -o telebit-relay-windows.exe ./cmd/telebit-relay/telebit-relay.go

go generate ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -o telebit-client-linux ./cmd/telebit/*.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -mod vendor -o telebit-client-macos ./cmd/telebit/*.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -o telebit-client-windows-debug.exe ./cmd/telebit/*.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -ldflags "-H windowsgui" -o telebit-client-windows.exe ./cmd/telebit/*.go
