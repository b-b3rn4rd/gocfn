package cfn_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/b-b3rn4rd/gocfn/pkg/cfn"
	"github.com/b-b3rn4rd/gocfn/pkg/deployer"
	"github.com/b-b3rn4rd/gocfn/pkg/packager"
	"github.com/b-b3rn4rd/gocfn/pkg/streamer"
	"github.com/b-b3rn4rd/gocfn/pkg/uploader"
	"github.com/b-b3rn4rd/gocfn/pkg/writer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

type mockedDeployer struct {
	waitForChangeSetResp    deployer.ChangeSetRecord
	waitForExecuteResp      deployer.StackRecord
	executeChangesetErr     error
	createChangeSetResp     deployer.ChangeSetRecord
	describeStackUnsafeResp cloudformation.Stack
}

type mockerPackager struct {
	exportResp     *packager.Template
	exportErr      error
	openResp       *packager.Template
	opentErr       error
	writeOutputErr error
	marshallResp   []byte
	marshallErr    error
}

func (p mockerPackager) Export(packageParams *packager.PackageParams) (*packager.Template, error) {
	return p.exportResp, p.exportErr
}

func (p mockerPackager) Open(filename string) (*packager.Template, error) {
	return p.openResp, p.opentErr
}

func (p mockerPackager) Marshall(filename string, template *packager.Template) ([]byte, error) {
	return p.marshallResp, p.marshallErr
}

func (p mockerPackager) WriteOutput(outputTemplateFile *string, data []byte) error {
	return p.writeOutputErr
}

type mockedStreamer struct {
	describeStackEventsResp streamer.StackEventsRecord
}

func (s mockedStreamer) StartStreaming(stackName *string, seenEvents streamer.StackEvents, wr *writer.StringWriter, done <-chan bool) error {
	<-done
	return nil
}

func (s mockedStreamer) DescribeStackEvents(stackName *string, seenEvents streamer.StackEvents) (stackEvents *streamer.StackEventsRecord) {
	return &s.describeStackEventsResp
}

func (s mockedDeployer) WaitForChangeSet(stackName *string, changeSetID *string) (res *deployer.ChangeSetRecord) {
	return &s.waitForChangeSetResp
}

func (s mockedDeployer) WaitForExecute(stackName *string, changeSet *deployer.ChangeSetRecord, stmr streamer.Streameriface) (res *deployer.StackRecord) {
	return &s.waitForExecuteResp
}

func (s mockedDeployer) ExecuteChangeset(stackName *string, changeSetID *string) error {
	return s.executeChangesetErr
}

func (s mockedDeployer) CreateChangeSet(deployParams *deployer.DeployParams) *deployer.ChangeSetRecord {
	return &s.createChangeSetResp
}

func (s mockedDeployer) DescribeStackUnsafe(stackName *string) *cloudformation.Stack {
	return &s.describeStackUnsafeResp
}

