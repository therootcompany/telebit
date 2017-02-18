# RVPN Server

Admin Interface
---------------
Developing the API calls and buliding the structure behind it.

- build get mux router package `go get "github.com/gorilla/mux"`
- run `go-rvpn-server`
- execute stunnel (as below)
- browse to <https://127.0.0.2:8000/api/servers>

```json
{"Servers":[{"ServerName":"0xc420019500","Domains":[{"Domain":"127.0.0.1","BytesIn":0,"BytesOut":2607},{"Domain":"hfc.daplie.me","BytesIn":0,"BytesOut":0},{"Domain":"test.hfc.daplie.me","BytesIn":0,"BytesOut":0}],"Duration":372.34454292,"BytesIn":65,"BytesOut":2607}]}
```

The above is telemetry from the RVPN server. (now supports domain byte tracking)

Work is continuing, please review make sure it is what you are looking for.

We will need to deal with server name, I am placing the point address for now, not sure how to handle the name.



Traffic Testing
---------------
* run go-rvpn-server
* execute stunnel.js

```bash
bin/stunnel.js --locals http://hfc.daplie.me:8443,http://test.hfc.daplie.me:3001,http://127.0.0.1:8080 --stunneld wss://localhost.daplie.me:3502 --secret abc123
```

* Browse to http://127.0.0.1:8080

```
INFO: connection: 2017/02/13 20:46:07.815340 connection.go:95: &{0xc420017350 0xc420181540 0xc42014acc0 127.0.0.1:49412 91 3390 0xc42014ad20 [hfc.daplie.me test.hfc.daplie.me 127.0.0.1]}
```
91, and 3390 are bytes in and bytes out.

```
INFO: external: 2017/02/13 20:46:07.814294 listener_webrequest.go:75: connState
&{{0xc42015e690}} 127.0.0.1:8080 127.0.0.1:49416
active
INFO: external: 2017/02/13 20:46:07.814327 listener_webrequest.go:24: handlerWebRequestExternal
INFO: external: 2017/02/13 20:46:07.814358 listener_webrequest.go:30: "GET /favicon.ico HTTP/1.1\r\nHost: 127.0.0.1:8080\r\nAccept: image/webp,image/*,*/*;q=0.8\r\nAccept-Encoding: gzip, deflate, sdch, br\r\nAccept-Language: en-US,en;q=0.8\r\nCache-Control: no-cache\r\nConnection: keep-alive\r\nPragma: no-cache\r\nReferer: http://127.0.0.1:8080/\r\nUser-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36\r\n\r\n"
INFO: external: 2017/02/13 20:46:07.814372 listener_webrequest.go:50: &{0xc420017350 0xc420181540 0xc42014acc0 127.0.0.1:49412 78 2963 0xc42014ad20 [hfc.daplie.me test.hfc.daplie.me 127.0.0.1]} 127.0.0.1 49416
header:  ipv4,127.0.0.2,49416,398,na
meta:  {[254 27] 0 [0 0 0 0] [254 27 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0] 0}
Data:  GET /favicon.ico HTTP/1.1
Host: 127.0.0.1:8080
Accept: image/webp,image/*,*/*;q=0.8
Accept-Encoding: gzip, deflate, sdch, br
Accept-Language: en-US,en;q=0.8
Cache-Control: no-cache
Connection: keep-alive
Pragma: no-cache
Referer: http://127.0.0.1:8080/
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36
```

```
[error] closing ',undefined,undefined' because 'unsupported service 'undefined'' (-1 clients)
[_onLocalClose]
<Buffer fe 0a 2c 2c 2c 31 2c 65 72 72 6f 72 00>
not v1 (or data is corrupt)
[error] closing ',undefined,undefined' because 'unsupported service 'undefined'' (-1 clients)
[_onLocalClose]
<Buffer fe 0a 2c 2c 2c 31 2c 65 72 72 6f 72 00>
not v1 (or data is corrupt)
```
Above is an attempt to connection...

```
INFO: external: 2017/02/13 20:50:45.883267 listener_webrequest.go:75: connState
&{{0xc42015e690}} 127.0.0.1:8080 127.0.0.1:49416
active
INFO: external: 2017/02/13 20:50:45.883298 listener_webrequest.go:24: handlerWebRequestExternal
INFO: external: 2017/02/13 20:50:45.883331 listener_webrequest.go:30: "GET /favicon.ico HTTP/1.1\r\nHost: 127.0.0.1:8080\r\nAccept: image/webp,image/*,*/*;q=0.8\r\nAccept-Encoding: gzip, deflate, sdch, br\r\nAccept-Language: en-US,en;q=0.8\r\nConnection: keep-alive\r\nReferer: http://127.0.0.1:8080/\r\nUser-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36\r\n\r\n"
INFO: external: 2017/02/13 20:50:45.883338 listener_webrequest.go:46: unable to match  127.0.0.1  to an existing connection
INFO: external: 2017/02/13 20:50:45.883362 listener_webrequest.go:75: connState
&{{0xc42015e690}} 127.0.0.1:8080 127.0.0.1:49416
```
The above is a connection to a domain that was not registered.


