package main

import (
	"github.com/alecthomas/kingpin"
	"github.com/b-b3rn4rd/cfn/pkg/command"
)

var (
	packageCommand            = kingpin.Command("package", "Packages the local artifacts (local paths) that your AWS CloudFormation template references.")
	packageTemplateFile       = packageCommand.Flag("template-file", "The path where your AWS CloudFormation template is located.").Required().ExistingFile()
	packageOutputTemplateFile = packageCommand.Flag("output-template-file", "The path to the file where the command writes the output AWS CloudFormation template.").String()

	packageS3Bucket    = packageCommand.Flag("s3-bucket", "The name of the S3 bucket where this command uploads your CloudFormation template.").Required().String()
	packageForceUpload = packageCommand.Flag("force-upload", "Indicates whether to override existing files in the S3 bucket.").Bool()
	packageUseJSON     = packageCommand.Flag("use-json", "Indicates whether to use JSON as the format for the output AWS CloudFormation template. YAML is used by default.").Bool()
	packageS3Prefix    = packageCommand.Flag("s3-prefix", "A prefix name that the command adds to the artifacts name when it uploads them to the S3 bucket.").String()
	packageKmsKeyID    = packageCommand.Flag("kms-key-id", "The ID of an AWS KMS key that the command uses to encrypt artifacts that are at rest in the S3 bucket.").String()
)

func (c *GoCfn) packaage(packageParams *command.PackageParams) {
	template, err := c.pckgr.Export(packageParams)

	var raw []byte

	if *packageUseJSON {
		raw, err = template.JSON()
	} else {
		raw, err = template.YAML()
	}

	if err != nil {
		c.logger.WithError(err).Error("error while exporting package")
		exiter(1)
		return
	}

	if *packageParams.OutputTemplateFile == "" {
		c.logger.Debug("output file is not specified, sending to stdout")
		strOutWriter.Write(string(raw))
		return
	}

	err = c.pckgr.WriteOutput(packageParams.OutputTemplateFile, raw)

	if err != nil {
		c.logger.WithError(err).Error("error while writing output")
		exiter(1)
		return
	}

	return
}
