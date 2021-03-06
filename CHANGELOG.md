# 0.5.2 (September 14, 2016)

* BUG: The ec2-snapper `delete` command now gracefully exits if no snapshots exist for the given instance rather than
  exiting with an error.

# 0.5.1 (September 11, 2016)

* ENHANCEMENT: ec2-snapper now also tags the EBS volume snapshots it creates as part of the process of creating an AMI.
  This allows you to find an AMI by name and restore an entire EC2 Instance, or find individual EBS Volumes by name and
  restore just a single volume.

# 0.5.0 (April 15, 2016)

* ENHANCEMENT: Add support for a `report` command for reporting CloudWatch metrics.

# 0.4.2 (April 15, 2016)

* BUG: Fix version number.

# 0.4.1 (April 14, 2016)

* TWEAK: Use gox for cross-compile.
* BUG: Fix extra newline issue in binaries.
* BUG: Fix issues in `circle.yml`.

# 0.4.0 (April 14, 2016)

* REFACTOR: The AWS region is passed in via the `--region` param.
* ENHANCEMENT: You can now specify an instance name via the `--instance-name` parameter instead of using `--instance-id`
  (which might change every time you redeploy).
* ENHANCEMENT: Added unit and integration tests.
* TWEAK: Publish binaries directly in GitHub instead of bintray.

# 0.3.0 (February 11, 2016)

* ENHANCEMENT: Created AMIs now include the specified name in the `Name` tag. [GH-4](https://github.com/josh-padnick/ec2-snapper/pull/4)
* ENHANCEMENT: Added `ec2-snapper version` subcommand.
* BUG: Fixed [ec2-snapper only deletes 1 snapshot per AMI](https://github.com/josh-padnick/ec2-snapper/issues/5)
* TWEAK: Updated to latest version of AWS SDK for Golang

# 0.2.0 (July 17, 2015)

* FEATURE: Added the ability to say "always leave at least X AMI's in place"
* BUG: Fixed [AWS API sometimes fails to add tags as requested](https://github.com/josh-padnick/ec2-snapper/issues/1)

# 0.1.0 (June 8, 2015)

* Initial release
