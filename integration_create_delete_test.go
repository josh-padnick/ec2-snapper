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
// to delete that snapshot.
func TestCreateAndDelete(t *testing.T) {
	t.Parallel()

	logger, ui := createLoggerAndUi("TestCreateAndDelete")
	session := session.New(&aws.Config{Region: aws.String(AWS_REGION_FOR_TESTING)})
	svc := ec2.New(session)

	instance, instanceName := launchInstance(svc, logger, t)
	defer terminateInstance(instance, svc, logger, t)
	waitForInstanceToStart(instance, svc, logger, t)

	snapshotId := takeSnapshotWithVerification(instanceName, *instance.InstanceId, ui, svc, logger, t)
	deleteSnapshotWithVerification(instanceName, snapshotId, ui, svc, logger, t)
}

// An integration test that runs an EC2 instance, uses create_command to take a snapshot of it, and then calls the
// delete_command to delete that snapshot, but setting the older than parmaeter in a way that should prevent any actual
// deletion.
func TestDeleteRespectsOlderThan(t *testing.T) {
	t.Parallel()

	logger, ui := createLoggerAndUi("TestDeleteRespectsOlderThan")
	session := session.New(&aws.Config{Region: aws.String(AWS_REGION_FOR_TESTING)})
	svc := ec2.New(session)

	instance, instanceName := launchInstance(svc, logger, t)
	defer terminateInstance(instance, svc, logger, t)
	waitForInstanceToStart(instance, svc, logger, t)

	snapshotId := takeSnapshotWithVerification(instanceName, *instance.InstanceId, ui, svc, logger, t)
	// Always try to delete the snapshot at the end so the tests don't litter the AWS account with snapshots
	defer deleteSnapshotWithVerification(instanceName, snapshotId, ui, svc, logger, t)

	// Set olderThan to "10h" to ensure the snapshot, which is only a few seconds old, does not get deleted
	deleteSnapshotForInstance(instanceName, "10h", 0, ui, logger, t)
	waitForSnapshotToBeDeleted(snapshotId, svc, logger, t)
	verifySnapshotWorks(snapshotId, svc, logger, t)
}

// An integration test that runs an EC2 instance, uses create_command to take a snapshot of it, and then calls the
// delete_command to delete that snapshot, but setting the at least parameter in a way that should prevent any actual
// deletion.
func TestDeleteRespectsAtLeast(t *testing.T) {
	t.Parallel()

	logger, ui := createLoggerAndUi("TestDeleteRespectsAtLeast")
	session := session.New(&aws.Config{Region: aws.String(AWS_REGION_FOR_TESTING)})
	svc := ec2.New(session)

	instance, instanceName := launchInstance(svc, logger, t)
	defer terminateInstance(instance, svc, logger, t)
	waitForInstanceToStart(instance, svc, logger, t)

	snapshotId := takeSnapshotWithVerification(instanceName, *instance.InstanceId, ui, svc, logger, t)
	// Always try to delete the snapshot at the end so the tests don't litter the AWS account with snapshots
	defer deleteSnapshotWithVerification(instanceName, snapshotId, ui, svc, logger, t)

	// Set atLeast to 1 to ensure the snapshot, which is the only one that exists, does not get deleted
	deleteSnapshotForInstance(instanceName, "0h", 1, ui, logger, t)
	waitForSnapshotToBeDeleted(snapshotId, svc, logger, t)
	verifySnapshotWorks(snapshotId, svc, logger, t)
}

func TestCreateWithInvalidInstanceName(t *testing.T) {
	t.Parallel()

	_, ui := createLoggerAndUi("TestCreateWithInvalidInstanceName")
	cmd := CreateCommand{
		Ui: ui,
		AwsRegion: AWS_REGION_FOR_TESTING,
		InstanceName: "not-a-valid-instance-name",
		AmiName: "this-ami-should-not-be-created",
	}

	_, err := create(cmd)

	if err == nil {
		t.Fatalf("Expected an error when creating a snapshot of an instance name that doesn't exist, but instead got nil")
	}
}

func TestCreateWithInvalidInstanceId(t *testing.T) {
	t.Parallel()

	_, ui := createLoggerAndUi("TestCreateWithInvalidInstanceId")
	cmd := CreateCommand{
		Ui: ui,
		AwsRegion: AWS_REGION_FOR_TESTING,
		InstanceId: "not-a-valid-instance-id",
		AmiName: "this-ami-should-not-be-created",
	}

	_, err := create(cmd)

	if err == nil {
		t.Fatalf("Expected an error when creating a snapshot of an instance id that doesn't exist, but instead got nil")
	}
}

func TestDeleteWithInvalidInstanceName(t *testing.T) {
	t.Parallel()

	_, ui := createLoggerAndUi("TestDeleteWithInvalidInstanceName")
	cmd := DeleteCommand{
		Ui: ui,
		AwsRegion: AWS_REGION_FOR_TESTING,
		InstanceName: "not-a-valid-instance-name",
		OlderThan: "0h",
		RequireAtLeast: 0,
	}

	err := deleteSnapshots(cmd)

	if err == nil {
		t.Fatalf("Expected an error when deleting a snapshot of an instance name that doesn't exist, but instead got nil")
	}
}

