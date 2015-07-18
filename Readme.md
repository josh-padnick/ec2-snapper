# ec2-snapper

ec2-snapper is a simple command-line tool for creating and deleting AMI's of your EC2 instances.  It was designed to make it easy to delete all AMI's for a given EC2 instance which are older than X days/hours/minutes.  It works especially well as part of a cronjob.

## Download
Download the latest version using the links below:

- [ec2-snapper for Linux 64-bit](https://s3-us-west-2.amazonaws.com/phxdevops-tmp/ec2-snapper/0.2.0/linux_amd64/ec2-snapper)
- [ec2-snapper for Linux 32-bit](https://s3-us-west-2.amazonaws.com/phxdevops-tmp/ec2-snapper/0.2.0/linux_386/ec2-snapper)
- [ec2-snapper for Mac/OS X 64-bit](https://s3-us-west-2.amazonaws.com/phxdevops-tmp/ec2-snapper/0.2.0/darwin_amd64/ec2-snapper)
- [ec2-snapper for Mac/OS X 32-bit](https://s3-us-west-2.amazonaws.com/phxdevops-tmp/ec2-snapper/0.2.0/darwin_386/ec2-snapper)
- [ec2-snapper for Windows 64-bit](https://s3-us-west-2.amazonaws.com/phxdevops-tmp/ec2-snapper/0.2.0/windows_amd64/ec2-snapper.exe)
- [ec2-snapper for Windows 32-bit](https://s3-us-west-2.amazonaws.com/phxdevops-tmp/ec2-snapper/0.2.0/windows_386/ec2-snapper.exe)

## Motivation
For the full story, see the [Motivating Blog Post](https://joshpadnick.com/2015/06/18/a-simple-tool-for-snapshotting-your-ec2-instances/).

One of the best parts of working with EC2 instances is you can create a snapshot of the EC2 instance as an Amazon Machine Image (AMI).  The problem is that deleting AMI's is a really clunky experience:

1. Deleting an AMI is a two-part process.  First, you have to de-register the AMI.  Then you have to delete the corresponding EBS volume snapshot.

2. Finding the corresponding snapshot is cumbersome.

3. There's no out-of-the-box way to delete all AMI's older than X days.

I wrote ec2-snapper so I could use a simple command-line tool to create snapshots, delete them with one command, and delete ones older than a certain age.  It works especially well when run as a cronjob on a nightly basis.

I personally use it to backup my Wordpress blog which is running as a single EC2 instance.  If my EC2 instance were to fail, I can instantly launch a new EC2 instance from the latest snapshot.  Since I run ec2-snapper nightly, I'm subject to up to 24 hours of data loss, which is tolerable for my needs.

## Prerequisites

### 1. Indicate your AWS Region
ec2-snapper needs to know which AWS region to operate on. You can tell it by exporting this environment variable:

```
AWS_REGION=us-west-2
```
Common USA regions are:

- North Virginia = `us-east-1`
- North California = `us-west-1`
- Oregon = `us-west-2`

_*Tip: To export an environment variable on most linux distro's, edit the file `/etc/environment`.  You may need to reboot before this takes effect, or you can try the command `. /etc/environment` to apply the changes immediately.*_

### 2. Setup Your AWS Credentials
You will also need to authenticate to AWS.

#### Option 1: Set Environment Variables
One option is to authenticate by exporting the following environment variables:

```
AWS_ACCESS_KEY_ID=AKID1234567890
AWS_SECRET_ACCESS_KEY=MY-SECRET-KEY
```

#### Option 2: Use IAM Roles
If you're running ec2-snapper on an Amazon EC2 instance, the preferred way to authenticate is by assigning an [IAM Role](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html) to your EC2 instance.  Note that IAM roles can only be assigned when an EC2 instance is being launched, and not after the fact.

#### Account Permissions
Whichever method you use to authenticate, the AWS account you use to authenticate will need the limited set of IAM permissions in this IAM policy:

```
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


## Usage
Try any of the following commands to get a full list of all arguments:

```
./ec2-snapper --help
./ec2-snapper create --help
./ec2-snapper delete --help
```

### Create an AMI
```
ec2-snapper create --instance=i-c724be30 --name=MyEc2Instance --dry-run --no-reboot
```
You must specify the EC2 instance ID (e.g. `i-c724be30`) to be snapshotted, and give it a name such as "MyWebsite.com".  A current timestamp will automatically be appended to the name.  

For example, `./ec2-snapper create --instance i-c724be30 --name "MyWebsite.com"` resulted in an AMI named "MyWebsite.com - 2015-06-08 at 08_26_51 (UTC)".

Adding `--dry-run` will simulate the command without actually taking a snapshot.

`--no-reboot` explicitly indicates whether to reboot the EC2 instance when taking the snapshot.  The default is `true`.

Note that the last two args can either be written as `--dry-run` or `--dry-run=true`.  

### Delete AMIs older than X days / Y hours / Z minutes
```
ec2-snapper delete --instance=i-c724b30 --older-than=30d --dry-run
```
You must specify the EC2 instance ID (e.g. `i-c724be30`) originally used to create the AMIs you wish to delete, even if that EC2 instance has since been stopped or terminated.  

`--older-than` accepts time values like `30d`, `5h` or `15m` for 30 days, 5 hours, or 15 minutes, respectively.  For example, `--older-than=30d` tells ec2-snapper to delete any AMI for the given EC2 instance that is older than 30 days.

`--require-at-least` ensures that in no event will there be fewer than the specified number of total AMIs for this instance.  For example, `--require-at-least=5` tells ec2-snapper to always make sure there are at least 5 total AMIs for the given instance, even if these AMIs are marked for deletion based on the `--older-than` command.

`--dry-run` will list the AMIs that would have been deleted, but does not actually delete them.

## Contributors
This was my first golang program, so I'm sure the code can benefit from various optimizations.  Pull requests and bug reports are always welcome.
