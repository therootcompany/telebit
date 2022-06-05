#!/bin/bash

go generate -mod vendor ./...
go build -mod vendor -race -o ./telebit-mgmt cmd/mgmt/*.go

my_version="$(./telebit-mgmt version | cut -d' ' -f4 | sd ':' '.' | sd 'T' '_')"

rm -rf ~/srv/telebit-mgmt/bin/telebit-mgmt
rsync -avhP telebit-mgmt ~/srv/telebit-mgmt/bin/"telebit-mgmt-${my_version}"
ln -s ~/srv/telebit-mgmt/bin/"telebit-mgmt-${my_version}" ~/srv/telebit-mgmt/bin/telebit-mgmt
