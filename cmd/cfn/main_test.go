package main

import (
	"testing"
	"github.com/b-b3rn4rd/cfn/pkg/streamer"
	"github.com/b-b3rn4rd/cfn/pkg/deployer"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/b-b3rn4rd/cfn/pkg/uploader"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/b-b3rn4rd/cfn/pkg/writer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/sirupsen/logrus"
)

type mockedDeployer struct {
	waitForChangeSetResp deployer.ChangeSetRecord
	waitForExecuteResp deployer.StackRecord
	executeChangesetErr error
	createChangeSetResp deployer.ChangeSetRecord
	describeStackUnsafeResp cloudformation.Stack
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

func (s mockedDeployer) WaitForChangeSet(stackName *string, changeSetId *string) (res *deployer.ChangeSetRecord) {
	return &s.waitForChangeSetResp
}

func (s mockedDeployer) WaitForExecute(stackName *string, changeSet *deployer.ChangeSetRecord, stmr streamer.Streameriface) (res *deployer.StackRecord) {
	return &s.waitForExecuteResp
}

func (s mockedDeployer) ExecuteChangeset(stackName *string, changeSetId *string) error {
	return s.executeChangesetErr
}

func (s mockedDeployer) CreateChangeSet(stackName *string, templateFile *string, parameters []*cloudformation.Parameter, capabilities []*string, noExecuteChangeset *bool, roleArn *string, notificationArns []*string, tags []*cloudformation.Tag, forceDeploy *bool, s3Uploader uploader.Uploaderiface) *deployer.ChangeSetRecord {
	return &s.createChangeSetResp
}


func (s mockedDeployer) DescribeStackUnsafe(stackName *string) *cloudformation.Stack {
	return &s.describeStackUnsafeResp
}

func TestDeploy(t *testing.T)  {
	var cfnSvc cloudformationiface.CloudFormationAPI

	tests := map[string]struct{
		// mocks
		dplr deployer.Deployeriface
		strmr streamer.Streameriface

		// params
		s3Uploader uploader.Uploaderiface
		stackName *string
		templateFile *string
		parameters []*cloudformation.Parameter
		capabilities []*string
		noExecuteChangeset *bool
		roleArn *string
		notificationArns []*string
		failOnEmptyChangeset *bool
		tags []*cloudformation.Tag
		forceDeploy *bool
	} {
		"deploy calls fatal error if CreateChangeSet produced an error": {
			dplr: mockedDeployer{
				createChangeSetResp: deployer.ChangeSetRecord{
					Err: errors.New("error"),
				},
			},
		},
		"deploy calls fatal error if WaitForChangeSet produced an error and its not empty stack": {
			dplr: mockedDeployer{
				createChangeSetResp: deployer.ChangeSetRecord{
					Err: errors.New("error"),
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			logger, hook := logrustest.NewNullLogger()
			cfn := New(test.dplr, cfnSvc, test.strmr, logger)
			cfn.deploy(
				test.s3Uploader,
				test.stackName,
				test.templateFile,
				test.parameters,
				test.capabilities,
				test.noExecuteChangeset,
				test.roleArn,
				test.notificationArns,
				test.failOnEmptyChangeset,
				test.tags,
				test.forceDeploy,
			)
			assert.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level)
		})
	}
}
