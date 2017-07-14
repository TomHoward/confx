#!/bin/sh

set -e
set -u

if [ "$#" -lt 1 ]; then
	echo 1>&2 "Usage: $(basename $0) <version>";
	exit 1
fi

readonly VERSION=$1

OS=darwin GOARCH=amd64 go build -o confx-darwin-amd64 -ldflags "-s -w -X main.version=$VERSION -X main.commitId=$(git rev-parse --short HEAD)"
echo "confx-darwin-amd64"
OS=linux GOARCH=amd64 go build -o confx-linux-amd64 -ldflags "-s -w -X main.version=$VERSION -X main.commitId=$(git rev-parse --short HEAD)"
echo "confx-linux-amd64"
