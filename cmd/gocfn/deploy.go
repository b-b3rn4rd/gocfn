package main

import (
	"github.com/alecthomas/kingpin"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/b-b3rn4rd/gocfn/pkg/cfn"
	"github.com/b-b3rn4rd/gocfn/pkg/cli"
	"github.com/b-b3rn4rd/gocfn/pkg/deployer"
	"github.com/b-b3rn4rd/gocfn/pkg/uploader"
	"github.com/spf13/afero"
)

var (
	deployCommand              = kingpin.Command("deploy", "Deploys the specified AWS CloudFormation template by creating and then executing a change set.")
	deployTemplateFile         = deployCommand.Flag("template-file", "The path where your AWS CloudFormation template is located.").Required().ExistingFile()
	deployStackName            = deployCommand.Flag("name", "The name of the AWS CloudFormation stack you're deploying to.").Required().String()
	deployS3Bucket             = deployCommand.Flag("s3-bucket", "The name of the S3 bucket where this command uploads your CloudFormation template.").String()
	deployForceUpload          = deployCommand.Flag("force-upload", "Indicates whether to override existing files in the S3 bucket.").Bool()
	deployS3Prefix             = deployCommand.Flag("s3-prefix", "A prefix name that the command adds to the artifacts name when it uploads them to the S3 bucket.").String()
	deployKmsKeyID             = deployCommand.Flag("kms-key-id", "The ID of an AWS KMS key that the command uses to encrypt artifacts that are at rest in the S3 bucket.").String()
	deployParameterOverrides   = cli.CFNParameters(deployCommand.Flag("parameter-overrides", "A list of parameter structures that specify input parameters for your stack template."))
	deployCapabilities         = deployCommand.Flag("capabilities", "A list of capabilities that you must specify before AWS Cloudformation can create certain stacks.").Enums("CAPABILITY_IAM", "CAPABILITY_NAMED_IAM")
	deployNoExecuteChangeset   = deployCommand.Flag("no-execute-changeset", "Indicates whether to execute the change set. Specify this flag if you want to view your stack changes before executing").Bool()
	deployRoleArn              = deployCommand.Flag("role-arn", "The Amazon Resource Name (ARN) of an AWS Identity and Access Management (IAM) role").String()
	deployNotificationArns     = deployCommand.Flag("notification-arns", "The Amazon Resource Name (ARN) of an AWS Identity and Access Management (IAM) role.").Strings()
	deployFailOnEmptyChangeset = deployCommand.Flag("fail-on-empty-changeset", "Specify if the CLI should return a non-zero exit code if there are no changes to be made to the stack").Bool()
	deployTags                 = cli.CFNTags(deployCommand.Flag("tags", "A list of tags to associate with the stack that is created or updated."))
	deployForceDeploy          = deployCommand.Flag("force-deploy", "Force CloudFormation stack deployment if it's in CREATE_FAILED state.").Bool()
	deployStream               = deployCommand.Flag("stream", "Stream stack events during creation or update process.").Bool()
)

func deploy(sess client.ConfigProvider) {
	s3Svc := s3.New(sess)

	var s3Uploader uploader.Uploaderiface

	cfn := cfn.New(sess, logger, *deployStream)

	if *deployS3Bucket != "" {
		uSvc := s3manager.NewUploaderWithClient(s3Svc)
		s3Uploader = uploader.New(
			s3Svc,
			uSvc,
			logger,
			deployS3Bucket,
			deployS3Prefix,
			deployKmsKeyID,
			deployForceUpload,
			afero.NewOsFs(),
		)
	}

	body, err := cfn.Deploy(&deployer.DeployParams{
		S3Uploader:           s3Uploader,
		StackName:            aws.StringValue(deployStackName),
		TemplateFile:         aws.StringValue(deployTemplateFile),
		Parameters:           *deployParameterOverrides,
		Capabilities:         *deployCapabilities,
		NoExecuteChangeset:   aws.BoolValue(deployNoExecuteChangeset),
		RoleArn:              aws.StringValue(deployRoleArn),
		NotificationArns:     *deployNotificationArns,
		FailOnEmptyChangeset: aws.BoolValue(deployFailOnEmptyChangeset),
		Tags:                 *deployTags,
		ForceDeploy:          aws.BoolValue(deployForceDeploy),
	})
	if err != nil {
		logger.WithError(err).Error("error while running deploy command")
		exiter(1)
		return
	}

	switch body.(type) {
	case *cloudformation.Stack, *cloudformation.DescribeChangeSetOutput:
		jsonOutWriter.Write(body)
	}
}