func TestDeploy(t *testing.T) {
	tests := map[string]struct {
		// mocks
		dplr  deployer.Deployeriface
		strmr streamer.Streameriface
		pckgr packager.Packageriface

		// params
		s3Uploader           uploader.Uploaderiface
		stackName            *string
		templateFile         *string
		parameters           []*cloudformation.Parameter
		capabilities         []*string
		noExecuteChangeset   *bool
		roleArn              *string
		notificationArns     []*string
		failOnEmptyChangeset *bool
		tags                 []*cloudformation.Tag
		forceDeploy          *bool

		// output
		stdOut string
	}{
		"deploy calls fatal error if CreateChangeSet produced an error": {
			dplr: mockedDeployer{
				createChangeSetResp: deployer.ChangeSetRecord{
					Err: errors.New("error"),
				},
			},
		},
		"deploy calls fatal error if WaitForChangeSet produced an error and its not empty stack": {
			failOnEmptyChangeset: aws.Bool(false),
			dplr: mockedDeployer{
				createChangeSetResp: deployer.ChangeSetRecord{
					ChangeSet: &cloudformation.DescribeChangeSetOutput{
						ChangeSetId: aws.String("1"),
					},
				},
				waitForChangeSetResp: deployer.ChangeSetRecord{
					Err: errors.New("error"),
					ChangeSet: &cloudformation.DescribeChangeSetOutput{
						StackId:     aws.String("hello"),
						ChangeSetId: aws.String("1"),
					},
				},
			},
		},
		"deploy returns stack information if WaitForChangeSet reports empty changeset": {
			failOnEmptyChangeset: aws.Bool(false),
			dplr: mockedDeployer{
				createChangeSetResp: deployer.ChangeSetRecord{
					ChangeSet: &cloudformation.DescribeChangeSetOutput{
						ChangeSetId: aws.String("1"),
					},
				},
				waitForChangeSetResp: deployer.ChangeSetRecord{
					Err: errors.New("The submitted information didn't contain changes."),
					ChangeSet: &cloudformation.DescribeChangeSetOutput{
						StackId:     aws.String("hello"),
						ChangeSetId: aws.String("1"),
					},
				},
				describeStackUnsafeResp: cloudformation.Stack{
					StackId: aws.String("hello"),
				},
			},
			stdOut: func() string {
				s := cloudformation.Stack{
					StackId: aws.String("hello"),
				}
				raw, _ := json.MarshalIndent(s, "", "    ")

				return string(raw) + "\n"
			}(),
		},
		"deploy calls fatal error if WaitForChangeSet reports and changeset and failOnEmptyChangeset": {
			failOnEmptyChangeset: aws.Bool(true),
			dplr: mockedDeployer{
				createChangeSetResp: deployer.ChangeSetRecord{
					ChangeSet: &cloudformation.DescribeChangeSetOutput{
						ChangeSetId: aws.String("1"),
					},
				},
				waitForChangeSetResp: deployer.ChangeSetRecord{
					Err: errors.New("The submitted information didn't contain changes."),
					ChangeSet: &cloudformation.DescribeChangeSetOutput{
						StackId:     aws.String("hello"),
						ChangeSetId: aws.String("1"),
					},
				},
			},
		},
		"deploy returns changeSet if noExecuteChangeset is given": {
			failOnEmptyChangeset: aws.Bool(false),
			noExecuteChangeset:   aws.Bool(true),
			dplr: mockedDeployer{
				createChangeSetResp: deployer.ChangeSetRecord{
					ChangeSet: &cloudformation.DescribeChangeSetOutput{
						StackId:     aws.String("hello"),
						ChangeSetId: aws.String("1"),
					},
				},
				waitForChangeSetResp: deployer.ChangeSetRecord{
					ChangeSet: &cloudformation.DescribeChangeSetOutput{
						StackId:     aws.String("hello"),
						ChangeSetId: aws.String("1"),
					},
				},
			},
			stdOut: func() string {
				s := cloudformation.DescribeChangeSetOutput{
					StackId:     aws.String("hello"),
					ChangeSetId: aws.String("1"),
				}
				raw, _ := json.MarshalIndent(s, "", "    ")

				return string(raw) + "\n"
			}(),
		},
		"deploy calls fatal error if ExecuteChangeset return an error": {
			failOnEmptyChangeset: aws.Bool(false),
			noExecuteChangeset:   aws.Bool(false),
			dplr: mockedDeployer{
				createChangeSetResp: deployer.ChangeSetRecord{
					ChangeSet: &cloudformation.DescribeChangeSetOutput{
						ChangeSetId: aws.String("1"),
					},
				},
				waitForChangeSetResp: deployer.ChangeSetRecord{
					ChangeSet: &cloudformation.DescribeChangeSetOutput{
						StackId:     aws.String("hello"),
						ChangeSetId: aws.String("1"),
					},
				},
				executeChangesetErr: errors.New("some error"),
			},
		},
		"deploy calls fatal error if DescribeStackEvents return an error": {
			failOnEmptyChangeset: aws.Bool(false),
			noExecuteChangeset:   aws.Bool(false),
			dplr: mockedDeployer{
				createChangeSetResp: deployer.ChangeSetRecord{
					ChangeSet: &cloudformation.DescribeChangeSetOutput{
						ChangeSetId: aws.String("1"),
					},
				},
				waitForChangeSetResp: deployer.ChangeSetRecord{
					ChangeSet: &cloudformation.DescribeChangeSetOutput{
						StackId:     aws.String("hello"),
						ChangeSetId: aws.String("1"),
					},
				},
			},
			strmr: mockedStreamer{
				describeStackEventsResp: streamer.StackEventsRecord{
					Err: errors.New("error"),
				},
			},
		},
		"deploy calls fatal error if WaitForExecute return an error": {
			failOnEmptyChangeset: aws.Bool(false),
			noExecuteChangeset:   aws.Bool(false),
			dplr: mockedDeployer{
				createChangeSetResp: deployer.ChangeSetRecord{
					ChangeSet: &cloudformation.DescribeChangeSetOutput{
						ChangeSetId: aws.String("1"),
					},
				},
				waitForChangeSetResp: deployer.ChangeSetRecord{
					ChangeSet: &cloudformation.DescribeChangeSetOutput{
						StackId:     aws.String("hello"),
						ChangeSetId: aws.String("1"),
					},
				},
				waitForExecuteResp: deployer.StackRecord{
					Err: errors.New("me here"),
				},
			},
		},

		"deploy write stack output to stdout": {
			failOnEmptyChangeset: aws.Bool(false),
			noExecuteChangeset:   aws.Bool(false),
			dplr: mockedDeployer{
				createChangeSetResp: deployer.ChangeSetRecord{
					ChangeSet: &cloudformation.DescribeChangeSetOutput{
						ChangeSetId: aws.String("1"),
					},
				},
				waitForChangeSetResp: deployer.ChangeSetRecord{
					ChangeSet: &cloudformation.DescribeChangeSetOutput{
						StackId:     aws.String("hello"),
						ChangeSetId: aws.String("1"),
					},
				},
				waitForExecuteResp: deployer.StackRecord{
					Stack: &cloudformation.Stack{
						StackId:     aws.String("hello"),
						StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
					},
				},
			},

			stdOut: func() string {
				s := cloudformation.Stack{
					StackId:     aws.String("hello"),
					StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
				}
				raw, _ := json.MarshalIndent(s, "", "    ")

				return string(raw) + "\n"
			}(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			stdOut := &bytes.Buffer{}

			logger, hook := logrustest.NewNullLogger()
			cfn := cfn.NewWithOptions(
				cfn.Deployer(test.dplr),
				cfn.Packager(test.pckgr),
				cfn.Streamer(test.strmr),
				cfn.Logger(logger))

			cfn.Deploy(&deployer.DeployParams{
				test.s3Uploader,
				aws.StringValue(test.stackName),
				aws.StringValue(test.templateFile),
				test.parameters,
				aws.StringValueSlice(test.capabilities),
				aws.BoolValue(test.noExecuteChangeset),
				aws.StringValue(test.roleArn),
				aws.StringValueSlice(test.notificationArns),
				aws.BoolValue(test.failOnEmptyChangeset),
				test.tags,
				aws.BoolValue(test.forceDeploy),
			},
			)

			if hook.LastEntry() != nil {
				assert.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level)
			}

			assert.Equal(t, test.stdOut, stdOut.String())
		})
	}
}

func TestPackage(t *testing.T) {
	tests := map[string]struct {
		packageParams  *packager.PackageParams
		exportResp     *packager.Template
		exportErr      error
		writeOutputErr error
		marshallResp   []byte
		marshallErr    error
	}{
		"exits with error with Export has error": {
			exportErr: errors.New("error"),
			packageParams: &packager.PackageParams{
				TemplateFile: "example.yml",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			pkgr := &mockerPackager{
				exportErr: test.exportErr,
			}

			logger, hook := logrustest.NewNullLogger()
			cfn := cfn.NewWithOptions(cfn.Logger(logger), cfn.Packager(pkgr))
			cfn.Package(test.packageParams)

			if hook.LastEntry() != nil {
				assert.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level)
			}
		})
	}
}
