# Telebit

A secure, end-to-end Encrypted tunnel.

Because friends don't let friends localhost.

## Install Go

Installs Go to `~/.local/opt/go` for MacOS and Linux:

```bash
curl -fsS https://webinstall.dev/golang | bash
```

Windows 10:

```bash
curl.exe -fsSA "MS" https://webinstall.dev/golang | powershell
```

**Note**: The _minimum required go version_ is shown in `go.mod`. DO NOT use with `GOPATH`!

## Relay Server

All dependencies are included, at the correct version in the `./vendor` directory.

```bash
go generate ./...

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -o telebit-relay-linux ./cmd/telebit/*.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -mod vendor -o telebit-relay-macos ./cmd/telebit/*.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -o telebit-relay-windows-debug.exe ./cmd/telebit/*.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -ldflags "-H windowsgui" -o telebit-relay-windows.exe ./cmd/telebit/*.go
```

The binary can be built with `VENDOR_ID` and `CLIENT_SECRET` built into the binary.
See `examples/run-as-client.sh`.

### Configure

Command-line flags or `.env` may be used.

See `./telebit-relay --help` for all options, and `examples/relay.env` for their corresponding ENVs.

### Example

Copy `examples/relay.env` as `.env` in the working directory.

```bash
# For Tunnel Relay Server
API_HOSTNAME=devices.example.com
LISTEN=:80,:443
LOCALS=https:mgmt.devices.example.com:3010
VERBOSE=true

# For Device Management & Authentication
AUTH_URL=http://localhost:3010/api

# For Let's Encrypt / ACME registration
ACME_AGREE=true
ACME_EMAIL=letsencrypt@example.com

# For Let's Encrypt / ACME challenges
ACME_RELAY_URL=http://localhost:3010/api/dns
SECRET=xxxxxxxxxxxxxxxx
GODADDY_API_KEY=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
GODADDY_API_SECRET=xxxxxxxxxxxxxxxxxxxxxx
```

Note: It is not necessary to specify the `--flags` when using the ENVs.

```bash
./telebit-relay \
    --api-hostname $API_HOSTNAME \
    --auth-url "$AUTH_URL" \
    --acme-agree "$ACME_AGREE" \
    --acme-email "$ACME_EMAIL" \
    --acme-relay-url "$ACME_RELAY_URL" \
    --secret "$SECRET" \
    --listen "$LISTEN"
```

## Management Server

```bash
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

To build `signjwt`:

```bash
go build -mod=vendor -ldflags "-s -w" -o signjwt cmd/signjwt/*.go
```

To generate an `admin` token:

```bash
VERDOR_ID="test-id"
SECRET="xxxxxxxxxxx"
TOKEN=$(./signjwt \
    --expires-in 15m \
    --vendor-id $VENDOR_ID \
    --secret $SECRET \
    --machine-ppid $SECRET
)
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

Copy `examples/client.env` as `.env` in the working directory.

```bash
# For Client
VENDOR_ID=test-id
CLIENT_SUBJECT=newieb
CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxx
AUTH_URL="https://mgmt.devices.example.com/api"
TUNNEL_RELAY_URL=wss://devices.example.com/ws
LOCALS=https:newbie.devices.example.com:3000,http:newbie.devices.example.com:3000
#PORT_FORWARDS=3443:3001,8443:3002

# For Debugging
VERBOSE=true
#VERBOSE_BYTES=true
#VERBOSE_RAW=true

# For Let's Encrypt / ACME registration
ACME_AGREE=true
ACME_EMAIL=letsencrypt@example.com

# For Let's Encrypt / ACME challenges
ACME_RELAY_URL="https://mgmt.devices.example.com/api/dns"
```

```bash
./telebit-client \
    --auth-url $AUTH_URL \
    --vendor-id "$VENDOR_ID" \
    --secret "$CLIENT_SECRET" \
    --tunnel-relay-url $TUNNEL_RELAY_URL \
    --listen "$LISTEN" \
    --locals "$LOCALS" \
    --acme-agree="$ACME_AGREE" \
    --acme-email "$ACME_EMAIL" \
    --acme-relay-url $ACME_RELAY_URL \
    --verbose=$VERBOSE
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

## Glossary

```
--vendor-id         $VENDOR_ID          an arbitrary id used as part of authentication
--secret            $SECRET             the secret for creating JWTs
--auth-url          $AUTH_URL           the full url prefix of the server that will validate tokens
--tunnel-relay-url  $TUNNEL_RELAY_URL   the full url of the websocket tunnel server
--locals            $LOCALS             a list of `scheme:domainname:port`
                                        for forwarding incoming `domainname` to local `port`
--port-forwards     $PORT_FORWARDS      a list of `remote:local` tcp port-forwarding
--verbose           $VERBOSE            logs everything, including abbreviated data (as hex)
                    $VERBOSE_BYTES      logs full data (as hex)
                    $VERBOSE_RAW        logs full data (as string)
--acme-agree        $ACME_AGREE         agree to the ACME service agreement
--acme-email        $ACME_EMAIL         the webmaster email for ACME notices
--acme-relay-url    $ACME_RELAY_URL     the server that will relay ACME DNS-01 requests
```
