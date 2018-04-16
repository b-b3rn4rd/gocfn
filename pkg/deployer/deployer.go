package deployer

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/b-b3rn4rd/gocfn/pkg/streamer"
	"github.com/b-b3rn4rd/gocfn/pkg/writer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	//"sync"
	//"os"
	"os"
	"sync"

	"github.com/b-b3rn4rd/gocfn/pkg/uploader"
)

// DeployParams parameters required for deploy command
type DeployParams struct {
	S3Uploader           uploader.Uploaderiface
	StackName            string
	TemplateFile         string
	Parameters           []*cloudformation.Parameter
	Capabilities         []string
	NoExecuteChangeset   bool
	RoleArn              string
	NotificationArns     []string
	FailOnEmptyChangeset bool
	Tags                 []*cloudformation.Tag
	ForceDeploy          bool
}

type Deployeriface interface {
	WaitForChangeSet(*string, *string) *ChangeSetRecord
	WaitForExecute(*string, *ChangeSetRecord, streamer.Streameriface) *StackRecord
	ExecuteChangeset(*string, *string) error
	CreateChangeSet(deployParams *DeployParams) *ChangeSetRecord
	DescribeStackUnsafe(stackName *string) *cloudformation.Stack
}

type StackRecord struct {
	Stack *cloudformation.Stack
	Err   error
}

type ChangeSetRecord struct {
	ChangeSet     *cloudformation.DescribeChangeSetOutput
	StackEvents   streamer.StackEvents
	ChangeSetType *string
	Err           error
}

type Deployer struct {
	svc             cloudformationiface.CloudFormationAPI
	logger          *logrus.Logger
	changesetPrefix string
}

func New(svc cloudformationiface.CloudFormationAPI, logger *logrus.Logger) *Deployer {
	return &Deployer{
		svc:             svc,
		logger:          logger,
		changesetPrefix: "cfn-cloudformation-package-deploy",
	}
}

func (s *Deployer) hasStack(stackName *string) (bool, *cloudformation.Stack, error) {
	s.logger.WithField("stackName", *stackName).Debug("Checking if stack exists")

	resp, err := s.svc.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: stackName,
	})

	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("Stack with id %s does not exist", *stackName)) {
			s.logger.WithField("stackName", *stackName).Debug("Stack does not exist")
			return false, nil, nil
		}

		return false, nil, errors.Wrap(err, "AWS error while running DescribeStack")
	}

	if len(resp.Stacks) == 0 {
		s.logger.WithField("stackName", *stackName).Debug("Stack does not exist")

		return false, nil, nil
	}

	if *resp.Stacks[0].StackStatus == cloudformation.StackStatusReviewInProgress {
		s.logger.WithField("stackName", *stackName).Debug(fmt.Sprintf("Stack status is %s, treat like it does not exist", *resp.Stacks[0].StackStatus))

		return false, nil, nil
	}

	s.logger.WithField("stackName", *stackName).Debug(fmt.Sprintf("Stack exist with status %s", *resp.Stacks[0].StackStatus))

	return true, resp.Stacks[0], nil
}

func (s *Deployer) createChangeSet(deployParams *DeployParams) (res *ChangeSetRecord) {

	res = &ChangeSetRecord{
		ChangeSet: &cloudformation.DescribeChangeSetOutput{},
	}

	changesetName := fmt.Sprintf("%s-%s", s.changesetPrefix, strconv.FormatInt(time.Now().Unix(), 10))
	description := fmt.Sprintf("Created by cfn at %s", time.Now().UTC().String())

	changeSetInput := &cloudformation.CreateChangeSetInput{
		ChangeSetName: aws.String(changesetName),
		StackName:     aws.String(deployParams.StackName),
		ChangeSetType: aws.String(cloudformation.ChangeSetTypeCreate),
		Tags:          deployParams.Tags,
		Capabilities:  aws.StringSlice(deployParams.Capabilities),
		Description:   aws.String(description),
	}

	hasStack, stack, err := s.hasStack(aws.String(deployParams.StackName))
	if err != nil {
		res.Err = err
		return
	}

	if hasStack && s.hasFailedCreation(stack.StackStatus) {
		if !deployParams.ForceDeploy {
			res.Err = fmt.Errorf("stack is in %s and can't be updated, unless --force is specified", *stack.StackStatus)
			return
		}

		err := s.deleteStack(aws.String(deployParams.StackName))
		if err != nil {
			res.Err = errors.Wrap(err, "Error while running DeleteStack")
			return
		}

		hasStack = false
	}

	if hasStack {
		changeSetInput.ChangeSetType = aws.String(cloudformation.ChangeSetTypeUpdate)
		deployParams.Parameters = s.mergeParameters(deployParams.Parameters, stack)
	}

	changeSetInput.Parameters = deployParams.Parameters

	if deployParams.S3Uploader != nil {
		s.logger.WithField("stackName", aws.String(deployParams.StackName)).Debug("Bucket is specified trying to upload the template")
		templateURL, err := deployParams.S3Uploader.UploadWithDedup(&deployParams.TemplateFile, "template")

		if err != nil {
			res.Err = err
			return
		}

		s.logger.WithField("stackName", aws.String(deployParams.StackName)).WithField("templateURL", templateURL).Debug("Stack is going to be created from s3 bucket")
		changeSetInput.TemplateURL = aws.String(templateURL)
	} else {
		raw, _ := ioutil.ReadFile(deployParams.TemplateFile)
		changeSetInput.TemplateBody = aws.String(string(raw))
	}

	if len(deployParams.NotificationArns) != 0 {
		changeSetInput.NotificationARNs = aws.StringSlice(deployParams.NotificationArns)
	}

	if deployParams.RoleArn != "" {
		changeSetInput.RoleARN = aws.String(deployParams.RoleArn)
	}

	s.logger.WithField("stackName", aws.String(deployParams.StackName)).Debug("Running CreateChangeSet")
	resp, err := s.svc.CreateChangeSet(changeSetInput)

	if err != nil {
		res.Err = errors.Wrap(err, "AWS error while running CreateChangeSet")
		return
	}

	res.ChangeSetType = changeSetInput.ChangeSetType
	res.ChangeSet.ChangeSetId = resp.Id

	return
}

