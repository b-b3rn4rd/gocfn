package deployer

import (
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/sirupsen/logrus"
	"github.com/b-b3rn4rd/cfn/pkg/uploader"
	"github.com/pkg/errors"
	"strings"
	"fmt"
	"time"
	"strconv"
	"github.com/aws/aws-sdk-go/aws"
	"io/ioutil"
)

type Deployer struct {
	svc cloudformationiface.CloudFormationAPI
	logger *logrus.Logger
	changesetPrefix string
}

func New(svc cloudformationiface.CloudFormationAPI, logger *logrus.Logger) *Deployer {
	return &Deployer{
		svc:svc,
		logger:logger,
		changesetPrefix: "gocfn-cloudformation-package-deploy-",
	}
}

func (s *Deployer) hasStack(stackName *string) (bool, error, *cloudformation.Stack) {
	s.logger.WithField("stackName", *stackName).Debug("Checking if stack exists")

	resp, err := s.svc.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: stackName,
	})

	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("Stack with id %s does not exist", *stackName)) {
			s.logger.WithField("stackName", *stackName).Debug("Stack does not exist")
			return false, nil, nil
		} else {
			return false, errors.Wrap(err, "AWS error while running DescribeStack"), nil
		}
	}

	if len(resp.Stacks) == 0 {
		s.logger.WithField("stackName", *stackName).Debug("Stack does not exist")

		return false, nil, nil
	}

	if *resp.Stacks[0].StackStatus == cloudformation.StackStatusReviewInProgress {
		s.logger.WithField("stackName", *stackName).Debug(fmt.Sprintf("Stack status is %s, treat like it does not exist", *resp.Stacks[0].StackStatus))

		return false, nil, nil
	}

	s.logger.WithField("stackName", *stackName).Debug("Stack exist")

	return true, nil, resp.Stacks[0]
}

func (s *Deployer) createChangeSet(stackName *string, templateFile *string, parameters []*cloudformation.Parameter, capabilities []*string, roleArn *string, notificationArns []*string, tags []*cloudformation.Tag, forceDeploy *bool, s3Uploader *uploader.Uploader) (string, string, error) {
	t := time.Now()
	changesetName := fmt.Sprintf("%s-%s", s.changesetPrefix, strconv.FormatInt(t.Unix(), 10))
	description := fmt.Sprintf("Created by gocfn at %s", t.UTC().String())

	changeSetInput := &cloudformation.CreateChangeSetInput{
		ChangeSetName: aws.String(changesetName),
		StackName: stackName,
		ChangeSetType: aws.String(cloudformation.ChangeSetTypeCreate),
		Tags: tags,
		Capabilities: capabilities,
		Description: aws.String(description),
	}


	hasStack, err, stack := s.hasStack(stackName)

	if err != nil {
		return "", "", err
	}

	if hasStack && s.hasFailedCreation(stack.StackStatus) {
		if !*forceDeploy {
			return "", "", errors.New(fmt.Sprintf("Stack is in %s and can't be updated, unless --force is specified", *stack.StackStatus))
		}

		_, err := s.deleteStack(stackName)

		if err != nil {
			return "", "", errors.Wrap(err, "Error while running DeleteStack")
		}

		hasStack = false
	}

	if hasStack {
		changeSetInput.ChangeSetType = aws.String(cloudformation.ChangeSetTypeUpdate)
		parameters = s.mergeParameters(parameters, stack)
	}

	changeSetInput.Parameters = parameters

	if s3Uploader != nil {
		templateUrl, err := s3Uploader.UploadWithDedup(templateFile, "template")

		if err != nil {
			return "", "", err
		}

		changeSetInput.TemplateURL = aws.String(templateUrl)
	} else {
		raw, _ := ioutil.ReadFile(*templateFile)
		changeSetInput.TemplateBody = aws.String(string(raw))
	}

	if len(notificationArns) != 0 {
		changeSetInput.NotificationARNs = notificationArns
	}

	if *roleArn != "" {
		changeSetInput.RoleARN = roleArn
	}

	s.logger.WithField("stackName", *stackName).Debug("Running CreateChangeSet")
	resp, err := s.svc.CreateChangeSet(changeSetInput)

	if err != nil {
		return "", "", errors.Wrap(err, "AWS error while running CreateChangeSet")
	}

	return  *resp.Id, *changeSetInput.ChangeSetType, nil
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
				ParameterKey: p.ParameterKey,
				UsePreviousValue: aws.Bool(true),
			})
		}
	}

	return parameters
}

