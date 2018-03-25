package deployer_test

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/b-b3rn4rd/cfn/pkg/command"
	"github.com/b-b3rn4rd/cfn/pkg/deployer"
	"github.com/b-b3rn4rd/cfn/pkg/streamer"
	"github.com/b-b3rn4rd/cfn/pkg/uploader"
	"github.com/b-b3rn4rd/cfn/pkg/writer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type mockedCloudFormationAPI struct {
	describeStacksOutput                cloudformation.DescribeStacksOutput
	describeStacksErr                   error
	createChangeSetOutput               cloudformation.CreateChangeSetOutput
	createChangeSetErr                  error
	deleteStackOutput                   cloudformation.DeleteStackOutput
	deleteStackErr                      error
	waitUntilStackDeleteCompleteErr     error
	waitUntilChangeSetCreateCompleteErr error
	describeChangeSetOutput             cloudformation.DescribeChangeSetOutput
	describeChangeSetErr                error
	executeChangeSetOutput              cloudformation.ExecuteChangeSetOutput
	executeChangeSetErr                 error
	cloudformationiface.CloudFormationAPI
	waitUntilStackCreateCompleteErr error
	waitUntilStackUpdateCompleteErr error
}

type mockedStreamer struct {
	startStreaming            error
	describeStackEventsOutput streamer.StackEventsRecord
}

func (s mockedStreamer) StartStreaming(stackName *string, seenEvents streamer.StackEvents, wr *writer.StringWriter, done <-chan bool) error {
	<-done
	return s.startStreaming
}

func (s mockedStreamer) DescribeStackEvents(stackName *string, seenEvents streamer.StackEvents) (stackEvents *streamer.StackEventsRecord) {
	return &s.describeStackEventsOutput
}

func (m mockedCloudFormationAPI) DescribeStacks(input *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
	return &m.describeStacksOutput, m.describeStacksErr
}

func (m mockedCloudFormationAPI) CreateChangeSet(input *cloudformation.CreateChangeSetInput) (*cloudformation.CreateChangeSetOutput, error) {
	return &m.createChangeSetOutput, m.createChangeSetErr
}

func (m mockedCloudFormationAPI) DeleteStack(input *cloudformation.DeleteStackInput) (*cloudformation.DeleteStackOutput, error) {
	return &m.deleteStackOutput, m.deleteStackErr
}

func (m mockedCloudFormationAPI) WaitUntilStackDeleteComplete(input *cloudformation.DescribeStacksInput) error {
	return m.waitUntilStackDeleteCompleteErr
}

func (m mockedCloudFormationAPI) WaitUntilChangeSetCreateComplete(input *cloudformation.DescribeChangeSetInput) error {
	return m.waitUntilChangeSetCreateCompleteErr
}

func (m mockedCloudFormationAPI) DescribeChangeSet(input *cloudformation.DescribeChangeSetInput) (*cloudformation.DescribeChangeSetOutput, error) {
	return &m.describeChangeSetOutput, m.describeChangeSetErr
}

func (m mockedCloudFormationAPI) ExecuteChangeSet(input *cloudformation.ExecuteChangeSetInput) (*cloudformation.ExecuteChangeSetOutput, error) {
	return &m.executeChangeSetOutput, m.executeChangeSetErr
}

func (m mockedCloudFormationAPI) WaitUntilStackCreateComplete(input *cloudformation.DescribeStacksInput) error {
	return m.waitUntilStackCreateCompleteErr
}

func (m mockedCloudFormationAPI) WaitUntilStackUpdateComplete(input *cloudformation.DescribeStacksInput) error {
	return m.waitUntilStackUpdateCompleteErr
}

