# MGMT Server

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

# Build

```bash
go generate -mod vendor ./...

pushd cmd/mgmt
    go build -mod vendor -o telebit-mgmt
popd
```
