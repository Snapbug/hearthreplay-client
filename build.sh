#!/bin/bash
set -e

# get the git commit hash
SHA=$(git log --pretty=format:"%h" -n 1)

echo Generating bindata.go
go-bindata tmpl/

echo Building version: ${SHA}
echo "{\"version\": \"${SHA}\"}" >| ../server/tmpl/client.json

# commit the change -- creating a new sha -- oh well!
git commit -m "Updating client version" ../server/tmpl/client.json

mkdir out/${SHA}

env=windows
arch=amd64
echo Building client for ${env} ${arch}
GOOS=${env} GOARCH=${arch} godep go build -o out/${SHA}/hsrclient-${env}-${arch}-${SHA} -ldflags "-s -X main.version=${SHA}" client.go bindata.go

env=windows
arch=386
echo Building client for ${env} ${arch}
GOOS=${env} GOARCH=${arch} godep go build -o out/${SHA}/hsrclient-${env}-${arch}-${SHA} -ldflags "-s -X main.version=${SHA}" client.go bindata.go

env=darwin
arch=amd64
echo Building client for ${env} ${arch}
GOOS=${env} GOARCH=${arch} godep go build -o out/${SHA}/hsrclient-${env}-${arch}-${SHA} -ldflags "-s -X main.Version=${SHA}" client.go bindata.go

echo Updating version to s3
aws s3 sync out/${SHA} s3://update.hearthreplay.com --acl public-read

echo Push the changes, pull on server
