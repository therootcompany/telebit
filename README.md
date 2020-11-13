# Telebit

| **Telebit Client** | [Telebit Relay](/tree/master/cmd/telebit) | [Telebit Mgmt](/tree/master/cmd/mgmt) |

A secure, end-to-end Encrypted tunnel.

Because friends don't let friends localhost.

# Usage

```bash
telebit --env ./.env --verbose
```

Command-line flags or `.env` may be used.

```bash
# --acme-agree
export ACME_AGREE=true
# --acme-email
export ACME_EMAIL=johndoe@example.com
# --vendor-id
export VENDOR_ID=example.com
# --secret
export SECRET=YY-device-shared-secret-YY
# --tunnel-relay
export TUNNEL_RELAY_URL=https://tunnel.example.com/
# --locals
export LOCALS=https:*:3000
# --tls-locals
export TLS_LOCALS=https:*:3000
```

See `./telebit --help` for all options. \
See [`examples/client.env`][client-env] for detail explanations.

[client-env]: /tree/master/examples/client.env

### System Services

You can use `serviceman` to run `postgres`, `telebit`, and `telebit-mgmt` as system services

```bash
curl -fsS https://webinstall.dev/serviceman | bash
```

See the Cheat Sheet at https://webinstall.dev/serviceman

You can, of course, configure systemd (or whatever) by hand if you prefer.

## Example Local Web Application

The simplest way to test the tunnel is with a local web server.

```bash
mkdir -p tmp-app
pushd tmp-app/

cat << EOF > index.html
Hello, World!
EOF
```

### Ex: Caddy

```bash
curl -sS https://webinstall.dev/caddy | bash
```

```bash
caddy file-server --browse --listen :3000
```

### Ex: Python 3

```bash
python3 -m http.server 3000
```

# Build

```bash
git clone ssh://git@git.rootprojects.org:root/telebit.git
pushd telebit/
```

You can build with `go build`:

```bash
go build -mod vendor -race -o telebit cmd/telebit/telebit.go
```

Or with `goreleaser`:

```bash
goreleaser --rm-dist --skip-publish --snapshot
```

## Install Go

To install Go (on any of Windows, Mac, or Linux), see <https://webinstall.dev/golang>.

Installs Go to `~/.local/opt/go`.

**Mac, Linux**:

```bash
curl -fsS https://webinstall.dev/golang | bash
```

**Windows 10**:

```bash
curl.exe -fsSA "MS" https://webinstall.dev/golang | powershell
```

**Note**: The _minimum required go version_ is shown in `go.mod`. DO NOT use with `GOPATH`!

## Building Telebit

All dependencies are included, at the correct version in the `./vendor` directory.

```bash
go generate ./...

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -o telebit-linux ./cmd/telebit/*.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -mod vendor -o telebit-macos ./cmd/telebit/*.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -o telebit-windows-debug.exe ./cmd/telebit/*.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -ldflags "-H windowsgui" -o telebit-windows.exe ./cmd/telebit/*.go
```

The binary can be built with `VENDOR_ID` and `CLIENT_SECRET` built into the binary.
You can also change the `serviceName` and `serviceDescription` at build time.
See `examples/run-as-client.sh`.

# Help

```
Usage of telebit:
  ACME_AGREE
  --acme-agree
    	agree to the terms of the ACME service provider (required)
  --acme-directory string
    	ACME Directory URL
  ACME_EMAIL
  --acme-email string
    	email to use for Let's Encrypt / ACME registration
  --acme-http-01
    	enable HTTP-01 ACME challenges
  ACME_HTTP_01_RELAY_URL
  --acme-http-01-relay-url string
    	the base url of the ACME HTTP-01 relay, if not the same as the DNS-01 relay
  --acme-relay-url string
    	the base url of the ACME DNS-01 relay, if not the same as the tunnel relay
  --acme-staging
    	get fake certificates for testing
  --acme-storage string
    	path to ACME storage directory (default "./acme.d/")
  --acme-tls-alpn-01
    	enable TLS-ALPN-01 ACME challenges
  API_HOSTNAME
  --api-hostname string
    	the hostname used to manage clients
  --auth-url string
    	the base url for authentication, if not the same as the tunnel relay
  DEBUG
  --debug
    	show debug output (default true)
  --dns-01-delay duration
    	add an extra delay after dns self-check to allow DNS-01 challenges to propagate
  --dns-resolvers string
    	a list of resolvers in the format 8.8.8.8:53,8.8.4.4:53
  --env string
    	path to .env file
  --leeway duration
    	allow for time drift / skew (hard-coded to 15 minutes) (default 15m0s)
  LISTEN
  --listen string
    	list of bind addresses on which to listen, such as localhost:80, or :443
  LOCALS
  --locals string
    	a list of <from-domain>:<to-port>
  PORT_FORWARD
  --port-forward string
    	a list of <from-port>:<to-port> for raw port-forwarding
  SECRET
  --secret string
    	the same secret used by telebit-relay (used for JWT authentication)
  --spf-domain string
    	domain with SPF-like list of IP addresses which are allowed to connect to clients
  TLS_LOCALS
  --tls-locals string
    	like --locals, but TLS will be used to connect to the local port
  --token string
    	an auth token for the server (instead of generating --secret); use --token=false to ignore any $TOKEN in env
  TUNNEL_RELAY_URL
  --tunnel-relay-url string
    	the websocket url at which to connect to the tunnel relay
  VENDOR_ID
  --vendor-id string
    	a unique identifier for a deploy target environment
  VERBOSE
  VERBOSE_BYTES
  VERBOSE_RAW
  --verbose
    	log excessively
```
