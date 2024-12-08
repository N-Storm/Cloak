#!/usr/bin/env bash

set -eu

go install github.com/mitchellh/gox@latest

mkdir -p release

rm -f ./release/*

if [ -z "$v" ]; then
    echo "Version number cannot be null. Run with v=[version] release.sh"
    exit 1
fi

output="{{.Dir}}-{{.OS}}-{{.Arch}}-$v"
osarch="!darwin/arm !darwin/386"

echo "Compiling:"

os="windows linux"
arch="amd64 arm arm64"
pushd cmd/ck-client
CGO_ENABLED=0 gox -ldflags "-X main.version=${v}" -os="$os" -arch="$arch" -osarch="$osarch" -output="$output"
strip ck-client-linux-amd64*
aarch64-linux-gnu-strip ck-client-linux-arm64*
arm-linux-gnueabihf-strip ck-client-linux-arm-*
# for file in ck-client-*; do newname="$(echo ${file} | sed -E 's/(.exe)?$/-noCGO\1/')"; mv ${file} ../../release/${newname} ; done
mv ck-client-* ../../release

CGO_ENABLED=1 gox -ldflags "-X main.version=${v}" -os="$os" -arch="amd64" -osarch="$osarch" -output="$output"
strip ck-client-linux-amd64*
# aarch64-linux-gnu-strip ck-client-linux-arm64*
# arm-linux-gnueabihf-strip ck-client-linux-arm-*
for file in ck-client-*; do newname="$(echo ${file} | sed -E 's/(.exe)?$/-CGO\1/')"; mv ${file} ../../release/${newname} ; done
# mv ck-client-* ../../release
popd

os="linux"
arch="amd64 arm arm64"
pushd cmd/ck-server
CGO_ENABLED=0 gox -ldflags "-X main.version=${v}" -os="$os" -arch="$arch" -osarch="$osarch" -output="$output"
strip ck-server-linux-amd64*
aarch64-linux-gnu-strip ck-server-linux-arm64*
arm-linux-gnueabihf-strip ck-server-linux-arm-*
# for file in ck-server-*; do newname="$(echo ${file} | sed -E 's/(.exe)?$/-noCGO\1/')"; mv ${file} ../../release/${newname} ; done
mv ck-server-* ../../release

CGO_ENABLED=1 gox -ldflags "-X main.version=${v}" -os="$os" -arch="amd64" -osarch="$osarch" -output="$output"
strip ck-server-linux-amd64*
# aarch64-linux-gnu-strip ck-server-linux-arm64*
# arm-linux-gnueabihf-strip ck-server-linux-arm-*
for file in ck-server-*; do newname="$(echo ${file} | sed -E 's/(.exe)?$/-CGO\1/')"; mv ${file} ../../release/${newname} ; done
# mv ck-server-* ../../release
popd

sha256sum release/*
