package main

import (
	"github.com/mitchellh/cli"
	"fmt"
	"os"
)

type VersionCommand struct {
	cliRef cli.CLI
}

func (c *VersionCommand) Help() string {
	return `ec2-snapper version

Return the version of ec2-snapper.`
}

func (c *VersionCommand) Synopsis() string {
	return "Return the version of ec2-snapper"
}

func (c *VersionCommand) Run(args []string) int {
	fmt.Fprintf(os.Stdout,"You are running ec2-snapper version %s.\n", c.cliRef.Version)
	return 0
}
