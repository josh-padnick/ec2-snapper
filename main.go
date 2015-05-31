package main

import (
	"fmt"
	"time"
	//"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ec2"
)

func main() {

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

	fmt.Printf("==> Creating AMI for %s...\n", instanceId)

	resp, err := svc.CreateImage(&ec2.CreateImageInput{
		Name: &name,
		InstanceID: &instanceId,
		DryRun: &dryRun,
		NoReboot: &noReboot })
	if err != nil {
		panic(err)
	}

	fmt.Printf("==> Created %s named \"%s\"\n",*resp.ImageID,name)
}
