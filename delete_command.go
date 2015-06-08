package main

import (
	"time"
	"flag"

	"github.com/mitchellh/cli"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
//"os"
	"strconv"
)

type DeleteCommand struct {
	Ui 			cli.Ui
	InstanceId 	string
	OlderThan 	string
	DryRun		bool
}

// descriptions for args
var deleteDscrInstanceId = "The EC2 instance from which the AMIs to be deleted were originally created"
var deleteOlderThan = "Delete AMIs older than the specified time; accepts formats like '30d' or '4h'"
var deleteDscrDryRun = "Execute a simulated run. Lists AMIs to be deleted, but does not actually delete them."

func (c *DeleteCommand) Help() string {
	return `ec2-snapper create <args> [--help]

Create an AMI of the given EC2 instance.

Available args are:
--instance      ` + deleteDscrInstanceId + `
--older-than    ` + deleteOlderThan + `
--dry-run       ` + deleteDscrDryRun
}

func (c *DeleteCommand) Synopsis() string {
	return "Delete the specified AMIs"
}

func (c *DeleteCommand) Run(args []string) int {

	// Handle the command-line args
	cmdFlags := flag.NewFlagSet("delete", flag.ExitOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.StringVar(&c.InstanceId, "instance", "", deleteDscrInstanceId)
	cmdFlags.StringVar(&c.OlderThan, "older-than", "", deleteOlderThan)
	cmdFlags.BoolVar(&c.DryRun, "dry-run", false, deleteDscrDryRun)

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// Check for required command-line args
	if c.InstanceId == "" {
		c.Ui.Error("ERROR: The argument '--instance' is required.")
		return 1
	}

	if c.OlderThan == "" {
		c.Ui.Error("ERROR: The argument '--older-than' is required.")
		return 1
	}

	// Create an EC2 service object; AWS region is picked up from the "AWS_REGION" env var.
	svc := ec2.New(nil)

	// Get a list of the existing AMIs that meet our criteria
	resp, err := svc.DescribeImages(&ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("tag:ec2-snapper-instance-id"),
				Values: []*string{&c.InstanceId},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	if len(resp.Images) == 0 {
		c.Ui.Error("No results!")
	} else {
		c.Ui.Info("Found " + strconv.Itoa(len(resp.Images)) + " results.")
		c.Ui.Info(*resp.Images[0].ImageID)
	}


	// Get a list of all snapshots

	// For each AMI
	// - De-register the AMI
	// - Delete the corresponding AMI snapshot

	// Generate a nicely formatted timestamp for right now
//	const dateLayoutForAmiName = "2006-01-02 at 15_04_05 (MST)"
	time.Now()
	//t := time.Now()
//
//	// Create the AMI Snapshot
//	name := c.Name + " " + t.Format(dateLayoutForAmiName)
//	instanceId := c.InstanceId
//	dryRun := c.DryRun
//	noReboot := c.NoReboot
//
//	c.Ui.Output("==> Creating AMI for " + instanceId + "...")
//
//	resp, err := svc.CreateImage(&ec2.CreateImageInput{
//		Name: &name,
//		InstanceID: &instanceId,
//		DryRun: &dryRun,
//		NoReboot: &noReboot })
//	if err != nil {
//		panic(err)
//	}
//
//	// Assign tags to this AMI.  We'll use these when it comes time to delete the AMI
//	c.Ui.Output("==> Adding tags to AMI " + *resp.ImageID + "...")
//
//	//const dateLayoutForTags = "2006-01-02 at 15:04:05 (UTC)"
//	tagName1 := "ec2-snapper-instance-id"
//	tagName2 := "ec2-snapper-snapshot-date"
//	tagValue2 := time.Now().Format(time.RFC3339)
//
//	svc.CreateTags(&ec2.CreateTagsInput{
//		Resources: []*string{resp.ImageID},
//		Tags: []*ec2.Tag{
//			&ec2.Tag{ Key: &tagName1, Value: &c.InstanceId },
//			&ec2.Tag{ Key: &tagName2, Value: &tagValue2 },
//		},
//	})

	c.Ui.Info("==> Success! Deleted a bunch of things.")
	return 0
}

