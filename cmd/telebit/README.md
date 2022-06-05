# Telebit Relay

| [Telebit Client](/README.md) | **Telebit Relay** | [Telebit Mgmt](../mgmt) |

Secure tunnel, relay, and reverse-proxy server.

# Usage

**Only** port 443 must be public.

```bash
./telebit-relay --acme-http-01
```

```bash
# allow access to privileged ports
sudo setcap 'cap_net_bind_service=+ep' ./telebit-relay
```

Copy `examples/relay.env` as `.env` in the working directory.

```bash
# --secret
export SECRET=XxX-mgmt-secret-XxX
# --api-hostname
export API_HOSTNAME=tunnel.example.com
# --listen
export LISTEN=":443"
# --locals
export LOCALS=https:mgmt.example.com:6468
# --auth-url
export AUTH_URL=http://localhost:6468/api
# --proxy-http-01
export PROXY_HTTP_01=http://mgmt.example.com:6468
# --acme-agree
export ACME_AGREE=true
# --acme-email
export ACME_EMAIL=telebit@example.com
# --acme-relay
export ACME_RELAY_URL=http://localhost:6468/api/acme-relay
```

See `./telebit-relay --help` for all options. \
See [`examples/relay.env`][relay-env] for detail explanations.

[relay-env]: /examples/relay.env

Note: It is not necessary to specify the `--flags` when using the ENVs.

## API

### Discovery

Each telebit relay with expose its discovery endpoint at

- `.well-known/telebit.app/index.json`

The response will look something like

```json
```

## System Services

You can use `serviceman` to run `postgres`, `telebit`, and `telebit-mgmt` as system services

```bash
curl -fsS https://webinstall.dev/serviceman | bash
```

See the Cheat Sheet at https://webinstall.dev/serviceman

You can, of course, configure systemd (or whatever) by hand if you prefer.

# API

List all connected devices

```bash
bash examples/admin-list-devices.sh
```

```bash
curl -L https://devices.example.com/api/subscribers -H "Authorization: Bearer ${TOKEN}"
```

```json
{
    "success": true,
    "subscribers": [{ "since": "2020-07-22T08:20:40Z", "sub": "ruby", "sockets": ["73.228.72.97:50737"], "clients": 0 }]
}
```

Show connectivity, of a single device, if any

```bash
curl -L https://devices.example.com/api/subscribers -H "Authorization: Bearer ${TOKEN}"
```

```json
{
    "success": true,
    "subscribers": [{ "since": "2020-07-22T08:20:40Z", "sub": "ruby", "sockets": ["73.228.72.97:50737"], "clients": 0 }]
}
```

Force a device to disconnect:

```bash
bash examples/admin-disconnect-device.sh
```

# Build

You can build with `go build`:

```bash
go generate -mod vendor ./...
go build -mod vendor -race -o telebit-relay cmd/telebit/*.go
```

Or with `goreleaser`:

```bash
goreleaser --rm-dist --skip-publish --snapshot
```

Or cross-compile:

```bash
go generate -mod vendor ./...

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -o telebit-relay-linux ./cmd/telebit/*.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -mod vendor -o telebit-relay-macos ./cmd/telebit/*.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -o telebit-relay-windows-debug.exe ./cmd/telebit/*.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -ldflags "-H windowsgui" -o telebit-relay-windows.exe ./cmd/telebit/*.go
```
