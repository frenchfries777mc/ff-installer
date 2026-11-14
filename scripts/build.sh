#!/bin/sh
set -e
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H=windowsgui" -o setup.exe .
