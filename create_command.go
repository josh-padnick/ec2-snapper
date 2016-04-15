package main

import (
	"flag"
	"time"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mitchellh/cli"
	"errors"
	"fmt"
)

type CreateCommand struct {
	Ui           cli.Ui
	AwsRegion    string
	InstanceId   string
	InstanceName string
	AmiName      string
	DryRun       bool
	NoReboot     bool
}

const EC2_SNAPPER_INSTANCE_ID_TAG = "ec2-snapper-instance-id"

// descriptions for args
var createDscrAwsRegion = "The AWS region to use (e.g. us-west-2)"
var createDscrInstanceId = "The id of the instance from which to create the AMI"
var createDscrInstanceName = "The name (from tags) of the instance from which to create the AMI"
var createDscrAmiName = "The name of the AMI; the current timestamp will be automatically appended"
var createDscrDryRun = "Execute a simulated run"
var createDscrNoReboot = "If true, do not reboot the instance before creating the AMI. It is preferable to reboot the instance to guarantee a consistent filesystem when taking the snapshot, but the likelihood of an inconsistent snapshot is very low."

func (c *CreateCommand) Help() string {
	return `ec2-snapper create <args> [--help]

Create an AMI of the given EC2 instance.

Available args are:
--region      	` + createDscrAwsRegion + `
--instance-id   ` + createDscrInstanceId + `
--instance-name ` + createDscrInstanceName + `
--ami-name      ` + createDscrAmiName + `
--dry-run       ` + createDscrDryRun + `
--no-reboot     ` + createDscrNoReboot
}

func (c *CreateCommand) Synopsis() string {
	return "Create an AMI of the given EC2 instance"
}

func (c *CreateCommand) Run(args []string) int {

	// Handle the command-line args
	cmdFlags := flag.NewFlagSet("create", flag.ExitOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.StringVar(&c.AwsRegion, "region", "", createDscrAwsRegion)
	cmdFlags.StringVar(&c.InstanceId, "instance-id", "", createDscrInstanceId)
	cmdFlags.StringVar(&c.InstanceName, "instance-name", "", createDscrInstanceName)
	cmdFlags.StringVar(&c.AmiName, "ami-name", "", createDscrAmiName)
	cmdFlags.BoolVar(&c.DryRun, "dry-run", false, createDscrDryRun)
	cmdFlags.BoolVar(&c.NoReboot, "no-reboot", true, createDscrNoReboot)

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	if _, err := create(*c); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	return 0
}

func create(c CreateCommand) (string, error) {
	snapshotId := ""

	if err := validateCreateArgs(c); err != nil {
		return snapshotId, err
	}

	session := session.New(&aws.Config{Region: &c.AwsRegion})
	svc := ec2.New(session)

	if c.InstanceId == "" {
		instanceId, err := getInstanceIdByName(c.InstanceName, svc, c.Ui)
		if err != nil {
			return snapshotId, err
		}
		c.InstanceId = instanceId
	}

	// Generate a nicely formatted timestamp for right now
	const dateLayoutForAmiName = "2006-01-02 at 15_04_05 (MST)"
	t := time.Now()

	// Create the AMI Snapshot
	name := c.AmiName + " - " + t.Format(dateLayoutForAmiName)

	c.Ui.Output("==> Creating AMI for " + c.InstanceId + "...")

	resp, err := svc.CreateImage(&ec2.CreateImageInput{
		Name: &name,
		InstanceId: &c.InstanceId,
		DryRun: &c.DryRun,
		NoReboot: &c.NoReboot })
	if err != nil && strings.Contains(err.Error(), "NoCredentialProviders") {
		return snapshotId, errors.New("ERROR: No AWS credentials were found.  Either set the environment variables AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY, or run this program on an EC2 instance that has an IAM Role with the appropriate permissions.")
	} else if err != nil {
		return snapshotId, err
	}

	// Sleep here to give time for AMI to get found
	time.Sleep(3000 * time.Millisecond)

	// Assign tags to this AMI.  We'll use these when it comes time to delete the AMI
	snapshotId = *resp.ImageId
	c.Ui.Output("==> Adding tags to AMI " + snapshotId + "...")

	_, tagsErr := svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{&snapshotId},
		Tags: []*ec2.Tag{
			&ec2.Tag{ Key: aws.String(EC2_SNAPPER_INSTANCE_ID_TAG), Value: &c.InstanceId },
			&ec2.Tag{ Key: aws.String("Name"), Value: &c.AmiName },
		},
	})

	if tagsErr != nil {
		return snapshotId, tagsErr
	}

	// Check the status of the AMI
	respDscrImages, err := svc.DescribeImages(&ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("image-id"),
				Values: []*string{&snapshotId},
			},
		},
	})
	if err != nil {
		return snapshotId, err
	}

	// If no AMI at all was found, throw an error
	if len(respDscrImages.Images) == 0 {
		return snapshotId, errors.New("ERROR: Could not find the AMI just created.")
	}

	// If the AMI's status is failed throw an error
	if *respDscrImages.Images[0].State == "failed" {
		return snapshotId, errors.New("ERROR: AMI was created but entered a state of 'failed'. This is an AWS issue. Please re-run this command.  Note that you will need to manually de-register the AMI in the AWS console or via the API.")
	}

	// Announce success
	c.Ui.Info("==> Success! Created " + snapshotId + " named \"" + name + "\"")
	return snapshotId, nil
}

func validateCreateArgs(c CreateCommand) error {
	if c.AwsRegion == "" {
		return errors.New("ERROR: The argument '--region' is required.")
	}

	if (c.InstanceId == "" && c.InstanceName == "") || (c.InstanceId != "" && c.InstanceName != "") {
		return errors.New("ERROR: You must specify exactly one of '--instance-id' or '--instance-name'.")
	}

	if c.AmiName == "" {
		return errors.New("ERROR: The argument '--name' is required.")
	}

	return nil
}

func getInstanceIdByName(instanceName string, svc *ec2.EC2, ui cli.Ui) (string, error) {
	ui.Output(fmt.Sprintf("Looking up id for instance named %s", instanceName))

	nameTagFilter := ec2.Filter{
		Name: aws.String("tag:Name"),
		Values: []*string{aws.String(instanceName)},
	}

	result, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{Filters: []*ec2.Filter{&nameTagFilter}})
	if err != nil {
		return "", err
	}

	if len(result.Reservations) != 1 {
		return "", errors.New(fmt.Sprintf("Expected to find one result for instance name %s, but found %d", instanceName, len(result.Reservations)))
	}

	reservation := result.Reservations[0]

	if len(reservation.Instances) != 1 {
		return "", errors.New(fmt.Sprintf("Expected to find one instance with instance name %s, but found %d", instanceName, len(reservation.Instances)))
	}

	instance := reservation.Instances[0]
	ui.Output(fmt.Sprintf("Found id %s for instance named %s", *instance.InstanceId, instanceName))

	return *instance.InstanceId, nil
}