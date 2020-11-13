# Telebit Mgmt

# Config

```bash
VERBOSE=

PORT=6468

# JWT Verification Secret
#SECRET=XxxxxxxxxxxxxxxX

DB_URL=postgres://postgres:postgres@localhost:5432/postgres
DOMAIN=mgmt.example.com
TUNNEL_DOMAIN=tunnel.example.com

NAMECOM_USERNAME=johndoe
NAMECOM_API_TOKEN=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

## API

```bash
my_subdomain="ruby"
curl -X DELETE http://mgmt.example.com:3010/api/subscribers/ruby" -H "Authorization: Bearer ${TOKEN}"
```

```json
{ "success": true }
```

# Build

```bash
go generate -mod vendor ./...

pushd cmd/mgmt
    go build -mod vendor -o telebit-mgmt
popd
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
