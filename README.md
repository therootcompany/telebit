# RVPN Server

## restructured-http

- connection handling has been totally rewritten.
- on a specific port RVPN can determine the following:  
    - if a connection is encrypted or not encrypted
    - if a request is a wss_client
    - if a request is an admin/api request
    - if a request is a plain (to be forwarded) http request
    - or if a request is a different protocol (perhaps SSH)

To accomplish the above RVPN uses raw TCP sockets, buffered readers, and a custom Listener.  This allows protocol detection (multiple services on one port)
If we expose 443 and 80 to maximize the ability for tunnel clients and south bound traffic, the RVPN is able to deal with this traffic on a limited number of ports, and the most popular ports.
It is possible now to meter any point of the connection (not Interface Level, rather TCP)

There is now a connection manager that dynamically allows new GenericListeners to start on different ports when needed....

```go
	newListener := NewListenerRegistration(initialPort)
	gl.register <- newListener
```

A new listener is created by sending a NewListenerRegistration on the channel.

```go

	ln, err := net.ListenTCP("tcp", listenAddr)
	if err != nil {
		loginfo.Println("unable to bind", err)
		listenerRegistration.status = listenerFault
		listenerRegistration.err = err
		listenerRegistration.commCh <- listenerRegistration
		return
	}

	listenerRegistration.status = listenerAdded
	listenerRegistration.commCh <- listenerRegistration

```

Once the lister is fired up, I sends back a regisration status to the manager along with the new Listener and status.


### Build

```bash

hcamacho@Hanks-MBP:go-rvpn-server $ go get

```

### Execute RVPN

```bash

hcamacho@Hanks-MBP:go-rvpn-server $ ./go-rvpn-server 
INFO: packer: 2017/02/26 12:43:53.978133 run.go:48: startup
-=-=-=-=-=-=-=-=-=-=
INFO: connection: 2017/02/26 12:43:53.978959 connection_table.go:67: ConnectionTable starting
INFO: connection: 2017/02/26 12:43:53.979000 connection_table.go:50: Reaper waiting for  300  seconds


```

### Connect Tunnel client

```bash

hcamacho@Hanks-MBP:node-tunnel-client $ bin/stunnel.js --locals http://hfc.daplie.me:8443,http://test.hfc.daplie.me:3001,http://127.0.0.1:8080 --stunneld wss://localhost.daplie.me:8443 --secret abc123
[local proxy] http://hfc.daplie.me:8443
[local proxy] http://test.hfc.daplie.me:3001
[local proxy] http://127.0.0.1:8080
[connect] 'wss://localhost.daplie.me:8443'
[open] connected to 'wss://localhost.daplie.me:8443'

```

### Connect Admin

- add a host entry

```

127.0.0.1 tunnel.example.com rvpn.daplie.invalid

```

```bash
browse https://rvpn.daplie.invalid:8443

```

### Send some traffic

- run the RVPN (as above)
- run the tunnel client (as above)
- browse http://127.0.0.1:8443 && https://127.0.0.1:8443
- observe

```bash

hcamacho@Hanks-MBP:node-tunnel-client $ bin/stunnel.js --locals http://hfc.daplie.me:8443,http://test.hfc.daplie.me:3001,http://127.0.0.1:8080 --stunneld wss://localhost.daplie.me:8443 --secret abc123
[local proxy] http://hfc.daplie.me:8443
[local proxy] http://test.hfc.daplie.me:3001
[local proxy] http://127.0.0.1:8080
[connect] 'wss://localhost.daplie.me:8443'
[open] connected to 'wss://localhost.daplie.me:8443'
hello
fe1c495076342c3132372e302e302e312c383038302c3431332c68747470474554202f20485454502f312e310d0a486f73743a203132372e302e302e313a383434330d0a436f6e6e656374696f6e3a206b6565702d616c6976650d0a43616368652d436f6e74726f6c3a206d61782d6167653d300d0a557067726164652d496e7365637572652d52657175657374733a20310d0a557365722d4167656e743a204d6f7a696c6c612f352e3020284d6163696e746f73683b20496e74656c204d6163204f5320582031305f31325f3329204170706c655765624b69742f3533372e333620284b48544d4c2c206c696b65204765636b6f29204368726f6d652f35362e302e323932342e3837205361666172692f3533372e33360d0a4163636570743a20746578742f68746d6c2c6170706c69636174696f6e2f7868746d6c2b786d6c2c6170706c69636174696f6e2f786d6c3b713d302e392c696d6167652f776562702c2a2f2a3b713d302e380d0a4163636570742d456e636f64696e673a20677a69702c206465666c6174652c20736463682c2062720d0a4163636570742d4c616e67756167653a20656e2d55532c656e3b713d302e380d0a0d0a
�IPv4,127.0.0.1,8080,413,httpGET / HTTP/1.1
Host: 127.0.0.1:8443
Connection: keep-alive
Cache-Control: max-age=0
Upgrade-Insecure-Requests: 1
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36
Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8
Accept-Encoding: gzip, deflate, sdch, br
Accept-Language: en-US,en;q=0.8


[exit] loop closed 0

```

Looks like it aborts for some reaon.  I have this problem on on a new installation as well.




-=-=-=-=-=-=

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