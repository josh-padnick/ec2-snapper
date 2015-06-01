package main

import (
	"github.com/mitchellh/cli"
)

type DeleteCommand struct {
	Ui cli.Ui
}

func (c *DeleteCommand) Help() string {
	return "Help"
}

func (c *DeleteCommand) Synopsis() string {
	return "Delete one or more AMIs based on the args passed in."
}

func (c *DeleteCommand) Run(args []string) int {
	c.Ui.Output("Do something with delete command")
	return 0
}

