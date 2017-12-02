#!/bin/sh
PROGRAM=backend_salam
CONFIG=
export GOROOT=/usr/local/go
export GOPATH=/home/salam/golang/GOPATH
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH

supervisorctl stop salam
supervisorctl status salam
go build
supervisorctl start salam
supervisorctl status salam

#eval "./$PROGRAM $CONFIG"

