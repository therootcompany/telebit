# Telebit Relay

| [Telebit Client](../../) | **Telebit Relay** | [Telebit Mgmt](../mgmt) |

### Example

Copy `examples/relay.env` as `.env` in the working directory.

```bash
# For Tunnel Relay Server
API_HOSTNAME=devices.example.com
LISTEN=:443
LOCALS=https:mgmt.devices.example.com:3010
VERBOSE=false

# For Device Management & Authentication
AUTH_URL=http://localhost:3010/api

# For Let's Encrypt / ACME registration
ACME_AGREE=true
ACME_EMAIL=letsencrypt@example.com

# For Let's Encrypt / ACME challenges
ACME_HTTP_01_RELAY_URL=http://localhost:3010/api/http
ACME_RELAY_URL=http://localhost:3010/api/dns
SECRET=xxxxxxxxxxxxxxxx
GODADDY_API_KEY=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
GODADDY_API_SECRET=xxxxxxxxxxxxxxxxxxxxxx
```

Note: It is not necessary to specify the `--flags` when using the ENVs.

```bash
./telebit \
    --api-hostname $API_HOSTNAME \
    --auth-url "$AUTH_URL" \
    --acme-agree "$ACME_AGREE" \
    --acme-email "$ACME_EMAIL" \
    --secret "$SECRET" \
    --listen "$LISTEN"
```

### API

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
