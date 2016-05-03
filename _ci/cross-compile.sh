#!/usr/bin/env bash

#
# Cross compile this go program for every major architecture. You must install gox to run it:
# https://github.com/mitchellh/gox
#
gox -os "darwin linux windows" -output bin/ec2-snapper_{{.OS}}_{{.Arch}}