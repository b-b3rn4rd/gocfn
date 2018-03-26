package main

import (
	"github.com/alecthomas/kingpin"
	"github.com/b-b3rn4rd/cfn/pkg/command"
)

var (
	packageCommand      = kingpin.Command("package", "Packages the local artifacts (local paths) that your AWS CloudFormation template references.")
	packageTemplateFile = packageCommand.Flag("template-file", "The path where your AWS CloudFormation template is located.").Required().ExistingFile()

	packageS3Bucket    = packageCommand.Flag("s3-bucket", "The name of the S3 bucket where this command uploads your CloudFormation template.").String()
	packageForceUpload = packageCommand.Flag("force-upload", "Indicates whether to override existing files in the S3 bucket.").Bool()
	packageS3Prefix    = packageCommand.Flag("s3-prefix", "A prefix name that the command adds to the artifacts name when it uploads them to the S3 bucket.").String()
	packageKmsKeyID    = packageCommand.Flag("kms-key-id", "The ID of an AWS KMS key that the command uses to encrypt artifacts that are at rest in the S3 bucket.").String()
)

func (c *Cfn) packaage(deployParams *command.PackageParams) {

}
