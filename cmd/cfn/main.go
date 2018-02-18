package main

import (
	"fmt"
	"github.com/b-b3rn4rd/cfn/pkg/stack"
	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	//"time"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"io/ioutil"
)
var (
	version = "master"
	tracing = kingpin.Flag("trace", "Enable trace mode.").Short('t').Bool()
	debug   = kingpin.Flag("debug", "Enable debug logging.").Short('d').Bool()
	runCommand = kingpin.Command("run", "Create or update CloudFormation stack.")
	stackName = runCommand.Arg("name", "CloudFormation stack name.").String()
	stackTemplate = runCommand.Flag("template", "CloudFormation template.").ExistingFile()
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
		err := runCFNStack(*stackName, *svc, *stackTemplate)
		if err != nil {
			logger.WithError(err).Fatal("failed to process log data")
		}
	default:
		fmt.Printf(command)
	}
}

func runCFNStack(stackName string, svc cloudformation.CloudFormation, stackTemplate string) error {
	b, err := ioutil.ReadFile(stackTemplate)

	if err != nil {
		logger.WithError(err).Fatal("Failed to readfile")
	}

	fmt.Println(string(b))
	stack.New(logger, svc).RunStack(stackName)

	return nil
}