package main

import (
	"time"
	"flag"

	"github.com/mitchellh/cli"
	"github.com/awslabs/aws-sdk-go/service/ec2"
	//"os"
)

type CreateCommand struct {
	Ui 			cli.Ui
	InstanceId 	string
	Name 		string
	DryRun		bool
	NoReboot	bool
}

// descriptions for args
const dscrInstanceId = "The instance from which to create the AMI"
const dscrName = "The name of the AMI; the current timestamp will be automatically appended"
const dscrDryRun = "Execute a simulated run"
const dscrNoReboot = "If true, do not reboot the instance before creating the AMI. It is preferable to reboot the instance to guarantee a consistent filesystem when taking the snapshot, but the likelihood of an inconsistent snapshot is very low."

func (c *CreateCommand) Help() string {
	return `ec2-snapper create <args> [--help]

Create an AMI of the given EC2 instance.

Available args are:
--instance      ` + dscrInstanceId + `
--name          ` + dscrName + `
--dry-run       ` + dscrDryRun + `
--no-reboot     ` + dscrNoReboot
}

func (c *CreateCommand) Synopsis() string {
	return "Create an AMI of the given EC2 instance"
}

func (c *CreateCommand) Run(args []string) int {

	// Handle the command-line args
	cmdFlags := flag.NewFlagSet("create", flag.ExitOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.StringVar(&c.InstanceId, "instance", "", dscrInstanceId)
	cmdFlags.StringVar(&c.Name, "name", "", dscrName)
	cmdFlags.BoolVar(&c.DryRun, "dry-run", false, dscrDryRun)
	cmdFlags.BoolVar(&c.NoReboot, "no-reboot", true, dscrNoReboot)

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// Check for required command-line args
	if c.InstanceId == "" {
		c.Ui.Error("ERROR: The argument '--instance' is required.")
		return 1
	}

	if c.Name == "" {
		c.Ui.Error("ERROR: The argument '--name' is required.")
		return 1
	}

	// Create an EC2 service object; AWS region is picked up from the "AWS_REGION" env var.
	svc := ec2.New(nil)

	// Generate a nicely formatted timestamp for right now
	const layout = "2006-01-02 at 15_04_05 (MST)"
	t := time.Now()

	// Create the AMI Snapshot
	name := c.Name + " " + t.Format(layout)
	instanceId := c.InstanceId
	dryRun := c.DryRun
	noReboot := c.NoReboot

	c.Ui.Output("==> Creating AMI for " + instanceId + "...")

	resp, err := svc.CreateImage(&ec2.CreateImageInput{
		Name: &name,
		InstanceID: &instanceId,
		DryRun: &dryRun,
		NoReboot: &noReboot })
	if err != nil {
		panic(err)
	}

	// Assign tags to this AMI.  We'll use these when it comes time to delete the AMI
	tagName := "ec2-snapper-instance-id"

	c.Ui.Output("==> Adding tag " + tagName + " to AMI " + *resp.ImageID + "...")
	svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{resp.ImageID},
		Tags: []*ec2.Tag{&ec2.Tag{ Key: &tagName, Value: &c.InstanceId }},
	})

	c.Ui.Info("==> Success! Created " + *resp.ImageID + " named \"" + name + "\"")
	return 0
}

