# RVPN Server

Build Instructions
------------------
Get the dependencies

```bash
go get github.com/gorilla/websocket
```

Run the VPN
```bash
go build && ./go-rvpn-server
```

Activate a webbrowser:  https://127.0.0.1:8000/

Open Dev Console

Hit the Start WebSocket --> should turn "Green"

Put some test in the send, and hit the send button.

* observe java console, every 'this is a test' coming from the vpn to client...
* observe terminal console when pressing "send".

```
INFO: 2017/02/01 21:22:49 connection_table.go:23: register fired
INFO: 2017/02/01 21:22:49 connection_table.go:27: &{0xc420120040 0xc420163cc0 0xc4201254a0 [::1]:61392 false 0 0}
INFO: 2017/02/01 21:22:49 connection.go:71: activate timer &{0xc42027ec00 {2 1486005774583377390 5000000000 0xcf900 0xc42027ec00 0}}
INFO: 2017/02/01 21:22:49 connection.go:96: activate timer &{0xc420125500 {0 1486005774583361223 5000000000 0xcf900 0xc420125500 0}}
INFO: 2017/02/01 21:22:53 connection.go:62: [97 115 100 102 97 115 100 102 97 115 100 102 97 115 100 102]
INFO: 2017/02/01 21:22:53 connection.go:65: &{0xc420120040 0xc420163cc0 0xc4201254a0 [::1]:61392 false 16 0}
INFO: 2017/02/01 21:22:54 connection.go:103: Dwell Activated
INFO: 2017/02/01 21:22:56 connection.go:62: [97 115 100 102 97 115 100 102 97 115 100 102 97 115 100 102]
INFO: 2017/02/01 21:22:56 connection.go:65: &{0xc420120040 0xc420163cc0 0xc4201254a0 [::1]:61392 false 32 14}
INFO: 2017/02/01 21:22:58 connection.go:62: [97 115 100 102 97 115 100 102 97 115 100 102 97 115 100 102]
INFO: 2017/02/01 21:22:58 connection.go:65: &{0xc420120040 0xc420163cc0 0xc4201254a0 [::1]:61392 false 48 14}
INFO: 2017/02/01 21:22:59 connection.go:103: Dwell Activated
```
The last two numbers after false are bytes read, bytes written.




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