#!/usr/bin/env bash

#
# Cross compile this go program for every major architecture
#

VERSION=v0.3.0

rm -Rf ${VERSION}
rm "${VERSION}.zip"

env GOOS=darwin GOARCH=386 go build -v -o "${VERSION}/darwin_386/ec2-snapper" github.com/josh-padnick/ec2-snapper
env GOOS=darwin GOARCH=amd64 go build -v -o "${VERSION}/darwin_amd64/ec2-snapper" github.com/josh-padnick/ec2-snapper

env GOOS=linux GOARCH=386 go build -v -o "${VERSION}/linux_386/ec2-snapper" github.com/josh-padnick/ec2-snapper
env GOOS=linux GOARCH=amd64 go build -v -o "${VERSION}/linux_amd64/ec2-snapper" github.com/josh-padnick/ec2-snapper
env GOOS=linux GOARCH=arm go build -v -o "${VERSION}/linux_arm/ec2-snapper" github.com/josh-padnick/ec2-snapper

env GOOS=windows GOARCH=386 go build -v -o "${VERSION}/windows_386/ec2-snapper.exe" github.com/josh-padnick/ec2-snapper
env GOOS=windows GOARCH=amd64 go build -v -o "${VERSION}/windows_amd64/ec2-snapper.exe" github.com/josh-padnick/ec2-snapper

zip -r "${VERSION}.zip" "${VERSION}"