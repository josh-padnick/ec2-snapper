package main

import (
	"fmt"
	//"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ec2"
)

func main() {
	fmt.Println("Hello Josh!")

	// Create an EC2 service object; AWS region is picked up from the "AWS_REGION" env var.
	svc := ec2.New(nil)

	// Create the AMI Snapshot
	// Todo: Capture these vals from command-line args
	name := "AMI-2"
	instanceId := "i-c724be30"
	dryRun := false
	noReboot := true

	resp, err := svc.CreateImage(&ec2.CreateImageInput{
		Name: &name,
		InstanceID: &instanceId,
		DryRun: &dryRun,
		NoReboot: &noReboot })

	if err != nil {
		panic(err)
	}

	// Todo: Print out nice message
	fmt.Println("ImageID = ",*resp.ImageID)
}
