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
	cloudformationiface.CloudFormationAPI
}

func (m mockedCloudFormationAPI) DescribeStacks(input *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
	return &m.describeStacksOutput, m.describeStacksErr
}

func (m mockedCloudFormationAPI) CreateChangeSet(input *cloudformation.CreateChangeSetInput) (*cloudformation.CreateChangeSetOutput, error) {
	return &m.createChangeSetOutput, m.createChangeSetErr
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
	}

	for name, test := range tests {
		svc := mockedCloudFormationAPI{
			createChangeSetOutput: test.createChangeSetOutput,
			createChangeSetErr: test.createChangeSetErr,
			describeStacksOutput: test.describeStacksOutput,
			describeStacksErr: test.describeStacksErr,
		}

		t.Run(name, func(t *testing.T) {
			d := deployer.New(svc, logrus.New())
			resp := d.CreateChangeSet(test.stackName, test.templateFile, test.parameters, test.capabilities, test.noExecuteChangeset, test.roleArn, test.notificationArns, test.tags, test.forceDeploy, test.s3Uploader)

			assert.EqualError(t, test.changeSetRecord.Err, resp.Err.Error())
			assert.Equal(t, test.changeSetRecord.StackEvents, resp.StackEvents)
			assert.Equal(t, test.changeSetRecord.ChangeSetType, resp.ChangeSetType)
			assert.Equal(t, test.changeSetRecord.ChangeSet, resp.ChangeSet)
		})
	}
}