package main

import (
	"strings"

	"github.com/alecthomas/kingpin"
	"github.com/b-b3rn4rd/cfn/pkg/cli"
	"github.com/b-b3rn4rd/cfn/pkg/command"
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

func (c *GoCfn) deploy(deployParams *command.DeployParams) {

	changeSet := c.dplr.CreateChangeSet(deployParams)

	if changeSet.Err != nil {
		c.logger.WithError(changeSet.Err).Error("ChangeSet creation error")
		exiter(1)
		return
	}

	changeSetResult := c.dplr.WaitForChangeSet(deployParams.StackName, changeSet.ChangeSet.ChangeSetId)
	changeSet.ChangeSet = changeSetResult.ChangeSet
	changeSet.Err = changeSetResult.Err

	if changeSet.Err != nil {
		isEmptyChangeSet := strings.Contains(changeSet.Err.Error(), "The submitted information didn't contain changes.")

		if !*deployParams.FailOnEmptyChangeset && isEmptyChangeSet {
			jsonOutWriter.Write(c.dplr.DescribeStackUnsafe(deployParams.StackName))
			return
		}

		c.logger.WithError(changeSet.Err).Error("ChangeSet creation error")
		exiter(1)
		return
	}

	if *deployParams.NoExecuteChangeset {
		jsonOutWriter.Write(changeSet.ChangeSet)
		return
	}

	if c.stmr != nil {
		seenStackEvents := c.stmr.DescribeStackEvents(deployParams.StackName, nil)
		if seenStackEvents.Err != nil {
			c.logger.WithError(seenStackEvents.Err).Error("Error while gathering stack events")
			exiter(1)
			return
		}

		changeSet.StackEvents = seenStackEvents.Records
	}

	err := c.dplr.ExecuteChangeset(deployParams.StackName, changeSet.ChangeSet.ChangeSetId)

	if err != nil {
		c.logger.WithError(err).Error("ChangeSet execution error")
		exiter(1)
		return
	}

	res := c.dplr.WaitForExecute(deployParams.StackName, changeSet, c.stmr)

	if res.Err != nil {
		c.logger.WithError(res.Err).Error("ChangeSet execution error")
		exiter(1)
		return
	}

	jsonOutWriter.Write(res.Stack)
}
