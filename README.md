# Telebit

A secure, end-to-end Encrypted tunnel.

Because friends don't let friends localhost.

## Install Go

Installs Go to `~/.local/opt/go` for MacOS and Linux:

```bash
curl https://webinstall.dev/golang | bash
```

For Windows, see https://golang.org/dl

**Note**: The _minimum required go version_ is shown in `go.mod`. DO NOT use with `GOPATH`!

## Relay Server

All dependencies are included, at the correct version in the `./vendor` directory.

```bash
go generate ./...

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -o telebit-relay-linux ./cmd/telebit-relay/*.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -mod vendor -o telebit-relay-macos ./cmd/telebit-relay/*.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -o telebit-relay-windows-debug.exe ./cmd/telebit-relay/*.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -ldflags "-H windowsgui" -o telebit-relay-windows.exe ./cmd/telebit-relay/*.go
```

### Configure

Command-line flags or `.env` may be used.

See `./telebit-relay --help` for all options, and `examples/relay.env` for their corresponding ENVs.

### Example

```bash
./telebit-relay --acme-agree=true --auth-url=http://localhost:3010/api
```

Copy `examples/relay.env` as `.env` in the working directory.

## Management Server

```bash
pushd mplexy/

go generate ./...

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -o mgmt-server-linux ./cmd/mgmt/*.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -mod vendor -o mgmt-server-macos ./cmd/mgmt/*.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -o mgmt-server-windows-debug.exe ./cmd/mgmt/*.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -ldflags "-H windowsgui" -o mgmt-server-windows.exe ./cmd/mgmt/*.go
```

### Example

```bash
./telebit-mgmt --domain devices.example.com --port 3010
```

Copy `examples/mgmt.env` as `.env` in the working directory.

### Device Management API

Create a token with the same `SECRET` used with the `mgmt` server,
and add a device by its `subdomain`.

```bash
SECRET="xxxxxxxxxxx"
TOKEN=$(go run -mod=vendor cmd/signjwt/*.go $SECRET)
```

Authorize a device:

```bash
my_subdomain="xxxx"
my_mgmt_host=http://mgmt.example.com:3010
curl -X POST $my_mgmt_host/api/devices \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{ "slug": "'$my_subdomain'" }'
```

```json
{ "shared_key": "ZZZZZZZZ" }
```

Show data of a single device

```bash
my_subdomain="xxxx"
curl -L http://mgmt.example.com:3010/api/devices/${my_subdomain} -H "Authorization: Bearer ${TOKEN}"
```

```json
{ "subdomain": "sub1", "updated_at": "2020-05-20T12:00:01Z" }
```

Get a list of connected devices:

```bash
curl -L http://mgmt.example.com:3010/api/devices -H "Authorization: Bearer ${TOKEN}"
```

```json
[{ "subdomain": "sub1", "updated_at": "2020-05-20T12:00:01Z" }]
```

Get a list of disconnected devices:

```bash
curl -L http://mgmt.example.com:3010/api/devices?inactive=true -H "Authorization: Bearer ${TOKEN}"
```

Deauthorize a device:

```bash
my_subdomain="xxxx"
curl -L -X DELETE http://mgmt.example.com:3010/api/devices/${my_subdomain} -H "Authorization: Bearer ${TOKEN}"
```

## Relay Client

All dependencies are included, at the correct version in the `./vendor` directory.

```bash
pushd mplexy/

go generate ./...

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -o telebit-client-linux ./cmd/telebit/*.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -mod vendor -o telebit-client-macos ./cmd/telebit/*.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -o telebit-client-windows-debug.exe ./cmd/telebit/*.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -ldflags "-H windowsgui" -o telebit-client-windows.exe ./cmd/telebit/*.go
```

### Configure

Command-line flags or `.env` may be used.

See `./telebit-client --help` for all options, and `examples/client.env` for their corresponding ENVs.

### Example

```bash
./telebit-client --acme-agree=true \
    --relay wss://devices.example.com \
    --app-id test-id --secret ZR2rxYmcKJcmtKgmH9D5Qw \
    --acme-relay http://mgmt.example.com:3010/api/dns \
    --auth-url http://mgmt.example.com:3010/api \
    --locals http://xxx.devices.example.com:8080,https://xxx.devices.example.com:8080
```

## Local Web Application

Currently only raw TCP is tunneled.

This means that either the application must handle and terminate encrypted TLS connections, or use HTTP (instead of HTTPS).
This will be available in the next release.

```bash
mkdir -p tmp-app
pushd tmp-app/

cat << EOF > index.html
Hello, World!
EOF

python3 -m http.server 3000
```
