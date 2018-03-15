package deployer_test

import (
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"testing"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/b-b3rn4rd/cfn/pkg/deployer"
	"github.com/sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/b-b3rn4rd/cfn/pkg/uploader"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type mockedCloudFormationAPI struct {
	describeStacksOutput cloudformation.DescribeStacksOutput
	describeStacksErr error
	createChangeSetOutput cloudformation.CreateChangeSetOutput
	createChangeSetErr error
	deleteStackOutput cloudformation.DeleteStackOutput
	deleteStackErr error
	waitUntilStackDeleteCompleteErr error
	waitUntilChangeSetCreateCompleteErr error
	describeChangeSetOutput cloudformation.DescribeChangeSetOutput
	describeChangeSetErr error
	executeChangeSetOutput cloudformation.ExecuteChangeSetOutput
	executeChangeSetErr error
	cloudformationiface.CloudFormationAPI
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

func TestCreateChangeSet(t *testing.T) {
	tests := map[string]struct{
		stackName *string
		templateFile *string
		parameters []*cloudformation.Parameter
		capabilities []*string
		noExecuteChangeset *bool
		roleArn *string
		notificationArns []*string
		tags []*cloudformation.Tag
		forceDeploy *bool
		s3Uploader *uploader.Uploader
		// func
		describeStacksOutput cloudformation.DescribeStacksOutput
		describeStacksErr error
		createChangeSetOutput cloudformation.CreateChangeSetOutput
		createChangeSetErr error
		cloudformationiface.CloudFormationAPI
		deleteStackOutput cloudformation.DeleteStackOutput
		deleteStackErr error
		// resp
		changeSetRecord *deployer.ChangeSetRecord
	} {
		"CreateChangeSet returns error if cant describe stack": {
			stackName: aws.String("hello"),
			templateFile: aws.String("template.yml"),
			parameters: []*cloudformation.Parameter{},
			capabilities: []*string{},
			noExecuteChangeset: aws.Bool(false),
			notificationArns: []*string{},
			tags: []*cloudformation.Tag{},
			forceDeploy: aws.Bool(false),
			describeStacksErr: errors.New("cant describe stack error"),
			changeSetRecord: &deployer.ChangeSetRecord{
				Err: errors.Wrap(errors.New("cant describe stack error"), "AWS error while running DescribeStack"),
				ChangeSet:&cloudformation.DescribeChangeSetOutput{},
			},
		},
		"CreateChangeSet returns error if stack is in created failed and no force is specified": {
			stackName: aws.String("hello"),
			templateFile: aws.String("template.yml"),
			parameters: []*cloudformation.Parameter{},
			capabilities: []*string{},
			noExecuteChangeset: aws.Bool(false),
			roleArn: aws.String(""),
			notificationArns: []*string{},
			tags: []*cloudformation.Tag{},
			forceDeploy: aws.Bool(false),
			describeStacksOutput: cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{{
					StackStatus:aws.String(cloudformation.StackStatusCreateFailed),
				}},
			},
			changeSetRecord: &deployer.ChangeSetRecord{
				Err: errors.New("Stack is in CREATE_FAILED and can't be updated, unless --force is specified"),
				ChangeSet:&cloudformation.DescribeChangeSetOutput{},
			},
		},
		"CreateChangeSet is created if stack is in created failed and force is specified": {
			stackName: aws.String("hello"),
			templateFile: aws.String("template.yml"),
			parameters: []*cloudformation.Parameter{},
			capabilities: []*string{},
			noExecuteChangeset: aws.Bool(false),
			roleArn: aws.String(""),
			notificationArns: []*string{},
			tags: []*cloudformation.Tag{},
			forceDeploy: aws.Bool(true),
			deleteStackOutput: cloudformation.DeleteStackOutput{

			},
			describeStacksOutput: cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{{
					StackStatus:aws.String(cloudformation.StackStatusCreateFailed),
				}},
			},
			changeSetRecord: &deployer.ChangeSetRecord{
				ChangeSet:&cloudformation.DescribeChangeSetOutput{
					ChangeSetId: aws.String("test"),
				},
				ChangeSetType: aws.String(cloudformation.ChangeSetTypeCreate),
			},
			createChangeSetOutput: cloudformation.CreateChangeSetOutput{
				Id: aws.String("test"),
			},
		},
		"CreateChangeSet is failed if stack deletion has failed": {
			stackName: aws.String("hello"),
			templateFile: aws.String("template.yml"),
			parameters: []*cloudformation.Parameter{},
			capabilities: []*string{},
			noExecuteChangeset: aws.Bool(false),
			roleArn: aws.String(""),
			notificationArns: []*string{},
			tags: []*cloudformation.Tag{},
			forceDeploy: aws.Bool(true),
			deleteStackErr: errors.New("cant delete stack"),
			describeStacksOutput: cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{{
					StackStatus:aws.String(cloudformation.StackStatusCreateFailed),
				}},
			},
			changeSetRecord: &deployer.ChangeSetRecord{
				Err: errors.Wrap(errors.New("cant delete stack"),"Error while running DeleteStack"),
				ChangeSet:&cloudformation.DescribeChangeSetOutput{},
			},
		},
		"CreateChangeSet is updated if stack exists": {
			stackName: aws.String("hello"),
			templateFile: aws.String("template.yml"),
			parameters: []*cloudformation.Parameter{},
			capabilities: []*string{},
			noExecuteChangeset: aws.Bool(false),
			roleArn: aws.String(""),
			notificationArns: []*string{},
			tags: []*cloudformation.Tag{},
			forceDeploy: aws.Bool(true),
			deleteStackOutput: cloudformation.DeleteStackOutput{

			},
			describeStacksOutput: cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{{
					StackStatus:aws.String(cloudformation.StackStatusCreateComplete),
				}},
			},
			changeSetRecord: &deployer.ChangeSetRecord{
				ChangeSet:&cloudformation.DescribeChangeSetOutput{
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
			createChangeSetErr: test.createChangeSetErr,
			describeStacksOutput: test.describeStacksOutput,
			describeStacksErr: test.describeStacksErr,
			deleteStackOutput: test.deleteStackOutput,
			deleteStackErr: test.deleteStackErr,
		}

		t.Run(name, func(t *testing.T) {
			d := deployer.New(svc, logrus.New())
			resp := d.CreateChangeSet(test.stackName, test.templateFile, test.parameters, test.capabilities, test.noExecuteChangeset, test.roleArn, test.notificationArns, test.tags, test.forceDeploy, test.s3Uploader)

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
	tests := map[string]struct{
		stackName *string
		changeSetId *string
		waitUntilChangeSetCreateCompleteErr error
		describeChangeSetOutput cloudformation.DescribeChangeSetOutput
		describeChangeSetErr error
		// resp
		changeSetRecord *deployer.ChangeSetRecord
	} {
		"WaitForChangeSet returns errror if WaitUntilChangeSetCreateComplete has failed with unknown error": {
			stackName: aws.String("test-stack"),
			changeSetId: aws.String("one"),
			waitUntilChangeSetCreateCompleteErr: errors.New("wait error"),
			changeSetRecord:&deployer.ChangeSetRecord{
				Err:errors.Wrap(errors.New("wait error"), "AWS error while running WaitUntilChangeSetCreateComplete"),
				ChangeSet:&cloudformation.DescribeChangeSetOutput{
					StatusReason: aws.String("unknown reason"),
				},
			},
			describeChangeSetOutput: cloudformation.DescribeChangeSetOutput{
				StatusReason: aws.String("unknown reason"),
			},
		},
		"WaitForChangeSet returns custom errror if changeset does not contain changes": {
			stackName: aws.String("test-stack"),
			changeSetId: aws.String("one"),
			waitUntilChangeSetCreateCompleteErr: errors.New("The submitted information didn't contain changes."),
			changeSetRecord:&deployer.ChangeSetRecord{
				Err: errors.New("The submitted information didn't contain changes."),
				ChangeSet:&cloudformation.DescribeChangeSetOutput{
					StatusReason: aws.String("The submitted information didn't contain changes."),
				},
			},
			describeChangeSetOutput: cloudformation.DescribeChangeSetOutput{
				StatusReason: aws.String("The submitted information didn't contain changes."),
			},
		},
		"WaitForChangeSet fills ChangeSet field with DescribeChangeSet information": {
			stackName: aws.String("test-stack"),
			changeSetId: aws.String("one"),
			changeSetRecord:&deployer.ChangeSetRecord{

				ChangeSet:&cloudformation.DescribeChangeSetOutput{
					Status:aws.String("all good"),
					StatusReason:aws.String("no particular reason"),
					ChangeSetId:aws.String("one"),
				},
			},
			describeChangeSetOutput: cloudformation.DescribeChangeSetOutput{
				Status:aws.String("all good"),
				StatusReason:aws.String("no particular reason"),
				ChangeSetId:aws.String("one"),
			},
		},
	}

	for name, test := range tests {
		svc := mockedCloudFormationAPI{
			waitUntilChangeSetCreateCompleteErr: test.waitUntilChangeSetCreateCompleteErr,
			describeChangeSetOutput: test.describeChangeSetOutput,
			describeChangeSetErr: test.describeChangeSetErr,
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
	tests := map[string]struct{

	}{
		{

		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

		})
	}
}