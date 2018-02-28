package main

import (
	"github.com/b-b3rn4rd/cfn/pkg/stack"
	"github.com/b-b3rn4rd/cfn/pkg/cli"
	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	//"github.com/aws/aws-sdk-go/aws/session"
	//"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	//"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"fmt"
)

var (
	version = "master"
	tracing = kingpin.Flag("trace", "Enable trace mode.").Short('t').Bool()
	debug   = kingpin.Flag("debug", "Enable debug logging.").Short('d').Bool()
	//wait    = kingpin.Flag("wait", "Wait for stack completion.").Bool()
	deployCommand = kingpin.Command("deploy", "Deploys the specified AWS CloudFormation template by creating and then executing a change set.")
	templateFile = deployCommand.Flag("template-file", "The path where your AWS CloudFormation template is located.").Required().ExistingFile()
	stackName = deployCommand.Flag("name", "The name of the AWS CloudFormation stack you're deploying to.").Required().String()
	s3Bucket = deployCommand.Flag("s3-bucket", "The name of the S3 bucket where this command uploads your CloudFormation template.").String()
	forceUpload = deployCommand.Flag("force-upload", "Indicates whether to override existing files in the S3 bucket.").Bool()
	s3Prefix = deployCommand.Flag("s3-prefix", "A prefix name that the command adds to the artifacts name when it uploads them to the S3 bucket.").String()
	kmsKeyId = deployCommand.Flag("kms-key-id", "The ID of an AWS KMS key that the command uses to encrypt artifacts that are at rest in the S3 bucket.").String()
	parameterOverrides = cli.KeyVal(deployCommand.Flag("parameter-overrides", "A list of parameter structures that specify input parameters for your stack template."))
	capabilities = deployCommand.Flag("capabilities", "A list of capabilities that you must specify before AWS Cloudformation can create certain stacks.").Enum("CAPABILITY_IAM", "CAPABILITY_NAMED_IAM")
	roleArn = deployCommand.Flag("role-arn", "The Amazon Resource Name (ARN) of an AWS Identity and Access Management (IAM) role").String()
	notificationArns = deployCommand.Flag("notification-arns", "The Amazon Resource Name (ARN) of an AWS Identity and Access Management (IAM) role.").Strings()
	failOnEmptyChangeset = deployCommand.Flag("fail-on-empty-changeset", "Specify if the CLI should return a non-zero exit code if there are no changes to be made to the stack").Bool()
	noFailOnEmptyChangeset = deployCommand.Flag("no-fail-on-empty-changeset", "Causes the CLI to return an exit code of 0 if there are no changes to be made to the stack.").Bool()
	tags = cli.CFNTags(deployCommand.Flag("tags", "A list of tags to associate with the stack that is created or updated."))
	forceDeploy = deployCommand.Flag("force-deploy", "Force CloudFormation stack deployment if it's in CREATE_FAILED state.").Bool()

	logger = logrus.New()
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
	fmt.Println(cfnSvc)
	fmt.Println(tags)
	switch command {
		case "deploy":
			break
		default:
			break
	//case "run":
	//	parameters := append(*stackParameters, *stackParametersFile...)
	//	tags := append(*stackTags, *stackTagsFile...)
	//
	//	runStackParameters := stack.NewRunStackParameters(
	//		stackName,
	//		([]*cloudformation.Parameter)(parameters),
	//		([]*cloudformation.Tag)(tags),
	//		stackTemplateBody,
	//		stackTemplateUrl,
	//		[]*string{},
	//	)
	//
	//	_, err := runCFNStack(*svc, runStackParameters, *wait, *stackForce)
	//
	//	if err != nil {
	//		logger.WithError(err).Fatal("Error while deploying stack")
	//	}
	}
}

func runCFNStack(svc cloudformation.CloudFormation, runStackParameters *stack.RunStackParameters, wait bool, force bool) (bool, error) {
	return stack.New(logger, svc).RunStack(runStackParameters, wait, force)

}