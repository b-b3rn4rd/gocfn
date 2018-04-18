package cfn

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/b-b3rn4rd/gocfn/pkg/deployer"
	"github.com/b-b3rn4rd/gocfn/pkg/packager"
	"github.com/b-b3rn4rd/gocfn/pkg/streamer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

type Params interface {
	String() string
}

type Cfn struct {
	dplr   deployer.Deployeriface
	pckgr  packager.Packageriface
	stmr   streamer.Streameriface
	logger *logrus.Logger
}

func Deployer(dplr deployer.Deployeriface) func(cfn *Cfn) {
	return func(cfn *Cfn) {
		cfn.dplr = dplr
	}
}

func Packager(pckgr packager.Packageriface) func(cfn *Cfn) {
	return func(cfn *Cfn) {
		cfn.pckgr = pckgr
	}
}

func Streamer(stmr streamer.Streameriface) func(cfn *Cfn) {
	return func(cfn *Cfn) {
		cfn.stmr = stmr
	}
}

func Logger(logger *logrus.Logger) func(cfn *Cfn) {
	return func(cfn *Cfn) {
		cfn.logger = logger
	}
}

func New(sess client.ConfigProvider, logger *logrus.Logger, streamRequired bool) *Cfn {
	cfnSvc := cloudformation.New(sess)

	dplr := deployer.New(cfnSvc, logger)
	pckgr := packager.New(logger, afero.NewOsFs())

	var stmr streamer.Streameriface

	if streamRequired {
		stmr = streamer.New(cfnSvc, logger)
	}

	return &Cfn{
		dplr:   dplr,
		pckgr:  pckgr,
		stmr:   stmr,
		logger: logger,
	}
}

func NewWithOptions(options ...func(cfn *Cfn)) *Cfn {
	gocfn := &Cfn{}

	for _, option := range options {
		option(gocfn)
	}

	return gocfn
}

func (c *Cfn) Deploy(deployParams *deployer.DeployParams) (interface{}, error) {

	changeSet := c.dplr.CreateChangeSet(deployParams)

	if changeSet.Err != nil {
		return "", errors.Wrap(changeSet.Err, "changeSet creation error")
	}

	changeSetResult := c.dplr.WaitForChangeSet(
		aws.String(deployParams.StackName),
		changeSet.ChangeSet.ChangeSetId,
	)

	changeSet.ChangeSet = changeSetResult.ChangeSet
	changeSet.Err = changeSetResult.Err

	if changeSet.Err != nil {
		isEmptyChangeSet := strings.Contains(changeSet.Err.Error(), "The submitted information didn't contain changes.")

		if !deployParams.FailOnEmptyChangeset && isEmptyChangeSet {
			return c.dplr.DescribeStackUnsafe(aws.String(deployParams.StackName)), nil
		}

		return "", errors.Wrap(changeSet.Err, "changeSet creation error")

	}

	if deployParams.NoExecuteChangeset {
		return changeSet.ChangeSet, nil
	}

	if c.stmr != nil {
		seenStackEvents := c.stmr.DescribeStackEvents(aws.String(deployParams.StackName), nil)
		if seenStackEvents.Err != nil {
			return "", errors.Wrap(seenStackEvents.Err, "error while gathering stack events")
		}

		changeSet.StackEvents = seenStackEvents.Records
	}

	err := c.dplr.ExecuteChangeset(aws.String(deployParams.StackName), changeSet.ChangeSet.ChangeSetId)
	if err != nil {
		return "", errors.Wrap(err, "changeSet execution error")
	}

	res := c.dplr.WaitForExecute(aws.String(deployParams.StackName), changeSet, c.stmr)
	if res.Err != nil {
		return "", errors.Wrap(res.Err, "changeSet execution error")
	}

	return res.Stack, nil
}

func (c *Cfn) Package(packageParams *packager.PackageParams) (string, error) {
	template, err := c.pckgr.Export(packageParams)
	if err != nil {
		return "", errors.Wrap(err, "error while exporting package")
	}

	raw, err := c.pckgr.Marshall(packageParams.TemplateFile, template)

	if err != nil {
		return "", errors.Wrap(err, "error while marshalling template")
	}

	if packageParams.OutputTemplateFile == "" {
		c.logger.Debug("output file is not specified, sending to stdout")
		return string(raw), nil
	}

	err = c.pckgr.WriteOutput(aws.String(packageParams.OutputTemplateFile), raw)

	if err != nil {
		return "", errors.Wrap(err, "error while writing output")

	}

	return "", nil
}
