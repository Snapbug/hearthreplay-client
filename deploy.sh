#!/bin/bash
set -e

ROOT="hearthreplay-client"

function getchecksign() {
	checksum=$(openssl dgst -sha256 $1 | sed 's/^.* //')
	signature=$(openssl dgst -sha256 -sign ../../private.pem -keyform PEM $1| base64)
	echo "{\"version\": \"$2\", \"checksum\": \"${checksum}\", \"signature\":\"${signature}\"}" >| $3
}

# get the git commit hash
SHA=$(git log --pretty=format:"%h" -n 1)

echo Building version: ${SHA}

mkdir out/${SHA}

cd cmd/client

echo Generating bindata.go
go-bindata tmpl/

env=windows
for arch in amd64 386
do
	out=${ROOT}-${SHA}-${env}-${arch}
	echo Building ${out}
	GOOS=${env} GOARCH=${arch} godep go build -o ${out} -ldflags "-X main.Version=${SHA} -X main.debug=${SHA}"
	getchecksign ${out} ${SHA} ${env}-${arch}
	mv ${out} ../../out/${SHA}/
done

env=darwin
arch=amd64
out=${ROOT}-${SHA}-${env}-${arch}
echo Building ${out}
GOOS=${env} GOARCH=${arch} godep go build -o ${out} -ldflags "-X main.Version=${SHA}"
getchecksign ${out} ${SHA} ${env}-${arch}
mv ${out} ../../out/${SHA}/

cd ../..

echo 'Sync to s3'
aws s3 sync out/${SHA} s3://update.hearthreplay.com --acl public-read --exclude ".DS_Store"
