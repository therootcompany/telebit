# RVPN Server

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