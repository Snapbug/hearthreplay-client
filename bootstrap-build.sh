#!/bin/bash
set -ef

env=windows
for arch in amd64 386
do
	echo Building client for ${env} ${arch}
	GOOS=${env} GOARCH=${arch} godep go build -o out/boot/hearthreplay-updater-${env}-${arch}.exe -ldflags "-s" bootstrap.go
done

env=darwin
arch=amd64
echo Building client for ${env} ${arch}
GOOS=${env} GOARCH=${arch} godep go build -o "tmp/Hearthreplay Client.app/Contents/MacOS/hearthreplay-client-updater" -ldflags "-s" bootstrap.go

# build a dmg for macos
hdiutil create "Hearthreplay Client.dmg" -volname "Hearthreplay Client" -fs HFS+ -srcfolder "tmp/" -ov

mv "Hearthreplay Client.dmg" out/boot

aws s3 sync out/boot/ s3://update.hearthreplay.com --acl public-read --exclude ".DS_Store"