func (s *Deployer) waitForChangeset(stackName *string, changeSetId *string) error {
	s.logger.WithField("stackName", *stackName).Debug("Waiting for changeset to finish")

	describeChangeSetInput := &cloudformation.DescribeChangeSetInput{
		StackName: stackName,
		ChangeSetName: changeSetId,
	}

	err := s.svc.WaitUntilChangeSetCreateComplete(describeChangeSetInput)

	if err != nil {
		resp, _ := s.svc.DescribeChangeSet(describeChangeSetInput)
		if strings.Contains(*resp.StatusReason, "The submitted information didn't contain changes.") {
			return errors.New(*resp.StatusReason)
		}

		return errors.Wrap(err, "AWS error while running WaitUntilChangeSetCreateComplete")
	}

	return nil
}

func (s *Deployer) waitForExecute(stackName *string, changeSetType *string) error {
	var err error

	describeStackInput := &cloudformation.DescribeStacksInput{
		StackName: stackName,
	}

	s.logger.WithField("stackName", *stackName).Debug("Waiting for stack to be created/updated")

	if *changeSetType == cloudformation.ChangeSetTypeCreate {
		err = s.svc.WaitUntilStackCreateComplete(describeStackInput)
	} else {
		err = s.svc.WaitUntilStackUpdateComplete(describeStackInput)
	}


	return err
}

func (s *Deployer) ExecuteAndWaitForChangeset(stackName *string, changeSetId *string, changeSetType *string) error {

	s.logger.WithField("stackName", *stackName).Debug("Running ExecuteChangeSet")

	_, err := s.svc.ExecuteChangeSet(&cloudformation.ExecuteChangeSetInput{
		StackName: stackName,
		ChangeSetName: changeSetId,
	})

	if err != nil {
		return errors.Wrap(err, "AWS error while running ExecuteChangeSet")
	}

	err = s.waitForExecute(stackName, changeSetType)

	if err != nil {
		return errors.Wrap(err, "AWS error while running waiting for stack to finish")
	}

	return nil
}

func (s *Deployer) CreateAndWaitForChangeset(stackName *string, templateFile *string, parameters []*cloudformation.Parameter, capabilities []*string, noExecuteChangeset *bool, roleArn *string, notificationArns []*string, tags []*cloudformation.Tag, forceDeploy *bool, s3Uploader *uploader.Uploader) (string, string, error) {
	changeSetId, changeSetType, err := s.createChangeSet(stackName, templateFile, parameters, capabilities, roleArn, notificationArns, tags, forceDeploy, s3Uploader)

	if err != nil {
		return changeSetId, changeSetType, err
	}

	err = s.waitForChangeset(stackName, aws.String(changeSetId))

	if err != nil {
		return changeSetType, changeSetType, err
	}

	return changeSetId, changeSetType, nil
}

func (s *Deployer) hasFailedCreation(stackStatus *string) bool {
	return *stackStatus == cloudformation.StackStatusCreateFailed || *stackStatus == cloudformation.StackStatusRollbackComplete
}

func (s *Deployer) deleteStack(stackName *string) (bool, error) {

	s.logger.WithField("stackName", *stackName).Debug("Deleting stack")

	_, err := s.svc.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: stackName,
	})

	if err != nil {
		return false, err
	}

	s.logger.WithField("stackName", *stackName).Debug("Waiting for stack to be deleted")

	err = s.svc.WaitUntilStackDeleteComplete(&cloudformation.DescribeStacksInput{
		StackName: stackName,
	})

	if err != nil {
		return false, err
	}

	return true, nil
}