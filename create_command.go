package main

import (
	"time"

	"github.com/mitchellh/cli"
	"github.com/awslabs/aws-sdk-go/service/ec2"
)

type CreateCommand struct {
	Ui cli.Ui
}

func (c *CreateCommand) Help() string {
	return "Help"
}

func (c *CreateCommand) Synopsis() string {
	return "Create an AMI of the given EC2 instance"
}

func (c *CreateCommand) Run(args []string) int {
	// Create an EC2 service object; AWS region is picked up from the "AWS_REGION" env var.
	svc := ec2.New(nil)

	// Generate a nicely formatted timestamp for right now
	const layout = "2006-01-02 at 15-04-05 (MST)"
	t := time.Now()

	// Create the AMI Snapshot
	// Todo: Capture these vals from command-line args
	name := "AMI-2"
	name += " " + t.Format(layout)
	instanceId := "i-c724be30"
	dryRun := false
	noReboot := true

	c.Ui.Output("==> Creating AMI for " + instanceId + "...")

	resp, err := svc.CreateImage(&ec2.CreateImageInput{
		Name: &name,
		InstanceID: &instanceId,
		DryRun: &dryRun,
		NoReboot: &noReboot })
	if err != nil {
		panic(err)
	}

	c.Ui.Info("==> Created " + *resp.ImageID + " named \"" + name + "\"")
	return 0
}

