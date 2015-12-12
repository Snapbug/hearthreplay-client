#!/bin/bash
set -e

# get the git commit hash
SHA=$(git log --pretty=format:'%h' -n 1)

# env=windows
# arch=amd64
# echo Building client for ${env} ${arch}
# GOOS=${env} GOARCH=${arch} godep go build -o out/${env}-${arch} -ldflags "-s -X main.version=${SHA}" client.go

# env=windows
# arch=386
# echo Building client for ${env} ${arch}
# GOOS=${env} GOARCH=${arch} godep go build -o out/${env}-${arch} -ldflags "-s -X main.version=${SHA}" client.go

env=darwin
arch=amd64
echo Building client for ${env} ${arch}
GOOS=${env} GOARCH=${arch} godep go build -o out/${env}-${arch} -ldflags "-s -X main.version=${SHA}" client.go

go-selfupdate out ${SHA}
