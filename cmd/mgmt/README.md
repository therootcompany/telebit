# Telebit Mgmt

| [Telebit Client](/README.md) | [Telebit Relay](../telebit) | **Telebit Mgmt** |

Device Management, Authorization, and ACME Relay Server.

# Usage

This does not need to be on a public port for client devices,
but it must be directly accessible by the telebit relay.

It must also run on port 80 if HTTP-01 challenges are being relayed.

This should be https-enabled unless on localhost behind the telebit relay.

```bash
./telebit-mgmt
```

```bash
# allow access to privileged ports
sudo setcap 'cap_net_bind_service=+ep' ./telebit-mgmt
```

Command-line flags or `.env` may be used.

```bash
# --secret
export SECRET=XxX-mgmt-secret-XxX
# --domain
export DOMAIN=devices.example.com
# --tunnel-domain
export TUNNEL_DOMAIN=tunnel.example.com
# --db-url
export DB_URL=postgres://postgres:postgres@localhost:5432/postgres
# --port
export PORT=6468
```

See `./telebit --help` for all options. \
See [`examples/mgmt.env`][mgmt-env] for detail explanations.

[mgmt-env]: /examples/mgmt.env

## System Services

You can use `serviceman` to run `postgres`, `telebit`, and `telebit-mgmt` as system services

```bash
curl -fsS https://webinstall.dev/serviceman | bash
```

See the Cheat Sheet at https://webinstall.dev/serviceman

You can, of course, configure systemd (or whatever) by hand if you prefer.

## Install Postgres

Install postgres and start it as a service on MacOS and Linux:

```bash
curl -sS https://webinstall.dev/postgres | bash
```

```bash
sudo env PATH="$PATH" \
    serviceman add --system --username $(whoami) --name postgres -- \
    postgres -D "$HOME/.local/share/postgres/var" -p 5432
```

See the Cheat Sheet at https://webinstall.dev/postgres

## Create Admin Token

The admin token can be used to interact with the server.

```bash
VENDOR_ID="example.com"
MGMT_SECRET=XxX-mgmt-secret-XxX
ADMIN_TOKEN=$(go run cmd/signjwt/*.go \
    --debug \
    --expires-in 15m \
    --vendor-id $VENDOR_ID \
    --secret $MGMT_SECRET \
    --machine-ppid $MGMT_SECRET
)
```

## Register New Device

This will return a new shared secret that can be used to register a new client device.

```bash
my_subdomain="foobar"
my_mgmt_host=https://mgmt.example.com

curl -X POST $my_mgmt_host/api/devices \
    -H "Authorization: Bearer ${ADMIN_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{ "slug": "'$my_subdomain'" }'
```

# API

```bash
my_subdomain="ruby"
curl -X DELETE http://mgmt.example.com:6468/api/subscribers/ruby" -H "Authorization: Bearer ${TOKEN}"
```

```json
{ "success": true }
```

Create a token with the same `SECRET` used with the `mgmt` server,
and add a device by its `subdomain`.

To build `signjwt`:

```bash
go build -mod=vendor -ldflags "-s -w" -o signjwt cmd/signjwt/*.go
```

To generate an `admin` token:

```bash
VENDOR_ID="test-id"
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
my_mgmt_host=http://mgmt.example.com:6468
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
curl -L http://mgmt.example.com:6468/api/devices/${my_subdomain} -H "Authorization: Bearer ${TOKEN}"
```

```json
{ "subdomain": "sub1", "updated_at": "2020-05-20T12:00:01Z" }
```

Get a list of connected devices:

```bash
curl -L http://mgmt.example.com:6468/api/devices -H "Authorization: Bearer ${TOKEN}"
```

```json
[{ "subdomain": "sub1", "updated_at": "2020-05-20T12:00:01Z" }]
```

Get a list of disconnected devices:

```bash
curl -L http://mgmt.example.com:6468/api/devices?inactive=true -H "Authorization: Bearer ${TOKEN}"
```

Deauthorize a device:

```bash
my_subdomain="xxxx"
curl -L -X DELETE http://mgmt.example.com:6468/api/devices/${my_subdomain} -H "Authorization: Bearer ${TOKEN}"
```

# Build

You can build with `go build`:

```bash
go generate -mod vendor ./...
go build -mod vendor -race -o telebit-mgmt cmd/mgmt/*.go
```

Or with `goreleaser`:

```bash
goreleaser --rm-dist --skip-publish --snapshot
```

Or cross-compile:

```bash
go generate -mod vendor ./...

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -o telebit-mgmt-linux ./cmd/mgmt/*.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -mod vendor -o telebit-mgmt-macos ./cmd/mgmt/*.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -o telebit-mgmt-windows-debug.exe ./cmd/mgmt/*.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -ldflags "-H windowsgui" -o telebit-mgmt-windows.exe ./cmd/mgmt/*.go
```
