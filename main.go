package main

import (
	"fmt"
	"os"

	"github.com/mitchellh/cli"
)

func main() {

	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	// CLI stuff
	c := cli.NewCLI("ec2-snapper", "0.3.0")
	c.Args = os.Args[1:]

	c.Commands = map[string]cli.CommandFactory{
		"create": func() (cli.Command, error) {
			return &CreateCommand{
				Ui:	&cli.ColoredUi{
					Ui:	ui,
					OutputColor: cli.UiColorNone,
					ErrorColor:  cli.UiColorRed,
					WarnColor:   cli.UiColorYellow,
					InfoColor:   cli.UiColorGreen,
				},
			}, nil
		},
		"delete": func() (cli.Command, error) {
			return &DeleteCommand{
				Ui: &cli.ColoredUi{
					Ui: ui,
					OutputColor: cli.UiColorNone,
					ErrorColor:  cli.UiColorRed,
					WarnColor:   cli.UiColorYellow,
					InfoColor:   cli.UiColorGreen,
				},
			}, nil
		},
		"version": func() (cli.Command, error) {
			return &VersionCommand{
				cliRef: *c,
			}, nil
		},
	}

	// Confirm that AWS credentials are set as environment
	if os.Getenv("AWS_REGION") == "" {
		fmt.Println("ERROR: You must set the AWS_REGION environment variable to a value like \"us-west-2\" or \"us-east-1\"")
		os.Exit(1)
	}

	exitStatus, err := c.Run()
	if err != nil {
		fmt.Println(os.Stderr, err.Error())
	}

	os.Exit(exitStatus)

}
