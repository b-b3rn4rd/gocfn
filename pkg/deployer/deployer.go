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
	s.logger.WithField("stackName", stackName).Debug("Checking if stack exists")

	resp, err := s.svc.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: stackName,
	})

	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("Stack with id %s does not exist", stackName)) {
			s.logger.WithField("stackName", stackName).Debug("Stack does not exist")
			return false, nil, nil
		} else {
			return false, errors.Wrap(err, "AWS error while running DescribeStack"), nil
		}
	}

	if len(resp.Stacks) == 0 {
		s.logger.WithField("stackName", stackName).Debug("Stack does not exist")

		return false, nil, nil
	}

	if *resp.Stacks[0].StackStatus == cloudformation.StackStatusReviewInProgress {
		s.logger.WithField("stackName", stackName).Debug(fmt.Sprintf("Stack status is %s, treat like it does not exist", *resp.Stacks[0].StackStatus))

		return false, nil, nil
	}

	s.logger.WithField("stackName", stackName).Debug("Stack exist")

	return true, nil, resp.Stacks[0]
}

func (s *Deployer) createChangeSet(stackName *string, templateFile *string, parameters []*cloudformation.Parameter, capabilities []*string, roleArn *string, notificationArns []*string, tags []*cloudformation.Tag, forceDeploy *bool, s3Uploader *uploader.Uploader) (string, error) {
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
		return "", err
	}

	if hasStack {
		changeSetInput.ChangeSetType = aws.String(cloudformation.ChangeSetTypeUpdate)
		parameters = s.mergeParameters(parameters, stack)
	}

	changeSetInput.Parameters = parameters

	if s3Uploader != nil {
		templateUrl, err := s3Uploader.UploadWithDedup(templateFile, "template")

		if err != nil {
			return "", err
		}

		changeSetInput.TemplateURL = aws.String(templateUrl)
	}

	if notificationArns != nil {
		changeSetInput.NotificationARNs = notificationArns
	}

	if roleArn != nil {
		changeSetInput.RoleARN = roleArn
	}

	resp, err := s.svc.CreateChangeSet(changeSetInput)

	if err != nil {
		return "", errors.Wrap(err, "AWS error while running CreateChangeSet")
	}

	return *resp.Id, nil
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

func (s *Deployer) CreateAndWaitForChangeset(stackName *string, templateFile *string, parameters []*cloudformation.Parameter, capabilities []*string, noExecuteChangeset *bool, roleArn *string, notificationArns []*string, failOnEmptyChangeset *bool, tags []*cloudformation.Tag, forceDeploy *bool, s3Uploader *uploader.Uploader) error {
	changeSetId, err := s.createChangeSet(stackName, templateFile, parameters, capabilities, roleArn, notificationArns, tags, forceDeploy, s3Uploader)

	if err != nil {
		return err
	}

	fmt.Println(changeSetId)
}