func TestCreateChangeSet(t *testing.T) {
	tests := map[string]struct {
		stackName          *string
		templateFile       *string
		parameters         []*cloudformation.Parameter
		capabilities       []*string
		noExecuteChangeset *bool
		roleArn            *string
		notificationArns   []*string
		tags               []*cloudformation.Tag
		forceDeploy        *bool
		s3Uploader         uploader.Uploaderiface
		// func
		describeStacksOutput  cloudformation.DescribeStacksOutput
		describeStacksErr     error
		createChangeSetOutput cloudformation.CreateChangeSetOutput
		createChangeSetErr    error
		cloudformationiface.CloudFormationAPI
		deleteStackOutput cloudformation.DeleteStackOutput
		deleteStackErr    error
		// resp
		changeSetRecord *deployer.ChangeSetRecord
	}{
		"CreateChangeSet returns error if cant describe stack": {
			stackName:          aws.String("hello"),
			templateFile:       aws.String("template.yml"),
			parameters:         []*cloudformation.Parameter{},
			capabilities:       []*string{},
			noExecuteChangeset: aws.Bool(false),
			notificationArns:   []*string{},
			tags:               []*cloudformation.Tag{},
			forceDeploy:        aws.Bool(false),
			describeStacksErr:  errors.New("cant describe stack error"),
			changeSetRecord: &deployer.ChangeSetRecord{
				Err:       errors.Wrap(errors.New("cant describe stack error"), "AWS error while running DescribeStack"),
				ChangeSet: &cloudformation.DescribeChangeSetOutput{},
			},
		},
		"CreateChangeSet returns error if stack is in created failed and no force is specified": {
			stackName:          aws.String("hello"),
			templateFile:       aws.String("template.yml"),
			parameters:         []*cloudformation.Parameter{},
			capabilities:       []*string{},
			noExecuteChangeset: aws.Bool(false),
			roleArn:            aws.String(""),
			notificationArns:   []*string{},
			tags:               []*cloudformation.Tag{},
			forceDeploy:        aws.Bool(false),
			describeStacksOutput: cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{{
					StackStatus: aws.String(cloudformation.StackStatusCreateFailed),
				}},
			},
			changeSetRecord: &deployer.ChangeSetRecord{
				Err:       errors.New("stack is in CREATE_FAILED and can't be updated, unless --force is specified"),
				ChangeSet: &cloudformation.DescribeChangeSetOutput{},
			},
		},
		"CreateChangeSet is created if stack is in created failed and force is specified": {
			stackName:          aws.String("hello"),
			templateFile:       aws.String("template.yml"),
			parameters:         []*cloudformation.Parameter{},
			capabilities:       []*string{},
			noExecuteChangeset: aws.Bool(false),
			roleArn:            aws.String(""),
			notificationArns:   []*string{},
			tags:               []*cloudformation.Tag{},
			forceDeploy:        aws.Bool(true),
			deleteStackOutput:  cloudformation.DeleteStackOutput{},
			describeStacksOutput: cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{{
					StackStatus: aws.String(cloudformation.StackStatusCreateFailed),
				}},
			},
			changeSetRecord: &deployer.ChangeSetRecord{
				ChangeSet: &cloudformation.DescribeChangeSetOutput{
					ChangeSetId: aws.String("test"),
				},
				ChangeSetType: aws.String(cloudformation.ChangeSetTypeCreate),
			},
			createChangeSetOutput: cloudformation.CreateChangeSetOutput{
				Id: aws.String("test"),
			},
		},
		"CreateChangeSet is failed if stack deletion has failed": {
			stackName:          aws.String("hello"),
			templateFile:       aws.String("template.yml"),
			parameters:         []*cloudformation.Parameter{},
			capabilities:       []*string{},
			noExecuteChangeset: aws.Bool(false),
			roleArn:            aws.String(""),
			notificationArns:   []*string{},
			tags:               []*cloudformation.Tag{},
			forceDeploy:        aws.Bool(true),
			deleteStackErr:     errors.New("cant delete stack"),
			describeStacksOutput: cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{{
					StackStatus: aws.String(cloudformation.StackStatusCreateFailed),
				}},
			},
			changeSetRecord: &deployer.ChangeSetRecord{
				Err:       errors.Wrap(errors.New("cant delete stack"), "Error while running DeleteStack"),
				ChangeSet: &cloudformation.DescribeChangeSetOutput{},
			},
		},
		"CreateChangeSet is updated if stack exists": {
			stackName:          aws.String("hello"),
			templateFile:       aws.String("template.yml"),
			parameters:         []*cloudformation.Parameter{},
			capabilities:       []*string{},
			noExecuteChangeset: aws.Bool(false),
			roleArn:            aws.String(""),
			notificationArns:   []*string{},
			tags:               []*cloudformation.Tag{},
			forceDeploy:        aws.Bool(true),
			deleteStackOutput:  cloudformation.DeleteStackOutput{},
			describeStacksOutput: cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{{
					StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
				}},
			},
			changeSetRecord: &deployer.ChangeSetRecord{
				ChangeSet: &cloudformation.DescribeChangeSetOutput{
					ChangeSetId: aws.String("test"),
				},
				ChangeSetType: aws.String(cloudformation.ChangeSetTypeUpdate),
			},
			createChangeSetOutput: cloudformation.CreateChangeSetOutput{
				Id: aws.String("test"),
			},
		},
	}

	for name, test := range tests {
		svc := mockedCloudFormationAPI{
			createChangeSetOutput: test.createChangeSetOutput,
			createChangeSetErr:    test.createChangeSetErr,
			describeStacksOutput:  test.describeStacksOutput,
			describeStacksErr:     test.describeStacksErr,
			deleteStackOutput:     test.deleteStackOutput,
			deleteStackErr:        test.deleteStackErr,
		}

		t.Run(name, func(t *testing.T) {
			d := deployer.New(svc, logrus.New())
			resp := d.CreateChangeSet(&command.DeployParams{
				StackName:          test.stackName,
				TemplateFile:       test.templateFile,
				Parameters:         test.parameters,
				Capabilities:       test.capabilities,
				NoExecuteChangeset: test.noExecuteChangeset,
				RoleArn:            test.roleArn,
				NotificationArns:   test.notificationArns,
				Tags:               test.tags,
				ForceDeploy:        test.forceDeploy,
				S3Uploader:         test.s3Uploader,
			},
			)

			if resp.Err != nil {
				assert.EqualError(t, test.changeSetRecord.Err, resp.Err.Error())
			}

			assert.Equal(t, test.changeSetRecord.StackEvents, resp.StackEvents)
			assert.Equal(t, test.changeSetRecord.ChangeSetType, resp.ChangeSetType)
			assert.Equal(t, test.changeSetRecord.ChangeSet, resp.ChangeSet)
		})
	}
}

