# 0.3.0 (July 17, 2015)

* ENHANCEMENT: Created AMIs now include the specified name in the `Name` tag. [GH-4](https://github.com/josh-padnick/ec2-snapper/pull/4)
* ENHANCEMENT: Added `ec2-snapper version` subcommand.
* BUG: Fixed [ec2-snapper only deletes 1 snapshot per AMI](https://github.com/josh-padnick/ec2-snapper/issues/5)
* TWEAK: Updated to latest version of AWS SDK for Golang

# 0.2.0 (July 17, 2015)

* FEATURE: Added the ability to say "always leave at least X AMI's in place"
* BUG: Fixed [AWS API sometimes fails to add tags as requested](https://github.com/josh-padnick/ec2-snapper/issues/1)

# 0.1.0 (June 8, 2015)

* Initial release
