package main

import (
	"testing"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/mitchellh/cli"
	"os"
	"fmt"
	"log"
	"encoding/base64"
	"time"
)

const AWS_REGION_FOR_TESTING = "us-east-1"
const AMAZON_LINUX_AMI_ID = "ami-08111162"

const TEST_FILE_PATH = "/home/ec2-user/test-file"
const USER_DATA_TEMPLATE =
`#!/bin/bash
set -e
echo '%s' > "%s"
`

// An integration test that runs an EC2 instance, uses create_command to take a snapshot of it, and then delete_command
// to delete that snapshot. Since testing these functions requires a lot of setup in an AWS account, it's faster and
// easier to do a single integration test rather than a number of smaller unit tests that each do a lot of setup and
// teardown.
func TestCreateAndDelete(t *testing.T) {
	t.Parallel()

	logger := log.New(os.Stdout, "TestCreateAndDelete ", log.LstdFlags)

	session := session.New(&aws.Config{Region: aws.String(AWS_REGION_FOR_TESTING)})
	svc := ec2.New(session)

	instance, uniqueId := launchInstance(svc, logger, t)
	defer terminateInstance(instance, svc, logger, t)
	waitForInstanceToStart(instance, svc, logger, t)

	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	snapshotId := takeSnapshot(instance, uniqueId, svc, ui, logger, t)
	waitForSnapshotToBeAvailable(snapshotId, svc, logger, t)
	verifySnapshotWorks(snapshotId, svc, logger, t)

	deleteSnapshotForInstance(instance, ui, logger, t)
	waitForSnapshotToBeDeleted(snapshotId, svc, logger, t)
	verifySnapshotIsDeleted(snapshotId, svc, logger, t)
}

func launchInstance(svc *ec2.EC2, logger *log.Logger, t *testing.T) (*ec2.Instance, string) {
	uniqueId := UniqueId()
	userData := fmt.Sprint(USER_DATA_TEMPLATE, uniqueId, TEST_FILE_PATH)

	logger.Printf("Launching EC2 instance in region %s. Its User Data will create a file %s with contents %s.", AWS_REGION_FOR_TESTING, TEST_FILE_PATH, uniqueId)

	runResult, err := svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:      aws.String(AMAZON_LINUX_AMI_ID),
		InstanceType: aws.String("t2.micro"),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		UserData:     aws.String(base64.StdEncoding.EncodeToString([]byte(userData))),
	})

	if err != nil {
		t.Fatal(err)
	}

	if len(runResult.Instances) != 1 {
		t.Fatalf("Expected to launch 1 instance but got %d", len(runResult.Instances))
	}

	instance := runResult.Instances[0]
	logger.Printf("Launched instance %s", *instance.InstanceId)

	tagInstance(instance, uniqueId, svc, logger, t)

	return instance, uniqueId
}

func tagInstance(instance *ec2.Instance, uniqueId string, svc *ec2.EC2, logger *log.Logger, t *testing.T) {
	logger.Printf("Adding tags to instance %s", *instance.InstanceId)

	_ , err := svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{instance.InstanceId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(fmt.Sprintf("ec2-snapper-unit-test-%s", uniqueId)),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func waitForInstanceToStart(instance *ec2.Instance, svc *ec2.EC2, logger *log.Logger, t *testing.T) {
	logger.Printf("Waiting for instance %s to start...", *instance.InstanceId)
	if err := svc.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{InstanceIds: []*string{instance.InstanceId}}); err != nil {
		t.Fatal(err)
	}
	logger.Printf("Instance %s is now running", *instance.InstanceId)
}

func terminateInstance(instance *ec2.Instance, svc *ec2.EC2, logger *log.Logger, t *testing.T) {
	logger.Printf("Terminating instance %s", *instance.InstanceId)
	if _, err := svc.TerminateInstances(&ec2.TerminateInstancesInput{InstanceIds: []*string{instance.InstanceId}}); err != nil {
		t.Fatal("Failed to terminate instance %s", *instance.InstanceId)
	}
}

func takeSnapshot(instance *ec2.Instance, uniqueId string, svc *ec2.EC2, ui cli.Ui, logger *log.Logger, t *testing.T) string {
	backupName := fmt.Sprintf("ec2-snapper-unit-test-create-%s", uniqueId)
	log.Printf("Creating a snapshot with name %s.", backupName)


	cmd := CreateCommand{
		Ui: ui,
		AwsRegion: AWS_REGION_FOR_TESTING,
		InstanceId: *instance.InstanceId,
		Name: backupName,
	}

	snapshotId, err := create(cmd)

	if err != nil {
		t.Fatal(err)
	}

	logger.Printf("Created snasphot %s", snapshotId)
	return snapshotId
}

func verifySnapshotWorks(snapshotId string, svc *ec2.EC2, logger *log.Logger, t *testing.T) {
	logger.Printf("Verifying snapshot %s exists", snapshotId)

	snapshots := findSnapshots(snapshotId, svc, logger, t)
	if len(snapshots) != 1 {
		t.Fatalf("Expected to find one snapshot with id %s but found %d", snapshotId, len(snapshots))
	}

	snapshot := snapshots[0]

	if *snapshot.State == ec2.ImageStateAvailable {
		logger.Printf("Found snapshot %s in expected state %s", snapshotId, *snapshot.State)
	} else {
		t.Fatalf("Expected image to be in state %s, but it was in state %s", ec2.ImageStateAvailable, *snapshot.State)
	}

	// TODO: fire up a new EC2 instance with the snapshot, SSH to it, and check the file we wrote is still there
}

func verifySnapshotIsDeleted(snapshotId string, svc *ec2.EC2, logger *log.Logger, t *testing.T) {
	logger.Printf("Verifying snapshot %s is deleted", snapshotId)
	snapshots := findSnapshots(snapshotId, svc, logger, t)
	if len(snapshots) != 0 {
		t.Fatalf("Expected to find zero snapshots with id %s but found %d", snapshotId, len(snapshots))
	}
}

func findSnapshots(snapshotId string, svc *ec2.EC2, logger *log.Logger, t *testing.T) []*ec2.Image {
	resp, err := svc.DescribeImages(&ec2.DescribeImagesInput{ImageIds: []*string{&snapshotId}})
	if err != nil {
		t.Fatal(err)
	}

	return resp.Images
}

func waitForSnapshotToBeAvailable(snapshotId string, svc *ec2.EC2, logger *log.Logger, t *testing.T) {
	logger.Printf("Waiting for snapshot %s to become available", snapshotId)

	if err := svc.WaitUntilImageAvailable(&ec2.DescribeImagesInput{ImageIds: []*string{&snapshotId}}); err != nil {
		t.Fatal(err)
	}
}

func waitForSnapshotToBeDeleted(snapshotId string, svc *ec2.EC2, logger *log.Logger, t *testing.T) {
	logger.Printf("Waiting for snapshot %s to be deleted", snapshotId)

	// We just do a simple sleep, as there is no built-in API call to wait for this.
	time.Sleep(30 * time.Second)
}

func deleteSnapshotForInstance(instance *ec2.Instance, ui cli.Ui, logger *log.Logger, t *testing.T) {
	logger.Printf("Deleting snapshot for instance %s", *instance.InstanceId)

	deleteCmd := DeleteCommand{
		Ui: ui,
		AwsRegion: AWS_REGION_FOR_TESTING,
		InstanceId: *instance.InstanceId,
		OlderThan: "0h",
		RequireAtLeast: 0,
	}

	if err := deleteSnapshots(deleteCmd); err != nil {
		t.Fatal(err)
	}
}