func TestWaitForChangeSet(t *testing.T) {
	tests := map[string]struct {
		stackName                           *string
		changeSetId                         *string
		waitUntilChangeSetCreateCompleteErr error
		describeChangeSetOutput             cloudformation.DescribeChangeSetOutput
		describeChangeSetErr                error
		// resp
		changeSetRecord *deployer.ChangeSetRecord
	}{
		"WaitForChangeSet returns errror if WaitUntilChangeSetCreateComplete has failed with unknown error": {
			stackName:                           aws.String("test-stack"),
			changeSetId:                         aws.String("one"),
			waitUntilChangeSetCreateCompleteErr: errors.New("wait error"),
			changeSetRecord: &deployer.ChangeSetRecord{
				Err: errors.Wrap(errors.New("wait error"), "AWS error while running WaitUntilChangeSetCreateComplete"),
				ChangeSet: &cloudformation.DescribeChangeSetOutput{
					StatusReason: aws.String("unknown reason"),
				},
			},
			describeChangeSetOutput: cloudformation.DescribeChangeSetOutput{
				StatusReason: aws.String("unknown reason"),
			},
		},
		"WaitForChangeSet returns custom errror if changeset does not contain changes": {
			stackName:                           aws.String("test-stack"),
			changeSetId:                         aws.String("one"),
			waitUntilChangeSetCreateCompleteErr: errors.New("The submitted information didn't contain changes."),
			changeSetRecord: &deployer.ChangeSetRecord{
				Err: errors.New("The submitted information didn't contain changes."),
				ChangeSet: &cloudformation.DescribeChangeSetOutput{
					StatusReason: aws.String("The submitted information didn't contain changes."),
				},
			},
			describeChangeSetOutput: cloudformation.DescribeChangeSetOutput{
				StatusReason: aws.String("The submitted information didn't contain changes."),
			},
		},
		"WaitForChangeSet fills ChangeSet field with DescribeChangeSet information": {
			stackName:   aws.String("test-stack"),
			changeSetId: aws.String("one"),
			changeSetRecord: &deployer.ChangeSetRecord{

				ChangeSet: &cloudformation.DescribeChangeSetOutput{
					Status:       aws.String("all good"),
					StatusReason: aws.String("no particular reason"),
					ChangeSetId:  aws.String("one"),
				},
			},
			describeChangeSetOutput: cloudformation.DescribeChangeSetOutput{
				Status:       aws.String("all good"),
				StatusReason: aws.String("no particular reason"),
				ChangeSetId:  aws.String("one"),
			},
		},
	}

	for name, test := range tests {
		svc := mockedCloudFormationAPI{
			waitUntilChangeSetCreateCompleteErr: test.waitUntilChangeSetCreateCompleteErr,
			describeChangeSetOutput:             test.describeChangeSetOutput,
			describeChangeSetErr:                test.describeChangeSetErr,
		}

		t.Run(name, func(t *testing.T) {
			d := deployer.New(svc, logrus.New())
			resp := d.WaitForChangeSet(test.stackName, test.changeSetId)

			if resp.Err != nil {
				assert.EqualError(t, test.changeSetRecord.Err, resp.Err.Error())
			}

			assert.Equal(t, test.changeSetRecord.ChangeSetType, resp.ChangeSetType)
			assert.Equal(t, test.changeSetRecord.ChangeSet, resp.ChangeSet)
		})
	}
}

