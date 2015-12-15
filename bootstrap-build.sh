#!/bin/bash
env=windows
for arch in amd64 386
do
	echo Building client for ${env} ${arch}
	GOOS=${env} GOARCH=${arch} godep go build -o out/boot/hearthreplay-updater-${env}-${arch}.exe -ldflags "-s" bootstrap.go
done

env=darwin
arch=amd64
echo Building client for ${env} ${arch}
GOOS=${env} GOARCH=${arch} godep go build -o out/boot/hearthreplay-updater-${env}-${arch} -ldflags "-s" bootstrap.go

aws s3 sync out/boot/ s3://update.hearthreplay.com --acl public-read --exclude ".DS_Store"
