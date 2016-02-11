#!/usr/bin/env bash

#
# Cross compile this go program for every major architecture
#

VERSION=0.3.0

GOOS=darwin
GOARCH=386
go build -v -o "${VERSION}/darwin_386/ec2-snapper" github.com/josh-padnick/ec2-snapper

GOOS=darwin
GOARCH=amd64
go build -v -o "${VERSION}/darwin_amd64/ec2-snapper" github.com/josh-padnick/ec2-snapper

GOOS=linux
GOARCH=386
go build -v -o "${VERSION}/linux_386/ec2-snapper" github.com/josh-padnick/ec2-snapper

GOOS=linux
GOARCH=amd64
go build -v -o "${VERSION}/linux_amd64/ec2-snapper" github.com/josh-padnick/ec2-snapper

GOOS=linux
GOARCH=arm
go build -v -o "${VERSION}/linux_arm/ec2-snapper" github.com/josh-padnick/ec2-snapper

GOOS=windows
GOARCH=386
go build -v -o "${VERSION}/windows_386/ec2-snapper.exe" github.com/josh-padnick/ec2-snapper

GOOS=windows
GOARCH=amd64
go build -v -o "${VERSION}/windows_amd64/ec2-snapper.exe" github.com/josh-padnick/ec2-snapper