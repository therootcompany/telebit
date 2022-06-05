#!/bin/bash

go generate -mod vendor ./...
go build -mod vendor -race -o ./telebit-relay cmd/telebit/*.go

my_version="$(./telebit-relay version | cut -d' ' -f4 | sd ':' '.' | sd 'T' '_')"

rm -rf ~/srv/telebit-relay/bin/telebit-relay
rsync -avhP telebit-relay ~/srv/telebit-relay/bin/"telebit-relay-${my_version}"
ln -s ~/srv/telebit-relay/bin/"telebit-relay-${my_version}" ~/srv/telebit-relay/bin/telebit-relay
sudo setcap 'cap_net_bind_service=+ep' ~/srv/telebit-relay/bin/"telebit-relay-${my_version}"
