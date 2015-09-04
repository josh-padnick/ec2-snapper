	package main

	import (
		"flag"
		"time"
		"strings"

		"github.com/aws/aws-sdk-go/service/ec2"
		"github.com/aws/aws-sdk-go/aws"
		"github.com/mitchellh/cli"
	)

	type CreateCommand struct {
		Ui 			cli.Ui
		InstanceId 	string
		Name 		string
		DryRun		bool
		NoReboot	bool
	}

	// descriptions for args
	var createDscrInstanceId = "The instance from which to create the AMI"
	var createDscrName = "The name of the AMI; the current timestamp will be automatically appended"
	var createDscrDryRun = "Execute a simulated run"
	var createDscrNoReboot = "If true, do not reboot the instance before creating the AMI. It is preferable to reboot the instance to guarantee a consistent filesystem when taking the snapshot, but the likelihood of an inconsistent snapshot is very low."

	func (c *CreateCommand) Help() string {
		return `ec2-snapper create <args> [--help]

	Create an AMI of the given EC2 instance.

	Available args are:
	--instance      ` + createDscrInstanceId + `
	--name          ` + createDscrName + `
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

		cmdFlags.StringVar(&c.InstanceId, "instance", "", createDscrInstanceId)
		cmdFlags.StringVar(&c.Name, "name", "", createDscrName)
		cmdFlags.BoolVar(&c.DryRun, "dry-run", false, createDscrDryRun)
		cmdFlags.BoolVar(&c.NoReboot, "no-reboot", true, createDscrNoReboot)

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
		const dateLayoutForAmiName = "2006-01-02 at 15_04_05 (MST)"
		t := time.Now()

		// Create the AMI Snapshot
		name := c.Name + " - " + t.Format(dateLayoutForAmiName)

		c.Ui.Output("==> Creating AMI for " + c.InstanceId + "...")

		resp, err := svc.CreateImage(&ec2.CreateImageInput{
			Name: &name,
			InstanceID: &c.InstanceId,
			DryRun: &c.DryRun,
			NoReboot: &c.NoReboot })
		if err != nil && strings.Contains(err.Error(), "NoCredentialProviders") {
			c.Ui.Error("ERROR: No AWS credentials were found.  Either set the environment variables AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY, or run this program on an EC2 instance that has an IAM Role with the appropriate permissions.")
			return 1
		} else if err != nil {
			panic(err)
		}

		// Sleep here to give time for AMI to get found
		time.Sleep(3000 * time.Millisecond)

		// Assign tags to this AMI.  We'll use these when it comes time to delete the AMI
		c.Ui.Output("==> Adding tags to AMI " + *resp.ImageID + "...")

		svc.CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{resp.ImageID},
			Tags: []*ec2.Tag{
				&ec2.Tag{ Key: aws.String("ec2-snapper-instance-id"), Value: &c.InstanceId },
				&*ec2.Tag{
				&ec2.Tag{ Key: aws.String("Name"), Value: &c.Name },
			},
		})

		// Check the status of the AMI
		respDscrImages, err := svc.DescribeImages(&ec2.DescribeImagesInput{
			Filters: []*ec2.Filter{
				&ec2.Filter{
					Name: aws.String("image-id"),
					Values: []*string{resp.ImageID},
				},
			},
		})
		if err != nil {
			panic(err)
		}

		// If no AMI at all was found, throw an error
		if len(respDscrImages.Images) == 0 {
			c.Ui.Error("ERROR: Could not find the AMI just created.")
			return 1
		}

		// If the AMI's status is failed throw an error
		if *respDscrImages.Images[0].State == "failed" {
			c.Ui.Error("ERROR: AMI was crexated but entered a state of 'failed'. This is an AWS issue. Please re-run this command.  Note that you will need to manually de-register the AMI in the AWS console or via the API.")
			return 1
		}

		// Announce success
		c.Ui.Info("==> Success! Created " + *resp.ImageID + " named \"" + name + "\"")
		return 0
	}