Branch: restructured
--------------------
* restructure into various packages, removing items from package "main"
* used caddy source as a guide
* weird debugging issue introduced, will not halt, break point unverified.
* although a test project debugs just fin

listener_client — the WSS client
--------------------------------
- removed support for anything admin
- injected the domains from the claim 
- domains are now included as initialDomains
- registration performans as normal but includes adding the domains to a map of domains, and a collection of domains on the connection.
- the system now supports look up fast in either direction, not sure if it will be needed.
- reads a chan during registration before allowing traffic, making sure all is well.
- registration returns a true on the channel if all is well. If it is not, false. Likely will add some text to pass back.

Connection
----------
- added support for boolean channel
- support for initial domains in a slice, these are brought back from the JWT as a interface and then are type asserted into the map
- removed all the old timer sender dwell stuff as a POC for traffic counts.

ConnectionTable
---------------
- added support for domain announcement after the WSS is connection. Not sure if we will need these. They have not been implemented.
- I assume all domains are registered with JWT unless I hear differently which would require a new WSS session
- expanded NewTable constructor
- populating domains into the domain map, and into the connection slice.
- added support for removing domains when a connection is removed.

Packer
------
- added support for a PackerHeader type and PackerData type
- these are connected in a Packer type
- support for calculated address family based on ip address property
- service field is set to "na"

Logging
-------
- unified package logging based on a package init.  Will likely need to remove this

Tests
-----
- stared to structure project for tests.  


Build Instructions
------------------

Create a subinterface:
```bash
sudo ifconfig lo0 alias 127.0.0.2 up
```
The above creates an alias the code is able to bind against for admin.  Admin is still in progress.

Get the dependencies

```bash
go get github.com/gorilla/websocket
go get github.com/dgrijalva/jwt-go

git clone git@git.daplie.com:Daplie/localhost.daplie.me-certificates.git 
ln -s localhost.daplie.me-certificates/certs/localhost.daplie.me certs
```

Run the VPN
```bash
go build && ./go-rvpn-server
```

In another terminal execute the client
``` bash
bin/stunnel.js --locals http:hfc.daplie.me:3000,http://test.hfc.daplie.me:3001 --stunneld wss://localhost.daplie.me:8000 --secret abc123
```

A good authentication
```
INFO: 2017/02/02 21:22:22 vpn-server.go:88: startup
INFO: 2017/02/02 21:22:22 vpn-server.go:90: :8000
INFO: 2017/02/02 21:22:22 vpn-server.go:73: starting Listener
INFO: 2017/02/02 21:22:22 connection_table.go:19: ConnectionTable starting
INFO: 2017/02/02 21:22:24 connection.go:113: websocket opening  127.0.0.1:55469
INFO: 2017/02/02 21:22:24 connection.go:127: access_token valid
INFO: 2017/02/02 21:22:24 connection.go:130: processing domains [hfc.daplie.me test.hfc.daplie.me]
```

Change the key on the tunnel client to test a valid secret
``` bash
INFO: 2017/02/02 21:24:13 vpn-server.go:88: startup
INFO: 2017/02/02 21:24:13 vpn-server.go:90: :8000
INFO: 2017/02/02 21:24:13 vpn-server.go:73: starting Listener
INFO: 2017/02/02 21:24:13 connection_table.go:19: ConnectionTable starting
INFO: 2017/02/02 21:24:15 connection.go:113: websocket opening  127.0.0.1:55487
INFO: 2017/02/02 21:24:15 connection.go:123: access_token invalid...closing connection
```

Connection to the External Interface.
http://127.0.0.1:8080

The request is dumped to stdio.  This is in preparation of taking that request and sending it back to the designated WSS connection
The system needs to track the response coming back, decouple it, and place it back on the wire in the form of a response stream.  Since

A Poor Man's Reverse VPN written in Go

Context
-------

Even in the worst of conditions the fanciest of firewalls can't stop a WebSocket
running over https from creating a secure tunnel.

Whether at home behind a router that lacks UPnP compliance, at school, work,
the library - or even on an airplane, we want any device (or even a browser or
app) to be able to serve from anywhere.

Motivation
----------

We originally wrote this in node.js as
[node-tunnel-server](https://git.daplie.com/Daplie/node-tunnel-server),
but there are a few problems:

* metering
* resource utilization
* binary transfer

### metering

We want to be able to meter all traffic on a socket.
In node.js it wasn't feasible to be able to track the original socket handle
all the way back from the web socket authentication through the various
wrappers.

A user connects via a websocket to the tunnel server
and an authentication token is presented.
If the connection is established the socket should then be metered and reported
including total bytes sent and received and size of payload bytes sent and
received (because the tunnelling adds some overhead).

### resource utilization

node.js does not support usage of multiple cores in-process.
The overhead of passing socket connections between processes seemed non-trivial
at best and likely much less efficient, and impossible at worst.

### binary transfer

node.js doesn't handle binary data very well. People will be transferring
gigabytes of data.

Short Term Goal
----

Build a server compatible with the node.js client (JWT authentication)
that can meter authenticated connections, utilize multiple cores efficiently,
and efficienty garbage collect gigabytes upon gigabytes of transfer.