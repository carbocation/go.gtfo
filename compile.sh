#!/bin/bash

#Enable negative pattern matching !(prod*).go
shopt -s extglob

WHICHFILES='./!(env_dev*).go'

GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o forum.linux  ${WHICHFILES} && ./nrsc-script forum.linux "static templates" -q
GOOS=darwin GOARCH=amd64 go build -o forum.osx  ${WHICHFILES} && ./nrsc-script forum.osx "static templates" -q

