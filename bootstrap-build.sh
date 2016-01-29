#!/bin/bash
set -ef

cd cmd/bootstrap

env=windows
for arch in amd64 386
do
	out=hearthreplay-updater-${env}-${arch}.exe
	echo Building ${out}
	GOOS=${env} GOARCH=${arch} godep go build -o ${out}
	mv ${out} ../../out/bootstrap/
done

env=darwin
arch=amd64
out=hearthreplay-updater
echo Building ${out}
GOOS=${env} GOARCH=${arch} godep go build -o "../../tmp/Hearthreplay Client.app/Contents/MacOS/${out}"

cd ../..

# build a dmg for macos
hdiutil create "Hearthreplay Client.dmg" -volname "Hearthreplay Client" -fs HFS+ -srcfolder "tmp/" -ov

mv "Hearthreplay Client.dmg" out/bootstrap/

echo 'Sync to s3'
aws s3 sync out/bootstrap/ s3://update.hearthreplay.com --acl public-read --exclude ".DS_Store"
