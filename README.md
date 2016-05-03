# ec2-snapper

ec2-snapper is a simple command-line tool for creating and deleting AMI's of your EC2 instances.  It was designed to make it easy to make backups of your AMI's and to cleanup old backups by deleting all AMI's (and their corresponding Snapshots) for a given EC2 instance which are older than X days/hours/minutes. It works especially well as part of a cronjob. It can also report custom metrics to CloudWatch, which can be useful for triggering alarms if a cronjob fails to run.

## Download
Download the latest version from the [releases page](https://github.com/josh-padnick/ec2-snapper/releases).

## Motivation
For the full story, see the [Motivating Blog Post](https://joshpadnick.com/2015/06/18/a-simple-tool-for-snapshotting-your-ec2-instances/).

One of the best parts of working with EC2 instances is you can create a snapshot of the EC2 instance as an Amazon Machine Image (AMI).  The problem is that deleting AMI's is a really clunky experience:

1. Deleting an AMI is a two-part process.  First, you have to de-register the AMI.  Then you have to delete the corresponding EBS volume snapshot.

2. Finding the corresponding snapshot is cumbersome.

3. There's no out-of-the-box way to delete all AMI's older than X days.

I wrote ec2-snapper so I could use a simple command-line tool to create snapshots, delete them with one command, and delete ones older than a certain age.  It works especially well when run as a cronjob on a nightly basis. It even supports sending custom metrics to CloudWatch, which you can use to trigger alarms in case a cronjob fails.

I personally use it to backup my Wordpress blog which is running as a single EC2 instance.  If my EC2 instance were to fail, I can instantly launch a new EC2 instance from the latest snapshot.  Since I run ec2-snapper nightly, I'm subject to up to 24 hours of data loss, which is tolerable for my needs.

## Prerequisites
You will need to setup your AWS credentials so ec2-snapper can authenticate to AWS.

### Option 1: Set Environment Variables
One option is to authenticate by exporting the following environment variables:

```bash
AWS_ACCESS_KEY_ID=AKID1234567890
AWS_SECRET_ACCESS_KEY=MY-SECRET-KEY
```

### Option 2: Use IAM Roles
If you're running ec2-snapper on an Amazon EC2 instance, the preferred way to authenticate is by assigning an [IAM Role](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html) to your EC2 instance.  Note that IAM roles can only be assigned when an EC2 instance is being launched, and not after the fact.

### Account Permissions
Whichever method you use to authenticate, the AWS account you use to authenticate will need the limited set of IAM permissions in this IAM policy:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "Stmt1433747550000",
            "Effect": "Allow",
            "Action": [
                "ec2:CreateImage",
                "ec2:CreateTags",
                "ec2:DeleteSnapshot",
                "ec2:DeregisterImage",
                "ec2:DescribeImages",
                "ec2:DescribeSnapshots"
            ],
            "Resource": [
                "*"
            ]
        }
    ]
}
```

## Installation
There's nothing to install.  Just download the binary and run it using the commands you see below.

## Usage
Try any of the following commands to get a full list of all arguments:

```bash
ec2-snapper --help
ec2-snapper create --help
ec2-snapper delete --help
ec2-snapper report --help
```

### Get the Version
```bash
ec2-snapper version
```

Returns the current version you're using of ec2-snapper.

### Create an AMI
For all options, run `ec2-snapper create --help`.

Example:

```bash
ec2-snapper create --region=us-west-2 --instance-id=i-c724be30 --ami-name=MyEc2Instance --dry-run --no-reboot
```
You must specify the AWS region (e.g. `--region=us-west-2`) and either the ID (e.g. `--instance-id=i-c724be30`) or the name as set in an EC2 tag called "Name" (e.g. `--instance-name=my-instance`) of an EC2 instance in that region to be snapshotted. You also specify what to name the AMI, such as "MyWebsite.com", using the `--ami-name` parameter.  A current timestamp will automatically be appended to the AMI name.

For example, `ec2-snapper create --instance-id=i-c724be30 --ami-name="MyWebsite.com"` resulted in an AMI named "MyWebsite.com - 2015-06-08 at 08_26_51 (UTC)".

Adding `--dry-run` will simulate the command without actually taking a snapshot.

`--no-reboot` explicitly indicates whether to reboot the EC2 instance when taking the snapshot.  The default is `true`.

Note that the last two args can either be written as `--dry-run` or `--dry-run=true`.  

### Delete AMIs older than X days / Y hours / Z minutes
For all options, run `ec2-snapper delete --help`.

Example:

```bash
ec2-snapper delete --region=us-west-2 --instance-id=i-c724b30 --older-than=30d --dry-run
```

You must specify the AWS region (e.g. `--region=us-west-2`) and either the ID (e.g. `--instance-id=i-c724be30`) or the name as set in an EC2 tag called "Name" (e.g. `--instance-name=my-instance`) of an EC2 instance in that region that was originally used to create the AMIs you wish to delete (even if that EC2 instance has since been stopped or terminated).

`--older-than` accepts time values like `30d`, `5h` or `15m` for 30 days, 5 hours, or 15 minutes, respectively.  For example, `--older-than=30d` tells ec2-snapper to delete any AMI for the given EC2 instance that is older than 30 days.

`--require-at-least` ensures that in no event will there be fewer than the specified number of total AMIs for this instance.  For example, `--require-at-least=5` tells ec2-snapper to always make sure there are at least 5 total AMIs for the given instance, even if these AMIs are marked for deletion based on the `--older-than` command.

`--dry-run` will list the AMIs that would have been deleted, but does not actually delete them.

### Report to CloudWatch
For all options, run `ec2-snapper report --help`.

Example:

```bash
ec2-snapper report --region=us-west-2 --name=MyEc2Backup --namespace=MyCustomMetrics --value=1
```

This command will write a custom metric to the specified region (e.g. `--region=us-west-2`) with the specified name (e.g. `--metric-name=MyEc2Backup`), namespace (e.g. `--namespace=MyCustomMetrics`), and value (e.g. `--value=1`). You can then add monitoring and alerting around this metric.

For example, let's say you use a cronjob to run ec2-snapper once per night, and if the job completes successfully, you fire the metric as shown in the example above. In that case, you could create a CloudWatch alarm that goes off if the value of the `MyEc2Backup` metric is less than 1 over a 24 hour period. You can configure the alarm to send you an email or text message whenever it goes into `INSUFFICIENT_DATA` state, which would be an indicator that the cronjob failed for some reason.

## Contributors
This was my first golang program, so I'm sure the code can benefit from various optimizations.  Pull requests and bug reports are always welcome.

### Running from source
The easiest way to run ec2-snapper from source is with the following command:

```bash
go run main.go *_command.go
```

This is necessary because all the code is in the `main` package, so you have to tell Go explicitly what to build and
run. For example, to run the `create` command, you could do:

```bash
go run main.go *_command.go create --region=us-west-2 --instance-id=i-c1234567 --ami-name=MyBackup
```

### Tests
This repo contains two types of tests:

1. Unit tests: fast, isolated tests of individual functions. They use the name format `unit_xxx_test.go`.
2. Integration tests: slower, end-to-end tests that create and delete real resources in an AWS account. **All the
   resources should fit into the AWS free tier, but if you've used up all your credits, you may be charged!**
   Integration tests use the name format `integration_xxx_test.go`.

To run the tests, first, set your AWS credentials using the environment variables `AWS_ACCESS_KEY_ID` and
`AWS_SECRET_ACCESS_KEY`.

To run all the tests:

```bash
./_ci/run-tests.sh
```

To run a specific test:

```bash
go test -run MY_TEST_NAME
```

### Release process

1. Update the version number in `main.go`.
1. Rebuild binaries by running `cross-compile.sh`.
1. Update `CHANGELOG.md`.
1. Commit all changes.
1. Create a new release using the [GitHub Release Page](https://github.com/josh-padnick/ec2-snapper/releases). Make
   sure to use the same version number as in step #1 and the changelog from step #3.

TODO: automate this process!