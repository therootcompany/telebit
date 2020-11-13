# Custom Telebit Server Setup Guide

There are three components to a telebit configuration:

1. the telebit relay
2. the management (authentication) server
3. (optional) DNS-01 and/or HTTP-01 ACME relay

# DNS

-   `devices.example.com` and `*.devices.example.com` should have A (and AAAA) records pointing to the tunnel server
    -   `https://devices.example.com/` is `TUNNEL_RELAY_URL`
    -   `devices.example.com` is the _primary_ or _base_ domain for the devices `telebit-mgmt --domain devices.example.com`

All of the devices need to be under the same domain. You are limited by Let's Encrypt to 10-20 certificates per week. We can solve for this in the future if needed - either by adding more domains or by adding devices.example.com to the PSL (the stated reason would be for browser security, NOT for Let's Encrypt limits).

-   Other domains can be pointed to the same server. For example:
    -   It would be OKAY to use `tunnel.example.com` as `TUNNEL_RELAY_URL`.
    -   It would be OKAY to use `auth.example.com` as `AUTH_URL`
-   It is fine to have the `AUTH_URL` on a different server.
-   having multiple tunnel server URLs is NOT supported, but this is a relatively small change to the `telebit-mgmt` in the future

## White Label Builds

```bash
go generate ./...

VENDOR_ID="example.com"

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
    -mod=vendor \
    -ldflags="-X 'main.VendorID=$VENDOR_ID'" \
    -o telebit-debug.exe \
    ./cmd/telebit/telebit.go

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
    -mod=vendor \
    -ldflags "-H windowsgui -X 'main.VendorID=$VENDOR_ID'" \
    -o telebit-windows.exe \
    ./cmd/telebit/telebit.go
```
