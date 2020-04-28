# RVPN Server

## branch: load-balancing

-   code now passes traffic using just Root tools
-   this will require serve-https and node-tunnel-client to work
-   the system supports round-robin load balancing

### Build RVPN

```bash
go build -mod vendor ./cmd/telebit/
go build -mod vendor ./cmd/telebitd/
```

### Setup Some Entries

`/etc/hosts`:

```txt
127.0.0.1 tunnel.example.com rvpn.rootprojects.invalid hfc2.rootprojects.org hfc.rootprojects.org
```

### Start Up Webserver

```bash
tmp $ cd /tmp
tmp $ vim index.html --- Place some index content
tmp $ serve-https -p 8080 -d /tmp --servername hfc.rootprojects.org --agree-tos --email henry.f.camacho@gmail.com
```

### Start Tunnel Client

```bash
node-tunnel-client $ bin/stunnel.js --locals http://hfc.rootprojects.org:8080,http://test1.hfc.rootprojects.org:8080 --stunneld wss://localhost.rootprojects.org:8443 --secret abc123
```

### Execute RVPN

```bash
./telebitd
```

```txt
INFO: packer: 2017/03/02 19:16:52.652109 run.go:47: startup
-=-=-=-=-=-=-=-=-=-=
INFO: genericlistener: 2017/03/02 19:16:52.652777 manager.go:77: ConnectionTable starting
INFO: genericlistener: 2017/03/02 19:16:52.652806 connection_table.go:67: ConnectionTable starting
INFO: genericlistener: 2017/03/02 19:16:52.652826 manager.go:84: &{map[] 0xc420072420 0xc420072480}
INFO: genericlistener: 2017/03/02 19:16:52.652832 connection_table.go:50: Reaper waiting for  300  seconds
INFO: genericlistener: 2017/03/02 19:16:52.652856 manager.go:100: register fired 8443
INFO: genericlistener: 2017/03/02 19:16:52.652862 manager.go:110: listener starting up  8443
INFO: genericlistener: 2017/03/02 19:16:52.652868 manager.go:111: &{map[] 0xc420072420 0xc420072480}
INFO: genericlistener: 2017/03/02 19:16:52.652869 conn_tracking.go:25: Tracking Running
```

### Browse via tunnel

https://hfc.rootprojects.org:8443

### Test Load Balancing

In a new terminal

```bash
node-tunnel-client $ bin/stunnel.js --locals http://hfc.rootprojects.org:8080,http://test1.hfc.rootprojects.org:8080 --stunneld wss://localhost.rootprojects.org:8443 --secret abc123
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
