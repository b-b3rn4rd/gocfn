package main

import (
	"os"
	"strings"

	"github.com/alecthomas/kingpin"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/b-b3rn4rd/cfn/pkg/command"
	"github.com/b-b3rn4rd/cfn/pkg/deployer"
	"github.com/b-b3rn4rd/cfn/pkg/streamer"
	"github.com/b-b3rn4rd/cfn/pkg/uploader"
	"github.com/b-b3rn4rd/cfn/pkg/writer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

var (
	version   = "master"
	debug     = kingpin.Flag("debug", "Enable debug logging.").Short('d').Bool()
	logger    = logrus.New()
	outWriter = writer.New(os.Stdout, writer.JSONFormatter)
	exiter    = os.Exit
)

type Cfn struct {
	dplr   deployer.Deployeriface
	cfnSvc cloudformationiface.CloudFormationAPI
	logger *logrus.Logger
	stmr   streamer.Streameriface
}

func New(dplr deployer.Deployeriface, svc cloudformationiface.CloudFormationAPI, stmr streamer.Streameriface, logger *logrus.Logger) *Cfn {
	return &Cfn{
		dplr:   dplr,
		cfnSvc: svc,
		logger: logger,
		stmr:   stmr,
	}
}

func main() {
	kingpin.Version(version)
	runCommand := kingpin.Parse()

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
		logger.SetLevel(logrus.DebugLevel)
	}

	logger.Formatter = &logrus.JSONFormatter{}

	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2")},
	)

	cfnSvc := cloudformation.New(sess)
	s3Svc := s3.New(sess)
	dplr := deployer.New(cfnSvc, logger)

	var s3Uploader uploader.Uploaderiface
	var stmr streamer.Streameriface

	if *deployStream {
		stmr = streamer.New(cfnSvc, logger)
	}

	cfn := New(dplr, cfnSvc, stmr, logger)

	switch runCommand {
	case "deploy":
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

		cfn.deploy(
			&command.DeployParams{
				S3Uploader:           s3Uploader,
				StackName:            deployStackName,
				TemplateFile:         deployTemplateFile,
				Parameters:           ([]*cloudformation.Parameter)(*deployParameterOverrides),
				Capabilities:         aws.StringSlice(*deployCapabilities),
				NoExecuteChangeset:   deployNoExecuteChangeset,
				RoleArn:              deployRoleArn,
				NotificationArns:     aws.StringSlice(*deployNotificationArns),
				FailOnEmptyChangeset: deployFailOnEmptyChangeset,
				Tags:                 ([]*cloudformation.Tag)(*deployTags),
				ForceDeploy:          deployForceDeploy,
			},
		)
	case "package":
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
		cfn.packaage(&command.PackageParams{
			S3Uploader:   s3Uploader,
			TemplateFile: packageTemplateFile,
		})
	}

}

func (c *Cfn) deploy(deployParams *command.DeployParams) {

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
			outWriter.Write(c.dplr.DescribeStackUnsafe(deployParams.StackName))
			return
		}

		c.logger.WithError(changeSet.Err).Error("ChangeSet creation error")
		exiter(1)
		return
	}

	if *deployParams.NoExecuteChangeset {
		outWriter.Write(changeSet.ChangeSet)
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

	outWriter.Write(res.Stack)
}
