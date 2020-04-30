# Docker Deployment for RVPN

-   install docker 1.13, or the latest stable CE release (Testing on MAC using 17.03.0-ce-mac1 (15583))
-   validate installation

```bash
hcamacho@Hanks-MBP:rvpn-docker $ docker-compose --version
docker-compose version 1.11.2, build dfed245

hcamacho@Hanks-MBP:rvpn-docker $ docker --version
Docker version 17.03.0-ce, build 60ccb22
```

-   checkout code into gopath

```bash
cd $GOPATH/src/git.coolaj86.com/coolaj86
git clone git@git.coolaj86.com:coolaj86/go-telebitd.git

cd telebit-relay
go get

```

## Execute Container Deployment

-   prep

```bash
cd rvpn-docker
hcamacho@Hanks-MBP:rvpn-docker $ docker-compose build
Building rvpn
Step 1/3 : FROM golang:1.7.5
 ---> 5cfb16b630ef
Step 2/3 : LABEL maintainer "henry.f.camacho@gmail.com"
 ---> Running in 5cdffef8e33d
 ---> f7e09c097612
Removing intermediate container 5cdffef8e33d
Step 3/3 : WORKDIR "/telebit-relay"
 ---> 182aa9c814f2
Removing intermediate container f136550d6d48
Successfully built 182aa9c814f2

```

-   execute container

```bash
hcamacho@Hanks-MBP:rvpn-docker $ docker-compose up
Creating network "rvpndocker_default" with the default driver
Creating rvpndocker_rvpn_1
Attaching to rvpndocker_rvpn_1
rvpn_1  | INFO: packer: 2017/03/04 18:13:00.994955 main.go:47: startup
rvpn_1  | -=-=-=-=-=-=-=-=-=-=
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:01.000063 conn_tracking.go:25: Tracking Running
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:01.000067 connection_table.go:67: ConnectionTable starting
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:01.000214 connection_table.go:50: Reaper waiting for  300  seconds
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:00.999757 manager.go:77: ConnectionTable starting
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:01.000453 manager.go:84: &{map[] 0xc4200124e0 0xc420012540}
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:01.000505 manager.go:100: register fired 8443
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:01.000613 manager.go:110: listener starting up  8443
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:01.000638 manager.go:111: &{map[] 0xc4200124e0 0xc420012540}
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:01.000696 listener_generic.go:55: :⃻
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.242287 listener_generic.go:87: Deadtime reached
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.242596 listener_generic.go:114: conn &{0xc420120000 0xc42011e000} 172.18.0.2:8443 172.18.0.1:38148
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.242627 listener_generic.go:131: TLS10
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.242641 listener_generic.go:148: Handle Encryption
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.242654 one_conn.go:22: Accept 172.18.0.2:8443 172.18.0.1:38148
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.242699 listener_generic.go:177: handle Stream
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.242722 listener_generic.go:178: conn &{0xc420120060 0xc420126000} 172.18.0.2:8443 172.18.0.1:38148
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.266803 listener_generic.go:191: identifed HTTP
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.267797 listener_generic.go:207: Valid WSS dected...sending to handler
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.267926 one_conn.go:32: addr
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.267947 one_conn.go:22: Accept 172.18.0.2:8443 172.18.0.1:38148
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.268045 one_conn.go:17: Accept
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.268062 one_conn.go:27: close
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.268066 listener_generic.go:421: Serve error:  EOF
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.268707 listener_generic.go:366: HandleFunc /
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.268727 listener_generic.go:369: websocket opening  172.18.0.1:38148   localhost.rootprojects.org:8443
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.269264 listener_generic.go:397: before connection table
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.269321 connection_table.go:79: register fired
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.269523 connection_table.go:90: adding domain  hfc.rootprojects.org  to connection  172.18.0.1:38148
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.269602 connection_table.go:90: adding domain  test1.hfc.rootprojects.org  to connection  172.18.0.1:38148
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.269821 listener_generic.go:410: connection registration accepted  &{0xc42012af00 172.18.0.1:38148 0xc420120ea0 [hfc.rootprojects.org test1.hfc.rootprojects.org] 0xc4200f49c0}
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.270168 connection.go:200: Reader Start  &{0xc420104990 0xc420077560 map[hfc.rootprojects.org:0xc4201ee7a0 test1.hfc.rootprojects.org:0xc4201ee7c0] 0xc42012af00 0xc420120f00 172.18.0.1:38148 0 0 {63624247982 269492963 0x8392a0} {0 0 <nil>} [hfc.rootprojects.org test1.hfc.rootprojects.org] 0xc4200f49c0 true}
rvpn_1  | INFO: genericlistener: 2017/03/04 18:13:02.270281 connection.go:242: Writer Start  &{0xc420104990 0xc420077560 map[hfc.rootprojects.org:0xc4201ee7a0 test1.hfc.rootprojects.org:0xc4201ee7c0] 0xc42012af00 0xc420120f00 172.18.0.1:38148 0 0 {63624247982 269492963 0x8392a0} {0 0 <nil>} [hfc.rootprojects.org test1.hfc.rootprojects.org] 0xc4200f49c0 true}

```

The line "Connection Registration Accepted indicates a client WSS registered, was authenticated and registered its domains with the RVPN