func (s *Deployer) mergeParameters(parameters []*cloudformation.Parameter, stack *cloudformation.Stack) []*cloudformation.Parameter {
	isParameterSpecified := func(parameterKey string) bool {
		for _, p := range parameters {
			if parameterKey == *p.ParameterKey {
				return true
			}
		}

		return false
	}

	for _, p := range stack.Parameters {
		if !isParameterSpecified(*p.ParameterKey) {
			parameters = append(parameters, &cloudformation.Parameter{
				ParameterKey:     p.ParameterKey,
				UsePreviousValue: aws.Bool(true),
			})
		}
	}

	return parameters
}

// Use following function ONLY when you are 100% confident that the stack exists
func (s *Deployer) DescribeStackUnsafe(stackName *string) *cloudformation.Stack {
	resp, _ := s.svc.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: stackName,
	})

	return resp.Stacks[0]
}

func (s *Deployer) WaitForChangeSet(stackName *string, changeSetID *string) (res *ChangeSetRecord) {
	res = &ChangeSetRecord{
		ChangeSet: &cloudformation.DescribeChangeSetOutput{},
	}

	s.logger.WithField("stackName", *stackName).Debug("Waiting for changeset to finish")

	describeChangeSetInput := &cloudformation.DescribeChangeSetInput{
		StackName:     stackName,
		ChangeSetName: changeSetID,
	}

	err := s.svc.WaitUntilChangeSetCreateComplete(describeChangeSetInput)

	resp, _ := s.svc.DescribeChangeSet(describeChangeSetInput)
	res.ChangeSet = resp

	if err != nil {
		if strings.Contains(*resp.StatusReason, "The submitted information didn't contain changes.") {
			s.logger.WithField("stackName", *stackName).Debug("ChangeSet does not contain changes")
			res.Err = errors.New(*resp.StatusReason)
		} else {
			res.Err = errors.Wrap(err, "AWS error while running WaitUntilChangeSetCreateComplete")
		}
	}

	return
}

func (s *Deployer) WaitForExecute(stackName *string, changeSet *ChangeSetRecord, stmr streamer.Streameriface) (res *StackRecord) {
	var err error

	res = &StackRecord{
		Stack: &cloudformation.Stack{},
	}

	describeStackInput := &cloudformation.DescribeStacksInput{
		StackName: stackName,
	}

	s.logger.WithField("stackName", *stackName).Debug("Waiting for stack to be created/updated")

	var wg sync.WaitGroup
	done := make(chan bool)

	wg.Add(1)

	go func() {
		defer wg.Done()
		if *changeSet.ChangeSetType == cloudformation.ChangeSetTypeCreate {
			err = s.svc.WaitUntilStackCreateComplete(describeStackInput)
		} else {
			err = s.svc.WaitUntilStackUpdateComplete(describeStackInput)
		}
		done <- true
	}()

	if stmr != nil {
		s.logger.WithField("stackName", *stackName).Debug("Stream is enabled, preparing to stream stack events")
		wg.Add(1)
		go func() {
			defer wg.Done()
			wr := writer.New(os.Stderr, writer.JSONFormatter)
			stmr.StartStreaming(stackName, changeSet.StackEvents, wr, done)
		}()
	} else {
		<-done
		s.logger.WithField("stackName", *stackName).Debug("Stack is ready and no streaming is required")
	}

	wg.Wait()

	res.Stack = s.DescribeStackUnsafe(stackName)

	if err != nil {
		res.Err = fmt.Errorf("failed creating/updating stack, status: %s", *res.Stack.StackStatus)
	}

	return
}

func (s *Deployer) ExecuteChangeset(stackName *string, changeSetID *string) error {

	s.logger.WithField("stackName", *stackName).Debug("Running ExecuteChangeSet")

	_, err := s.svc.ExecuteChangeSet(&cloudformation.ExecuteChangeSetInput{
		StackName:     stackName,
		ChangeSetName: changeSetID,
	})

	if err != nil {
		return errors.Wrap(err, "AWS error while running ExecuteChangeSet")
	}

	return nil
}

func (s *Deployer) CreateChangeSet(deployParams *DeployParams) *ChangeSetRecord {
	return s.createChangeSet(deployParams)
}

func (s *Deployer) hasFailedCreation(stackStatus *string) bool {
	return *stackStatus == cloudformation.StackStatusCreateFailed || *stackStatus == cloudformation.StackStatusRollbackComplete
}

func (s *Deployer) deleteStack(stackName *string) error {

	s.logger.WithField("stackName", *stackName).Debug("Deleting stack")

	_, err := s.svc.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: stackName,
	})
	if err != nil {
		return err
	}

	s.logger.WithField("stackName", *stackName).Debug("Waiting for stack to be deleted")

	err = s.svc.WaitUntilStackDeleteComplete(&cloudformation.DescribeStacksInput{
		StackName: stackName,
	})

	return err
}
