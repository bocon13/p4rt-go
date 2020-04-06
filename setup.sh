#!/bin/bash
# Use this script to download and compile P4RT

GOPATH=${GOPATH:-$HOME/go}
PATH=$PATH:$GOPATH/bin

set -x

# Install protoc for go
go get -u github.com/golang/protobuf/protoc-gen-go

# Install P4RT client deps
go get -d -v -u github.com/p4lang/p4runtime
go get -d -v -u github.com/google/protobuf
go get -d -v -u github.com/googleapis/googleapis
go get -v -u golang.org/x/sys/unix
go get -v -u github.com/bocon13/p4rt-go/p4rt
go get -v -u github.com/bocon13/p4rt-go/bin

cd $GOPATH/src || exit
protoc -Igithub.com/p4lang/p4runtime/proto \
  --go_out=github.com/p4lang/p4runtime/proto \
  github.com/p4lang/p4runtime/proto/p4/config/v1/p4types.proto \
  github.com/p4lang/p4runtime/proto/p4/config/v1/p4info.proto
protoc -I=github.com/google/protobuf/src:github.com/googleapis/googleapis:github.com/p4lang/p4runtime/proto \
  --go_out=plugins=grpc,Mp4/config/v1/p4info.proto=github.com/p4lang/p4runtime/proto/p4/config/v1:github.com/p4lang/p4runtime/proto \
  github.com/p4lang/p4runtime/proto/p4/v1/p4data.proto \
  github.com/p4lang/p4runtime/proto/p4/v1/p4runtime.proto
