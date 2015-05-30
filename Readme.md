# ec2-Snapper: A Simple Tool for Creating and Deleting EC2 Snapshots

## Background
...

## ToDo
- Dynamically detect region of current EC2 instance

## Prerequisites

### Setup Your AWS Credentials
This tool uses the [AWS SDK for Go](https://github.com/awslabs/aws-sdk-go), which has at least two different ways of configuring your AWS credentials.  I recommend setting the following environment variables:

```
AWS_ACCESS_KEY_ID=AKID1234567890
AWS_SECRET_ACCESS_KEY=MY-SECRET-KEY
```

### Indicate your AWS Region
Tell `ec2-snapper` which AWS region to operate on by exporting this environment variable:

```
AWS_REGION=us-west-2
```

Common USA regions are:

- North Virginia = `us-east-1`
- North California = `us-west-1`
- Oregon = `us-west-2` 