# vpc-nag

This is a basic script to quickly audit your account VPCs.

    $ go install github.com/guardian/vpc-nag
    $ vpc-nag --accountId [AWS_ACCOUNT_ID]

`vpc-nag` uses Prism behind the scenes so you will need to be on the VPC/in the
office to run it.

If you need to install Go, run:

    $ brew install go

Once audited, see
[here](https://docs.google.com/document/d/1gaUQJrR4K2jTp6t7ZHEQV_2Vp1b_FsfBl426j0roQi4/edit#)
how to fix problems.