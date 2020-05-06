# Telebit

A secure, end-to-end Encrypted tunnel.

Because friends don't let friends localhost.

## Install Go

Installs Go to `~/.local/opt/go` for MacOS and Linux:

```bash
curl https://webinstall.dev/golang | bash
```

For Windows, see https://golang.org/dl

## Relay Server

All dependencies are included, at the correct version in the `./vendor` directory.

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -o telebit-relay-linux ./cmd/telebit-relay/telebit-relay.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -mod vendor -o telebit-relay-macos ./cmd/telebit-relay/telebit-relay.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -o telebit-relay-windows.exe ./cmd/telebit-relay/telebit-relay.go
```

### Configure

Command-line flags or `.env` may be used.

See `./telebit-relay --help` for all options, and `examples/relay.env` for their corresponding ENVs.

### Example

```bash
./telebit-relay --acme-agree=true
```

Copy `examples/relay.env` as `.env` in the working directory.

## Relay Client

All dependencies are included, at the correct version in the `./vendor` directory.

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -o telebit-client-linux ./cmd/telebit/telebit.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -mod vendor -o telebit-client-macos ./cmd/telebit/telebit.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod vendor -o telebit-client-windows.exe ./cmd/telebit/telebit.go
```

### Configure

Command-line flags or `.env` may be used.

See `./telebit-client --help` for all options, and `examples/client.env` for their corresponding ENVs.

### Example

```bash
# For .env
SECRET=abcdef1234567890
```

```bash
node-tunnel-client $ bin/stunnel.js --locals http://hfc.rootprojects.org:8080,http://test1.hfc.rootprojects.org:8080 --relay wss://localhost.rootprojects.org:8443 --secret abcdef1234567890
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

### Check Results

-   you should see traffic going to both node-clients hitting the single webserver on the back end.
-   Browse: https://rvpn.rootprojects.invalid:8443/api/org.rootprojects.rvpn/servers

```javascript
{
	"type": "servers/GET",
	"schema": "",
	"txts": 1490473843,
	"txid": 8,
	"error": "ok",
	"error_description": "",
	"error_uri": "",
	"result": {
		"servers": [{
			"server_name": "0xc42014a0c0",
			"server_id": 1,
			"domains": [{
				"domain_name": "hfc.rootprojects.org",
				"server_id": 1,
				"bytes_in": 4055,
				"bytes_out": 8119,
				"requests": 12,
				"responses": 12,
				"source_addr": "127.0.0.1:55875"
			}, {
				"domain_name": "test1.hfc.rootprojects.org",
				"server_id": 1,
				"bytes_in": 0,
				"bytes_out": 0,
				"requests": 0,
				"responses": 0,
				"source_addr": "127.0.0.1:55875"
			}],
			"duration": 182.561747754,
			"idle": 21.445976033,
			"bytes_in": 8119,
			"bytes_out": 4055,
			"requests": 12,
			"responses": 12,
			"source_address": "127.0.0.1:55875"
		}, {
			"server_name": "0xc4200ea3c0",
			"server_id": 2,
			"domains": [{
				"domain_name": "hfc.rootprojects.org",
				"server_id": 2,
				"bytes_in": 1098,
				"bytes_out": 62,
				"requests": 2,
				"responses": 2,
				"source_addr": "127.0.0.1:56318"
			}, {
				"domain_name": "test1.hfc.rootprojects.org",
				"server_id": 2,
				"bytes_in": 0,
				"bytes_out": 0,
				"requests": 0,
				"responses": 0,
				"source_addr": "127.0.0.1:56318"
			}],
			"duration": 65.481814913,
			"idle": 23.589609269,
			"bytes_in": 62,
			"bytes_out": 1098,
			"requests": 2,
			"responses": 2,
			"source_address": "127.0.0.1:56318"
		}]
	}
}
```
