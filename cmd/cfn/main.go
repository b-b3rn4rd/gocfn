package main

import (
	"github.com/b-b3rn4rd/cfn/pkg/deployer"
	"github.com/b-b3rn4rd/cfn/pkg/uploader"
	"github.com/b-b3rn4rd/cfn/pkg/writer"
	"github.com/b-b3rn4rd/cfn/pkg/streamer"
	"github.com/b-b3rn4rd/cfn/pkg/cli"
	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"strings"
	"os"
	"github.com/spf13/afero"
)

var (
	version = "master"
	tracing = kingpin.Flag("trace", "Enable trace mode.").Short('t').Bool()
	debug   = kingpin.Flag("debug", "Enable debug logging.").Short('d').Bool()
	deployCommand = kingpin.Command("deploy", "Deploys the specified AWS CloudFormation template by creating and then executing a change set.")
	templateFile = deployCommand.Flag("template-file", "The path where your AWS CloudFormation template is located.").Required().ExistingFile()
	stackName = deployCommand.Flag("name", "The name of the AWS CloudFormation stack you're deploying to.").Required().String()
	s3Bucket = deployCommand.Flag("s3-bucket", "The name of the S3 bucket where this command uploads your CloudFormation template.").String()
	forceUpload = deployCommand.Flag("force-upload", "Indicates whether to override existing files in the S3 bucket.").Bool()
	s3Prefix = deployCommand.Flag("s3-prefix", "A prefix name that the command adds to the artifacts name when it uploads them to the S3 bucket.").String()
	kmsKeyId = deployCommand.Flag("kms-key-id", "The ID of an AWS KMS key that the command uses to encrypt artifacts that are at rest in the S3 bucket.").String()
	parameterOverrides = cli.CFNParameters(deployCommand.Flag("parameter-overrides", "A list of parameter structures that specify input parameters for your stack template."))
	capabilities = deployCommand.Flag("capabilities", "A list of capabilities that you must specify before AWS Cloudformation can create certain stacks.").Enums("CAPABILITY_IAM", "CAPABILITY_NAMED_IAM")
	noExecuteChangeset = deployCommand.Flag("no-execute-changeset", "Indicates whether to execute the change set. Specify this flag if you want to view your stack changes before executing").Bool()
	roleArn = deployCommand.Flag("role-arn", "The Amazon Resource Name (ARN) of an AWS Identity and Access Management (IAM) role").String()
	notificationArns = deployCommand.Flag("notification-arns", "The Amazon Resource Name (ARN) of an AWS Identity and Access Management (IAM) role.").Strings()
	failOnEmptyChangeset = deployCommand.Flag("fail-on-empty-changeset", "Specify if the CLI should return a non-zero exit code if there are no changes to be made to the stack").Bool()
	tags = cli.CFNTags(deployCommand.Flag("tags", "A list of tags to associate with the stack that is created or updated."))
	forceDeploy = deployCommand.Flag("force-deploy", "Force CloudFormation stack deployment if it's in CREATE_FAILED state.").Bool()
	stream = deployCommand.Flag("stream", "Stream stack events during creation or update processs.").Bool()
	logger = logrus.New()
	errWriter = writer.New(os.Stderr, writer.JsonFormatter)
	outWriter = writer.New(os.Stdout, writer.JsonFormatter)
)

func main()  {
	kingpin.Version(version)
	command := kingpin.Parse()

	if *debug {
		// set debug globally
		logrus.SetLevel(logrus.DebugLevel)
		// set debug in the logger we already created
		logger.SetLevel(logrus.DebugLevel)
	}

	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2")},
	)
	cfnSvc := (cloudformationiface.CloudFormationAPI)(cloudformation.New(sess))
	s3Svc := (s3iface.S3API)(s3.New(sess))

	var s3Uploader *uploader.Uploader

	if *s3Bucket != "" {
		uSvc := s3manager.NewUploaderWithClient(s3Svc)
		s3Uploader = uploader.New(s3Svc, uSvc, logger, s3Bucket, s3Prefix, kmsKeyId, forceUpload, afero.NewOsFs())
	}

	switch command {
		case "deploy":
			deploy(
				cfnSvc,
				s3Uploader,
				stackName,
				templateFile,
				([]*cloudformation.Parameter)(*parameterOverrides),
				capabilities,
				noExecuteChangeset,
				roleArn,
				notificationArns,
				failOnEmptyChangeset,
				([]*cloudformation.Tag)(*tags),
				forceDeploy,
			)
	}
}

func deploy(cfnSvc cloudformationiface.CloudFormationAPI, s3Uploader *uploader.Uploader, stackName *string, templateFile *string, parameters []*cloudformation.Parameter, capabilities *[]string, noExecuteChangeset *bool, roleArn *string, notificationArns *[]string, failOnEmptyChangeset *bool, tags []*cloudformation.Tag, forceDeploy *bool)  {
	runner := deployer.New(cfnSvc, logger)
	stmr := streamer.New(cfnSvc, logger)

	changeSet := runner.CreateChangeSet(stackName, templateFile, parameters, aws.StringSlice(*capabilities), noExecuteChangeset, roleArn, aws.StringSlice(*notificationArns), tags, forceDeploy, s3Uploader)

	if changeSet.Err != nil {
		logger.WithError(changeSet.Err).Fatal("ChangeSet creation error")
	}

	changeSetResult := runner.WaitForChangeSet(stackName, changeSet.ChangeSet.ChangeSetId)
	changeSet.ChangeSet = changeSetResult.ChangeSet
	changeSet.Err = changeSetResult.Err

	if changeSet.Err != nil {
		isEmptyChangeSet := strings.Contains(changeSet.Err.Error(), "The submitted information didn't contain changes.")

		if !*failOnEmptyChangeset && isEmptyChangeSet {
			outWriter.Write(runner.DescribeStackUnsafe(stackName))
			os.Exit(0)
		}

		logger.WithError(changeSet.Err).Fatal("ChangeSet creation error")
	}

	if *noExecuteChangeset {
		outWriter.Write(changeSet.ChangeSet)
		os.Exit(0)
	}

	if *stream {
		seenStackEvents := stmr.DescribeStackEvents(stackName, nil)
		if seenStackEvents.Err != nil {
			logger.WithError(seenStackEvents.Err).Fatal("Error while gathering stack events")
		}

		changeSet.StackEvents = seenStackEvents.Records
	}

	err := runner.ExecuteChangeset(stackName, changeSet.ChangeSet.ChangeSetId, changeSet.ChangeSetType)

	if err != nil {
		logger.WithError(err).Fatal("ChangeSet execution error")
	}

	res := runner.WaitForExecute(stackName, changeSet,  stream)

	if res.Err != nil {
		logger.WithError(res.Err).Fatal("ChangeSet execution error")
	} else {
		outWriter.Write(res.Stack)
	}
}
