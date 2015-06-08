package main

import (
	"time"
	"flag"

	"github.com/mitchellh/cli"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"strconv"
	"strings"
	"github.com/aws/aws-sdk-go/service/iam"
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

	// Warn the user that this is a dry run
	if c.DryRun {
		c.Ui.Warn("WARNING: This is a dry run, and no actions will be taken, despite what any output may say!")
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

	// Get the AWS Account ID of the current AWS account
	// We need this to do a more efficient lookup on the snapshot volumes
	// - Per http://docs.aws.amazon.com/general/latest/gr/acct-identifiers.html, we assume the Account Id is always 12 digits
	// - Per http://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html#arn-syntax-iam, we assume the current user's ARN
	//   is always of the form arn:aws:iam::account-id:user/user-name
	svcIam := iam.New(nil)

	respIam, err := svcIam.GetUser(&iam.GetUserInput{})
	awsAccountId := strings.Split(*respIam.User.ARN, ":")[4]
	c.Ui.Output("==> Identified current AWS Account Id as " + awsAccountId)

	// Get a list of every single snapshot in our account
	// (I wasn't able to find a better way to filter these, but suggestions welcome!)
	respDscrSnapshots, err := svc.DescribeSnapshots(&ec2.DescribeSnapshotsInput{
		OwnerIDs: []*string{&awsAccountId},
	})
	if err != nil {
		panic(err)
	}
	c.Ui.Output("==> Found " + strconv.Itoa(len(respDscrSnapshots.Snapshots)) + " snapshots in our account to search through.")

	// Begin deleting AMIs...
	if len(resp.Images) == 0 {
		c.Ui.Error("No AMIs were found for EC2 instance \"" + c.InstanceId + "\"")
	} else {
		for i := 0; i < len(resp.Images); i++ {
			// Step 1: De-register the AMI
			c.Ui.Output(*resp.Images[i].ImageID + ": De-registering...")
			_, err := svc.DeregisterImage(&ec2.DeregisterImageInput{
				DryRun: &c.DryRun,
				ImageID: resp.Images[i].ImageID,
			})
			if err != nil {
				if ! strings.Contains(err.Error(), "DryRunOperation") {
					panic(err)
				}
			}

			// Step 2: Delete the corresponding AMI snapshot
			// Look at the "description" for each Snapshot to see if it contains our AMI id
			snapshotId := ""
			for j := 0; j < len(respDscrSnapshots.Snapshots); j++ {
				if strings.Contains(*respDscrSnapshots.Snapshots[j].Description, *resp.Images[i].ImageID) {
					snapshotId = *respDscrSnapshots.Snapshots[j].SnapshotID
					break
				}
			}

			c.Ui.Output(*resp.Images[i].ImageID + ": Deleting snapshot " + snapshotId + "...")
			svc.DeleteSnapshot(&ec2.DeleteSnapshotInput{
				DryRun: &c.DryRun,
				SnapshotID: &snapshotId,
			})

			c.Ui.Output(*resp.Images[i].ImageID + ": Done!")
			c.Ui.Output("")
		}
	}

	// Generate a nicely formatted timestamp for right now
	//	const dateLayoutForAmiName = "2006-01-02 at 15_04_05 (MST)"
	time.Now()
	//t := time.Now()

	if c.DryRun {
		c.Ui.Info("==> DRY RUN. Had this not been a dry run, " + strconv.Itoa(len(resp.Images)) + " AMI's and their corresponding snapshots would have been deleted.")
	} else {
		c.Ui.Info("==> Success! Deleted " + strconv.Itoa(len(resp.Images)) + " AMI's and their corresponding snapshots.")
	}
	return 0
}

