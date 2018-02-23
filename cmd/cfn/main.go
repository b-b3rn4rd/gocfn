package main

import (
	"github.com/b-b3rn4rd/cfn/pkg/stack"
	"github.com/b-b3rn4rd/cfn/pkg/cli"
	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

var (
	version = "master"
	tracing = kingpin.Flag("trace", "Enable trace mode.").Short('t').Bool()
	debug   = kingpin.Flag("debug", "Enable debug logging.").Short('d').Bool()
	wait    = kingpin.Flag("wait", "Wait for stack completion.").Bool()
	runCommand = kingpin.Command("run", "Create or update CloudFormation stack.")
	stackName = runCommand.Arg("name", "The name that is associated with the stack.").String()
	stackTemplateBody = runCommand.Flag("template-body", "Structure containing the template body." ).ExistingFile()
	stackTemplateUrl = runCommand.Flag("template-url", "Location of file containing the template body." ).String()
	stackParameters = cli.CFNParameters(runCommand.Flag("parameters", "A list of Parameter structures that specify input parameters for the stack."))
	stackTags = cli.CFNTags(runCommand.Flag("tags", "Key-value pairs to associate with this stack."))
	stackParametersFile = cli.CFNParameters(runCommand.Flag("parameters-file", "A list of Parameter structures that specify input parameters for the stack."))
	stackTagsFile = cli.CFNTags(runCommand.Flag("tags-file", "Key-value pairs to associate with this stck."))
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
	svc := cloudformation.New(sess)

	switch command {
	case "run":
		parameters := append(*stackParameters, *stackParametersFile...)
		tags := append(*stackTags, *stackTagsFile...)

		runStackParameters := stack.NewRunStackParameters(
			stackName,
			([]*cloudformation.Parameter)(parameters),
			([]*cloudformation.Tag)(tags),
			stackTemplateBody,
			stackTemplateUrl,
			[]*string{},
		)

		err := runCFNStack(*svc, runStackParameters)

		if err != nil {
			logger.WithError(err).Fatal("failed to process log data")
		}
	}
}

func runCFNStack(svc cloudformation.CloudFormation, runStackParameters *stack.RunStackParameters) error {
	stack.New(logger, svc).RunStack(runStackParameters)
	return nil
}