#!/usr/bin/env bash

#
# Cross compile this go program for every major architecture
#
# TODO: this script should be converted to use gox, which can cross-compile in parallel

VERSION=v0.4.0

env GOOS=darwin GOARCH=386 go build -v -o "bin/darwin_386/ec2-snapper" github.com/josh-padnick/ec2-snapper
env GOOS=darwin GOARCH=amd64 go build -v -o "bin/darwin_amd64/ec2-snapper" github.com/josh-padnick/ec2-snapper

env GOOS=linux GOARCH=386 go build -v -o "bin/linux_386/ec2-snapper" github.com/josh-padnick/ec2-snapper
env GOOS=linux GOARCH=amd64 go build -v -o "bin/linux_amd64/ec2-snapper" github.com/josh-padnick/ec2-snapper
env GOOS=linux GOARCH=arm go build -v -o "bin/linux_arm/ec2-snapper" github.com/josh-padnick/ec2-snapper

env GOOS=windows GOARCH=386 go build -v -o "bin/windows_386/ec2-snapper.exe" github.com/josh-padnick/ec2-snapper
env GOOS=windows GOARCH=amd64 go build -v -o "bin/windows_amd64/ec2-snapper.exe" github.com/josh-padnick/ec2-snapper
