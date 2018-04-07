package main

import (
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/b-b3rn4rd/gocfn/pkg/command"
	"github.com/b-b3rn4rd/gocfn/pkg/deployer"
	"github.com/b-b3rn4rd/gocfn/pkg/packager"
	"github.com/b-b3rn4rd/gocfn/pkg/streamer"
	"github.com/b-b3rn4rd/gocfn/pkg/uploader"
	"github.com/b-b3rn4rd/gocfn/pkg/writer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

var (
	version       = "master"
	debug         = kingpin.Flag("debug", "Enable debug logging.").Short('d').Bool()
	logger        = logrus.New()
	jsonOutWriter = writer.New(os.Stdout, writer.JSONFormatter)
	strOutWriter  = writer.New(os.Stdout, writer.PlainFormatter)
	exiter        = os.Exit
)

type GoCfn struct {
	dplr   deployer.Deployeriface
	pckgr  packager.Packageriface
	stmr   streamer.Streameriface
	logger *logrus.Logger
}

func New(dplr deployer.Deployeriface, pckgr packager.Packageriface, stmr streamer.Streameriface, logger *logrus.Logger) *GoCfn {
	return &GoCfn{
		dplr:   dplr,
		pckgr:  pckgr,
		stmr:   stmr,
		logger: logger,
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
	pckgr := packager.New(logger, afero.NewOsFs())

	var s3Uploader uploader.Uploaderiface
	var stmr streamer.Streameriface

	if *deployStream {
		stmr = streamer.New(cfnSvc, logger)
	}

	cfn := New(dplr, pckgr, stmr, logger)

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

		cfn.deploy(&command.DeployParams{
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
		})

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
			S3Uploader:         s3Uploader,
			TemplateFile:       packageTemplateFile,
			OutputTemplateFile: packageOutputTemplateFile,
		})
	}

}
