#!/bin/bash
env=windows
arch=amd64
echo Building client for ${env} ${arch}
GOOS=${env} GOARCH=${arch} godep go build -o out/hearthreplay-updater-${env}-${arch}.exe -ldflags "-s" bootstrap.go

env=windows
arch=386
echo Building bootstrap for ${env} ${arch}
GOOS=${env} GOARCH=${arch} godep go build -o out/hearthreplay-updater-${env}-${arch}.exe -ldflags "-s" bootstrap.go

env=darwin
arch=amd64
echo Building client for ${env} ${arch}
GOOS=${env} GOARCH=${arch} godep go build -o out/hearthreplay-updater-${env}-${arch} -ldflags "-s" bootstrap.go

aws s3 sync out/ s3://update.hearthreplay.com --acl public-read --exclude ".DS_Store"
