package main

import (
	"flag"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/cli"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"math"
)

type DeleteCommand struct {
	Ui 				cli.Ui
	InstanceId 		string
	OlderThan 		string
	RequireAtLeast	int
	DryRun			bool
}

// descriptions for args
var deleteDscrInstanceId = "The EC2 instance from which the AMIs to be deleted were originally created."
var deleteOlderThan = "Delete AMIs older than the specified time; accepts formats like '30d' or '4h'."
var requireAtLeast = "Never delete AMIs such that fewer than this number of AMIs will remain. E.g. require at least 3 AMIs remain."
var deleteDscrDryRun = "Execute a simulated run. Lists AMIs to be deleted, but does not actually delete them."

func (c *DeleteCommand) Help() string {
	return `ec2-snapper create <args> [--help]

Create an AMI of the given EC2 instance.

Available args are:
--instance      	` + deleteDscrInstanceId + `
--older-than    	` + deleteOlderThan + `
--require-at-least      ` + requireAtLeast + `
--dry-run       	` + deleteDscrDryRun
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
	cmdFlags.IntVar(&c.RequireAtLeast, "require-at-least", 0, requireAtLeast)
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

	if c.RequireAtLeast < 0 {
		c.Ui.Error("ERROR: The argument '--require-at-least' must be a positive integer.")
		return 1
	}

	// Warn the user that this is a dry run
	if c.DryRun {
		c.Ui.Warn("WARNING: This is a dry run, and no actions will be taken, despite what any output may say!")
	}

	// Create an EC2 service object; AWS region is picked up from the "AWS_REGION" env var.
	svc := ec2.New(nil)

	// Get a list of the existing AMIs that were created for the given EC2 instance
	resp, err := svc.DescribeImages(&ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("tag:ec2-snapper-instance-id"),
				Values: []*string{&c.InstanceId},
			},
		},
	})
	if err != nil && strings.Contains(err.Error(), "NoCredentialProviders") {
		c.Ui.Error("ERROR: No AWS credentials were found.  Either set the environment variables AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY, or run this program on an EC2 instance that has an IAM Role with the appropriate permissions.")
		return 1
	} else if err != nil {
		panic(err)
	}
	if len(resp.Images) == 0 {
		c.Ui.Error("No AMIs were found for EC2 instance \"" + c.InstanceId + "\"")
		return 0
	}

	// Check that at least the --require-at-least number of AMIs exists
	// - Note that even if this passes, we still want to avoid deleting so many AMIs that we go below the threshold
	if len(resp.Images) <= c.RequireAtLeast {
		c.Ui.Info("NO ACTION TAKEN. There are currently " + strconv.Itoa(len(resp.Images)) + " AMIs, and --require-at-least=" + strconv.Itoa(c.RequireAtLeast) + " so no further action can be taken.")
		return 0
	}

	// Get the AWS Account ID of the current AWS account
	// We need this to do a more efficient lookup on the snapshot volumes
	awsAccountId := *resp.Images[0].OwnerID
	c.Ui.Output("==> Identified current AWS Account Id as " + awsAccountId)

	// Parse our date range
	match, _ := regexp.MatchString("^[0-9]*(h|d|m)$", c.OlderThan)
	if ! match {
		c.Ui.Error("The --older-than value of \"" + c.OlderThan + "\" is not formatted properly.  Use formats like 30d or 24h")
		return 0
	}

	var minutes float64
	var hours float64

	// We were given a time like "12h"
	if match, _ := regexp.MatchString("^[0-9]*(h)$", c.OlderThan); match {
		hours, _ = strconv.ParseFloat(c.OlderThan[0:len(c.OlderThan)-1], 64)
	}

	// We were given a time like "15d"
	if match, _ := regexp.MatchString("^[0-9]*(d)$", c.OlderThan); match {
		hours, _ = strconv.ParseFloat(c.OlderThan[0:len(c.OlderThan)-1], 64)
		hours *= 24
	}

	// We were given a time like "5m"
	if match, _ := regexp.MatchString("^[0-9]*(m)$", c.OlderThan); match {
		minutes, _ = strconv.ParseFloat(c.OlderThan[0:len(c.OlderThan)-1], 64)
		hours = minutes/60
	}

	// Now filter the AMIs to only include those within our date range
	var filteredAmis[]*ec2.Image
	for i := 0; i < len(resp.Images); i++ {
		now := time.Now()
		creationDate, err := time.Parse(time.RFC3339Nano, *resp.Images[i].CreationDate)
		if err != nil {
			panic(err)
		}

		duration := now.Sub(creationDate)

		if duration.Hours() > hours {
			filteredAmis = append(filteredAmis, resp.Images[i])
		}
	}
	c.Ui.Output("==> Found " + strconv.Itoa(len(filteredAmis)) + " total AMI(s) for deletion.")

	if len(filteredAmis) == 0 {
		c.Ui.Error("No AMIs to delete.")
		return 0
	}

	// Get a list of every single snapshot in our account
	// (I wasn't able to find a better way to filter these, but suggestions welcome!)
	respDscrSnapshots, err := svc.DescribeSnapshots(&ec2.DescribeSnapshotsInput{
		OwnerIDs: []*string{&awsAccountId},
	})
	if err != nil {
		panic(err)
	}
	c.Ui.Output("==> Found " + strconv.Itoa(len(respDscrSnapshots.Snapshots)) + " total snapshots in this account.")

	// Compute whether we should delete fewer AMIs to adhere to our --require-at-least requirement
	var numTotalAmis = len(resp.Images)
	var numFilteredAmis = len(filteredAmis)
	var numAmisToRemainAfterDelete = numTotalAmis - numFilteredAmis
	var numAmisToRemoveFromFiltered = math.Max(0.0, float64(c.RequireAtLeast - numAmisToRemainAfterDelete))

	if numAmisToRemoveFromFiltered > 0.0 {
		c.Ui.Output("==> Only deleting " + strconv.Itoa(len(filteredAmis) - int(numAmisToRemoveFromFiltered)) + " total AMIs to honor '--require-at-least=" + strconv.Itoa(c.RequireAtLeast) + "'.")
	}

	// Begin deleting AMIs...
	for i := 0; i < len(filteredAmis) - int(numAmisToRemoveFromFiltered); i++ {
		// Step 1: De-register the AMI
		c.Ui.Output(*filteredAmis[i].ImageID + ": De-registering AMI named \"" + *filteredAmis[i].Name + "\"...")
		_, err := svc.DeregisterImage(&ec2.DeregisterImageInput{
			DryRun: &c.DryRun,
			ImageID: filteredAmis[i].ImageID,
		})
		if err != nil {
			if ! strings.Contains(err.Error(), "DryRunOperation") {
				panic(err)
			}
		}

		// Step 2: Delete the corresponding AMI snapshot
		// Look at the "description" for each Snapshot to see if it contains our AMI id
		var snapshotIds []string
		for _, snapshot := range respDscrSnapshots.Snapshots {
			if strings.Contains(*snapshot.Description, *filteredAmis[i].ImageID) {
				snapshotIds = append(snapshotIds, *snapshot.SnapshotID)
			}
		}

		// Delete all snapshots that were found
		c.Ui.Output(*filteredAmis[i].ImageID + ": Found " + strconv.Itoa(len(snapshotIds)) + " snapshot(s) to delete")
		for _, snapshotId := range snapshotIds {
			c.Ui.Output(*filteredAmis[i].ImageID + ": Deleting snapshot " + snapshotId + "...")
			svc.DeleteSnapshot(&ec2.DeleteSnapshotInput{
				DryRun: &c.DryRun,
				SnapshotID: &snapshotId,
			})
		}

		c.Ui.Output(*filteredAmis[i].ImageID + ": Done!")
		c.Ui.Output("")
	}

	if c.DryRun {
		c.Ui.Info("==> DRY RUN. Had this not been a dry run, " + strconv.Itoa(len(filteredAmis)) + " AMI's and their corresponding snapshots would have been deleted.")
	} else {
		c.Ui.Info("==> Success! Deleted " + strconv.Itoa(len(filteredAmis) - int(numAmisToRemoveFromFiltered)) + " AMI's and their corresponding snapshots.")
	}
	return 0
}