func TestExecuteChangeset(t *testing.T) {
	tests := map[string]struct {
		stackName   *string
		changeSetId *string
		// func
		executeChangeSetOutput cloudformation.ExecuteChangeSetOutput
		executeChangeSetErr    error
		//
		err error
	}{
		"ExecuteChangeset returns nil when no errors occurred": {
			stackName:              aws.String("test-stack"),
			changeSetId:            aws.String("one"),
			executeChangeSetOutput: cloudformation.ExecuteChangeSetOutput{},
		},
		"ExecuteChangeset returns err error occurred": {
			stackName:           aws.String("test-stack"),
			changeSetId:         aws.String("one"),
			executeChangeSetErr: errors.New("execute error"),
			err:                 errors.Wrap(errors.New("execute error"), "AWS error while running ExecuteChangeSet"),
		},
	}

	for name, test := range tests {
		svc := mockedCloudFormationAPI{
			executeChangeSetOutput: test.executeChangeSetOutput,
			executeChangeSetErr:    test.executeChangeSetErr,
		}

		t.Run(name, func(t *testing.T) {
			d := deployer.New(svc, logrus.New())
			err := d.ExecuteChangeset(test.stackName, test.changeSetId)
			if err != nil {
				assert.EqualError(t, test.err, err.Error())
			}
		})
	}
}

func TestWaitForExecute(t *testing.T) {
	tests := map[string]struct {
		stackName                       *string
		changeSet                       *deployer.ChangeSetRecord
		waitUntilStackCreateCompleteErr error
		waitUntilStackUpdateCompleteErr error
		describeStacksOutput            cloudformation.DescribeStacksOutput
		//
		stackRecord deployer.StackRecord
		// streamer
		stmr streamer.Streameriface
	}{
		"Response contains custom error if WaitUntilStackCreateComplete returned an error": {
			stackName: aws.String("test-stack"),
			changeSet: &deployer.ChangeSetRecord{
				ChangeSetType: aws.String(cloudformation.ChangeSetTypeCreate),
			},
			waitUntilStackCreateCompleteErr: errors.New("error occurred"),
			stackRecord: deployer.StackRecord{
				Stack: &cloudformation.Stack{
					StackStatus: aws.String(cloudformation.StackStatusCreateFailed),
				},
				Err: errors.New(fmt.Sprintf("failed creating/updating stack, status: %s", cloudformation.StackStatusCreateFailed)),
			},
			describeStacksOutput: cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{{
					StackStatus: aws.String(cloudformation.StackStatusCreateFailed),
				}},
			},
		},
		"Response contains error if WaitUntilStackUpdateComplete returned an error": {
			stackName: aws.String("test-stack"),
			changeSet: &deployer.ChangeSetRecord{
				ChangeSetType: aws.String(cloudformation.ChangeSetTypeUpdate),
			},
			waitUntilStackUpdateCompleteErr: errors.New("error occurred"),
			stackRecord: deployer.StackRecord{
				Stack: &cloudformation.Stack{
					StackStatus: aws.String(cloudformation.StackStatusUpdateRollbackFailed),
				},
				Err: errors.New(fmt.Sprintf("failed creating/updating stack, status: %s", cloudformation.StackStatusUpdateRollbackFailed)),
			},
			describeStacksOutput: cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{{
					StackStatus: aws.String(cloudformation.StackStatusUpdateRollbackFailed),
				}},
			},
		},
		"Response contains no erros if stack was created": {
			stackName: aws.String("test-stack"),
			changeSet: &deployer.ChangeSetRecord{
				ChangeSetType: aws.String(cloudformation.ChangeSetTypeCreate),
			},
			stackRecord: deployer.StackRecord{
				Stack: &cloudformation.Stack{
					StackStatus: aws.String(cloudformation.StackStatusCreateFailed),
				},
			},
			describeStacksOutput: cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{{
					StackStatus: aws.String(cloudformation.StackStatusCreateFailed),
				}},
			},
		},
		"Streamer is invoked if specified": {
			stackName: aws.String("test-stack"),
			changeSet: &deployer.ChangeSetRecord{
				ChangeSetType: aws.String(cloudformation.ChangeSetTypeCreate),
			},
			stackRecord: deployer.StackRecord{
				Stack: &cloudformation.Stack{
					StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
				},
			},
			describeStacksOutput: cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{{
					StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
				}},
			},
			stmr: mockedStreamer{},
		},
	}

	for name, test := range tests {
		svc := mockedCloudFormationAPI{
			waitUntilStackCreateCompleteErr: test.waitUntilStackCreateCompleteErr,
			waitUntilStackUpdateCompleteErr: test.waitUntilStackUpdateCompleteErr,
			describeStacksOutput:            test.describeStacksOutput,
		}

		t.Run(name, func(t *testing.T) {
			d := deployer.New(svc, logrus.New())
			res := d.WaitForExecute(test.stackName, test.changeSet, test.stmr)

			if res.Err != nil {
				assert.EqualError(t, test.stackRecord.Err, res.Err.Error())
				assert.Equal(t, test.stackRecord.Stack, res.Stack)
			}
		})
	}
}
