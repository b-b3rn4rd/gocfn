package main

import (
	"fmt"

	"github.com/alecthomas/kingpin"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/b-b3rn4rd/gocfn/pkg/cfn"
	"github.com/b-b3rn4rd/gocfn/pkg/packager"
	"github.com/b-b3rn4rd/gocfn/pkg/uploader"
	"github.com/spf13/afero"
)

var (
	packageCommand            = kingpin.Command("package", "Packages the local artifacts (local paths) that your AWS CloudFormation template references.")
	packageTemplateFile       = packageCommand.Flag("template-file", "The path where your AWS CloudFormation template is located.").Required().ExistingFile()
	packageOutputTemplateFile = packageCommand.Flag("output-template-file", "The path to the file where the command writes the output AWS CloudFormation template.").String()

	packageS3Bucket    = packageCommand.Flag("s3-bucket", "The name of the S3 bucket where this command uploads your CloudFormation template.").Required().String()
	packageForceUpload = packageCommand.Flag("force-upload", "Indicates whether to override existing files in the S3 bucket.").Bool()
	packageS3Prefix    = packageCommand.Flag("s3-prefix", "A prefix name that the command adds to the artifacts name when it uploads them to the S3 bucket.").String()
	packageKmsKeyID    = packageCommand.Flag("kms-key-id", "The ID of an AWS KMS key that the command uses to encrypt artifacts that are at rest in the S3 bucket.").String()
)

func packaage(sess client.ConfigProvider) {

	s3Svc := s3.New(sess)

	var s3Uploader uploader.Uploaderiface

	cfn := cfn.New(sess, logger, *deployStream)

	if *packageS3Bucket != "" {
		uSvc := s3manager.NewUploaderWithClient(s3Svc)
		s3Uploader = uploader.New(
			s3Svc, uSvc,
			logger,
			packageS3Bucket,
			packageS3Prefix,
			packageKmsKeyID,
			packageForceUpload,
			afero.NewOsFs(),
		)
	}

	body, err := cfn.Package(&packager.PackageParams{
		S3Uploader:         s3Uploader,
		TemplateFile:       *packageTemplateFile,
		OutputTemplateFile: *packageOutputTemplateFile,
	})
	if err != nil {
		logger.WithError(err).Error("error while running package command")
		exiter(1)
		return
	}

	if body == "" {
		strOutWriter.Write(fmt.Sprintf(`
Successfully packaged artifacts and wrote output template to file %s"
Execute the following command to deploy the packaged template"
"cfn deploy --template-file %s --name <YOUR STACK NAME>"`,
			*packageOutputTemplateFile, *packageOutputTemplateFile),
		)

		return
	}

	strOutWriter.Write(body)

}