func TestDeleteWithInvalidInstanceId(t *testing.T) {
	t.Parallel()

	_, ui := createLoggerAndUi("TestDeleteWithInvalidInstanceId")
	cmd := DeleteCommand{
		Ui: ui,
		AwsRegion: AWS_REGION_FOR_TESTING,
		InstanceId: "not-a-valid-instance-id",
		OlderThan: "0h",
		RequireAtLeast: 0,
	}

	err := deleteSnapshots(cmd)

	if err == nil {
		t.Fatalf("Expected an error when deleting a snapshot of an instance id that doesn't exist, but instead got nil")
	}
}

func launchInstance(svc *ec2.EC2, logger *log.Logger, t *testing.T) (*ec2.Instance, string) {
	instanceName := fmt.Sprintf("ec2-snapper-unit-test-%s", UniqueId())
	userData := fmt.Sprint(USER_DATA_TEMPLATE, instanceName, TEST_FILE_PATH)

	logger.Printf("Launching EC2 instance in region %s. Its User Data will create a file %s with contents %s.", AWS_REGION_FOR_TESTING, TEST_FILE_PATH, instanceName)

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

	tagInstance(instance, instanceName, svc, logger, t)

	return instance, instanceName
}

func tagInstance(instance *ec2.Instance, instanceName string, svc *ec2.EC2, logger *log.Logger, t *testing.T) {
	logger.Printf("Adding tags to instance %s", *instance.InstanceId)

	_ , err := svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{instance.InstanceId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(instanceName),
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

func takeSnapshot(instanceName string, ui cli.Ui, logger *log.Logger, t *testing.T) string {
	log.Printf("Creating a snapshot with name %s.", instanceName)

	cmd := CreateCommand{
		Ui: ui,
		AwsRegion: AWS_REGION_FOR_TESTING,
		InstanceName: instanceName,
		AmiName: instanceName,
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

func waitForSnapshotToBeAvailable(instanceId string, svc *ec2.EC2, logger *log.Logger, t *testing.T) {
	logger.Printf("Waiting for snapshot for instance %s to become available", instanceId)

	instanceIdTagFilter := &ec2.Filter{
		Name: aws.String(fmt.Sprintf("tag:%s", EC2_SNAPPER_INSTANCE_ID_TAG)),
		Values: []*string{aws.String(instanceId)},
	}

	if err := svc.WaitUntilImageAvailable(&ec2.DescribeImagesInput{Filters: []*ec2.Filter{instanceIdTagFilter}}); err != nil {
		t.Fatal(err)
	}
}

func waitForSnapshotToBeDeleted(snapshotId string, svc *ec2.EC2, logger *log.Logger, t *testing.T) {
	logger.Printf("Waiting for snapshot %s to be deleted", snapshotId)

	// We just do a simple sleep, as there is no built-in API call to wait for this.
	time.Sleep(30 * time.Second)
}

func deleteSnapshotForInstance(instanceName string, olderThan string, requireAtLeast int, ui cli.Ui, logger *log.Logger, t *testing.T) {
	logger.Printf("Deleting snapshot for instance %s", instanceName)

	deleteCmd := DeleteCommand{
		Ui: ui,
		AwsRegion: AWS_REGION_FOR_TESTING,
		InstanceName: instanceName,
		OlderThan: olderThan,
		RequireAtLeast: requireAtLeast,
	}

	if err := deleteSnapshots(deleteCmd); err != nil {
		t.Fatal(err)
	}
}

func takeSnapshotWithVerification(instanceName string, instanceId string, ui cli.Ui, svc *ec2.EC2, logger *log.Logger, t *testing.T) string {
	snapshotId := takeSnapshot(instanceName, ui, logger, t)

	waitForSnapshotToBeAvailable(instanceId, svc, logger, t)
	verifySnapshotWorks(snapshotId, svc, logger, t)

	return snapshotId
}

func deleteSnapshotWithVerification(instanceName string, snapshotId string, ui cli.Ui, svc *ec2.EC2, logger *log.Logger, t *testing.T) {
	deleteSnapshotForInstance(instanceName, "0h", 0, ui, logger, t)
	waitForSnapshotToBeDeleted(snapshotId, svc, logger, t)
	verifySnapshotIsDeleted(snapshotId, svc, logger, t)
}

func createLoggerAndUi(testName string) (*log.Logger, cli.Ui) {
	logger := log.New(os.Stdout, testName + " ", log.LstdFlags)

	basicUi := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	prefixedUi := &cli.PrefixedUi{
		AskPrefix:		logger.Prefix(),
		AskSecretPrefix:	logger.Prefix(),
		OutputPrefix:		logger.Prefix(),
		InfoPrefix:		logger.Prefix(),
		ErrorPrefix:		logger.Prefix(),
		WarnPrefix:		logger.Prefix(),
		Ui:			basicUi,
		
	}
	
	return logger, prefixedUi
}