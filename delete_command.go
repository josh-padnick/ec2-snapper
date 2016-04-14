package main

import (
	"flag"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/cli"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"math"
	"errors"
)

type DeleteCommand struct {
	Ui 			cli.Ui
	AwsRegion 		string
	InstanceId 		string
	InstanceName 		string
	OlderThan 		string
	RequireAtLeast		int
	DryRun			bool
}

// descriptions for args
var deleteDscrAwsRegion = "The AWS region to use (e.g. us-west-2)"
var deleteDscrInstanceId = "The ID of the EC2 instance from which the AMIs to be deleted were originally created."
var deleteDscrInstanceName = "The name (from tags) of the EC2 instance from which the AMIs to be deleted were originally created."
var deleteOlderThan = "Delete AMIs older than the specified time; accepts formats like '30d' or '4h'."
var requireAtLeast = "Never delete AMIs such that fewer than this number of AMIs will remain. E.g. require at least 3 AMIs remain."
var deleteDscrDryRun = "Execute a simulated run. Lists AMIs to be deleted, but does not actually delete them."

func (c *DeleteCommand) Help() string {
	return `ec2-snapper create <args> [--help]

Create an AMI of the given EC2 instance.

Available args are:
--region      		` + deleteDscrAwsRegion + `
--instance-id      	` + deleteDscrInstanceId + `
--instance-name      	` + deleteDscrInstanceName + `
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
	cmdFlags.Usage = func() {
		c.Ui.Output(c.Help())
	}

	cmdFlags.StringVar(&c.AwsRegion, "region", "", deleteDscrAwsRegion)
	cmdFlags.StringVar(&c.InstanceId, "instance-id", "", deleteDscrInstanceId)
	cmdFlags.StringVar(&c.InstanceName, "instance-name", "", deleteDscrInstanceId)
	cmdFlags.StringVar(&c.OlderThan, "older-than", "", deleteOlderThan)
	cmdFlags.IntVar(&c.RequireAtLeast, "require-at-least", 0, requireAtLeast)
	cmdFlags.BoolVar(&c.DryRun, "dry-run", false, deleteDscrDryRun)

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	if err := deleteSnapshots(*c); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	return 0
}

func deleteSnapshots(c DeleteCommand) error {
	if err := validateDeleteArgs(c); err != nil {
		return err
	}

	if c.DryRun {
		c.Ui.Warn("WARNING: This is a dry run, and no actions will be taken, despite what any output may say!")
	}

	// Create an EC2 service object; AWS region is picked up from the "AWS_REGION" env var.
	session := session.New(&aws.Config{Region: &c.AwsRegion})
	svc := ec2.New(session)

	if c.InstanceId == "" {
		instanceId, err := getInstanceIdByName(c.InstanceName, svc, c.Ui)
		if err != nil {
			return err
		}
		c.InstanceId = instanceId
	}

	images, err := findImages(c.InstanceId, svc)
	if err != nil {
		return err
	}
	// Check that at least the --require-at-least number of AMIs exists
	// - Note that even if this passes, we still want to avoid deleting so many AMIs that we go below the threshold
	if len(images) <= c.RequireAtLeast {
		c.Ui.Info("NO ACTION TAKEN. There are currently " + strconv.Itoa(len(images)) + " AMIs, and --require-at-least=" + strconv.Itoa(c.RequireAtLeast) + " so no further action can be taken.")
		return nil
	}

	// Get the AWS Account ID of the current AWS account
	// We need this to do a more efficient lookup on the snapshot volumes
	awsAccountId := *images[0].OwnerId
	c.Ui.Output("==> Identified current AWS Account Id as " + awsAccountId)

	hours, err := parseOlderThanToHours(c.OlderThan)
	if err != nil {
		return err
	}

	filteredAmis, err := filterImagesByDateRange(images, hours)
	if err != nil {
		return err
	}
	c.Ui.Output("==> Found " + strconv.Itoa(len(filteredAmis)) + " total AMI(s) for deletion.")

	if len(filteredAmis) == 0 {
		c.Ui.Warn("No AMIs to delete.")
		return nil
	}

	allSnapshots, err := getAllSnapshots(awsAccountId, svc)
	if err != nil {
		return err
	}
	c.Ui.Output("==> Found " + strconv.Itoa(len(allSnapshots)) + " total snapshots in this account.")

	var numAmisToRemoveFromFiltered = computeNumAmisToRemove(images, filteredAmis, c.RequireAtLeast)
	if numAmisToRemoveFromFiltered > 0.0 {
		c.Ui.Output("==> Only deleting " + strconv.Itoa(len(filteredAmis) - int(numAmisToRemoveFromFiltered)) + " total AMIs to honor '--require-at-least=" + strconv.Itoa(c.RequireAtLeast) + "'.")
	}

	if err := deleteAmis(filteredAmis, allSnapshots, numAmisToRemoveFromFiltered, svc, c.DryRun, c.Ui); err != nil {
		return err
	}

	if c.DryRun {
		c.Ui.Info("==> DRY RUN. Had this not been a dry run, " + strconv.Itoa(len(filteredAmis)) + " AMI's and their corresponding snapshots would have been deleted.")
	} else {
		c.Ui.Info("==> Success! Deleted " + strconv.Itoa(len(filteredAmis) - int(numAmisToRemoveFromFiltered)) + " AMI's and their corresponding snapshots.")
	}
	return nil
}

// Get a list of every single snapshot in our account
// (I wasn't able to find a better way to filter these, but suggestions welcome!)
func getAllSnapshots(awsAccountId string, svc *ec2.EC2) ([]*ec2.Snapshot, error) {
	var noSnapshots []*ec2.Snapshot

	respDscrSnapshots, err := svc.DescribeSnapshots(&ec2.DescribeSnapshotsInput{
		OwnerIds: []*string{&awsAccountId},
	})
	if err != nil {
		return noSnapshots, err
	}

	return respDscrSnapshots.Snapshots, nil
}

// Compute whether we should delete fewer AMIs to adhere to our --require-at-least requirement
func computeNumAmisToRemove(images []*ec2.Image, filteredAmis []*ec2.Image, requireAtLeast int) float64 {
	var numTotalAmis = len(images)
	var numFilteredAmis = len(filteredAmis)
	var numAmisToRemainAfterDelete = numTotalAmis - numFilteredAmis
	return math.Max(0.0, float64(requireAtLeast - numAmisToRemainAfterDelete))
}

// Check for required command-line args
func validateDeleteArgs(c DeleteCommand) error {
	if c.AwsRegion == "" {
		return errors.New("ERROR: The argument '--region' is required.")
	}

	if (c.InstanceId == "" && c.InstanceName == "") || (c.InstanceId != "" && c.InstanceName != "") {
		return errors.New("ERROR: You must specify exactly one of '--instance-id' or '--instance-name'.")
	}

	if c.OlderThan == "" {
		return errors.New("ERROR: The argument '--older-than' is required.")
	}

	if c.RequireAtLeast < 0 {
		return errors.New("ERROR: The argument '--require-at-least' must be a positive integer.")
	}

	return nil
}

// Now filter the AMIs to only include those within our date range
func filterImagesByDateRange(images []*ec2.Image, olderThanHours float64) ([]*ec2.Image, error) {
	var filteredAmis[]*ec2.Image

	for i := 0; i < len(images); i++ {
		now := time.Now()
		creationDate, err := time.Parse(time.RFC3339Nano, *images[i].CreationDate)
		if err != nil {
			return filteredAmis, err
		}

		duration := now.Sub(creationDate)

		if duration.Hours() > olderThanHours {
			filteredAmis = append(filteredAmis, images[i])
		}
	}

	return filteredAmis, nil
}

// Get a list of the existing AMIs that were created for the given EC2 instance
func findImages(instanceId string, svc *ec2.EC2) ([]*ec2.Image, error) {
	var noImages []*ec2.Image

	// Get a list of the existing AMIs that were created for the given EC2 instance
	resp, err := svc.DescribeImages(&ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("tag:ec2-snapper-instance-id"),
				Values: []*string{&instanceId},
			},
		},
	})
	if err != nil && strings.Contains(err.Error(), "NoCredentialProviders") {
		return noImages, errors.New("ERROR: No AWS credentials were found.  Either set the environment variables AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY, or run this program on an EC2 instance that has an IAM Role with the appropriate permissions.")
	} else if err != nil {
		return noImages, err
	}
	if len(resp.Images) == 0 {
		return noImages, errors.New("No AMIs were found for EC2 instance \"" + instanceId + "\"")
	}

	return resp.Images, nil
}

func deleteAmis(amis []*ec2.Image, snapshots []*ec2.Snapshot, numAmisToRemoveFromFiltered float64, svc *ec2.EC2, dryRun bool, ui cli.Ui) error {
	for i := 0; i < len(amis) - int(numAmisToRemoveFromFiltered); i++ {
		// Step 1: De-register the AMI
		ui.Output(*amis[i].ImageId + ": De-registering AMI named \"" + *amis[i].Name + "\"...")
		_, err := svc.DeregisterImage(&ec2.DeregisterImageInput{
			DryRun: &dryRun,
			ImageId: amis[i].ImageId,
		})
		if err != nil {
			if ! strings.Contains(err.Error(), "DryRunOperation") {
				return err
			}
		}

		// Step 2: Delete the corresponding AMI snapshot
		// Look at the "description" for each Snapshot to see if it contains our AMI id
		var snapshotIds []string
		for _, snapshot := range snapshots {
			if strings.Contains(*snapshot.Description, *amis[i].ImageId) {
				snapshotIds = append(snapshotIds, *snapshot.SnapshotId)
			}
		}

		// Delete all snapshots that were found
		ui.Output(*amis[i].ImageId + ": Found " + strconv.Itoa(len(snapshotIds)) + " snapshot(s) to delete")
		for _, snapshotId := range snapshotIds {
			ui.Output(*amis[i].ImageId + ": Deleting snapshot " + snapshotId + "...")
			_, deleteErr := svc.DeleteSnapshot(&ec2.DeleteSnapshotInput{
				DryRun: &dryRun,
				SnapshotId: &snapshotId,
			})

			if deleteErr != nil {
				return deleteErr
			}
		}

		ui.Output(*amis[i].ImageId + ": Done!")
		ui.Output("")
	}

	return nil
}

// TODO: convert this to use Go's time.ParseDuration
func parseOlderThanToHours(olderThan string) (float64, error) {
	var minutes float64
	var hours float64

	// Parse our date range
	match, _ := regexp.MatchString("^[0-9]*(h|d|m)$", olderThan)
	if ! match {
		return hours, errors.New("The --older-than value of \"" + olderThan + "\" is not formatted properly.  Use formats like 30d or 24h")
	}

	// We were given a time like "12h"
	if match, _ := regexp.MatchString("^[0-9]*(h)$", olderThan); match {
		hours, _ = strconv.ParseFloat(olderThan[0:len(olderThan)-1], 64)
	}

	// We were given a time like "15d"
	if match, _ := regexp.MatchString("^[0-9]*(d)$", olderThan); match {
		hours, _ = strconv.ParseFloat(olderThan[0:len(olderThan)-1], 64)
		hours *= 24
	}

	// We were given a time like "5m"
	if match, _ := regexp.MatchString("^[0-9]*(m)$", olderThan); match {
		minutes, _ = strconv.ParseFloat(olderThan[0:len(olderThan)-1], 64)
		hours = minutes/60
	}

	return hours, nil
}