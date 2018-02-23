package stack

import (
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/sirupsen/logrus"
	"strings"
	"fmt"
	"io/ioutil"
	"github.com/aws/aws-sdk-go/aws"
)

type Stack struct {
	svc    cloudformation.CloudFormation
	logger *logrus.Logger
}

type StackEntry struct {
	Record *cloudformation.Stack
	Err     error
}

type CreateStackEntry struct {
	Record *cloudformation.CreateStackOutput
	Err error
}

type StackEventsEntry struct {
	Records map[string]*cloudformation.StackEvent
	Err error
}

type RunStackParameters struct {
	Parameters []*cloudformation.Parameter
	Tags []*cloudformation.Tag
	StackName *string
	TemplateBody *string
	TemplateURL *string
	Capabilities []*string
}

func NewRunStackParameters (stackName *string,
	parameters []*cloudformation.Parameter,
	tags []*cloudformation.Tag,
	templateBody *string,
	templateURL *string,
	capabilities []*string) *RunStackParameters {

	runStackParameters := &RunStackParameters{
		StackName: stackName,
	}


	if len(parameters) != 0 {
		runStackParameters.Parameters = parameters
	}


	if len(tags) != 0 {
		runStackParameters.Tags = tags
	}


	if *templateBody != "" {
		templateBodyBytes, _ := ioutil.ReadFile(*templateBody)
		templateBody := string(templateBodyBytes)
		runStackParameters.TemplateBody = &templateBody
	}

	if *templateURL != "" {
		runStackParameters.TemplateURL = templateURL
	}

	if len(capabilities) != 0 {
		runStackParameters.Capabilities = capabilities
	}

	return runStackParameters
}

func New(logger *logrus.Logger, svc cloudformation.CloudFormation) *Stack {
	return &Stack{
		logger:logger,
		svc: svc,
	}
}

func (s *Stack) describeStackEvents(stackName string, ch chan<- *StackEventsEntry) {
	stackEvents := map[string]*cloudformation.StackEvent{}

	err := s.svc.DescribeStackEventsPages(&cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stackName),
	}, func(page *cloudformation.DescribeStackEventsOutput, isLastPage bool) bool {
			for _, stackEvent := range page.StackEvents {
				stackEvents[*stackEvent.EventId] = stackEvent
			}
		return !isLastPage
	})

	if err != nil {
		ch <- &StackEventsEntry{Err:err}
		return
	}

	ch <- &StackEventsEntry{Records:stackEvents}
}

func (s *Stack) describeStack(stackName string, ch chan<-  *StackEntry) {
	resp, err := s.svc.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: &stackName,
	})

	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("Stack with id %s does not exist", stackName)) {
			ch <- &StackEntry{}
			s.logger.WithField("stackName", stackName).Debug("Stack does not exist")
		} else {
			ch <- &StackEntry{Err: err}
		}

		return
	}

	if len(resp.Stacks) == 0 {
		ch <- &StackEntry{}
		s.logger.WithField("stackName", stackName).Debug("Stack does not exist")

		return
	}

	ch <- &StackEntry{Record: resp.Stacks[0]}
}

func (s *Stack) isStackExists(stackName string) (bool, error) {

	s.logger.WithField("stackName", stackName).Debug("Checking if stack exists")
	stackExists := false

	ch := make(chan *StackEntry, 1)
	go s.describeStack(stackName, ch)

	stack := <- ch

	if stack.Err != nil {
		return false, stack.Err
	}

	if stack.Record != nil {
		stackExists = true
	}

	return stackExists, nil
}

func (s *Stack) RunStack(runStackParameters *RunStackParameters) {
	s.logger.WithField("stackName", runStackParameters.StackName).Debug("Running stack")

	exists, err := s.isStackExists(*runStackParameters.StackName)

	if err != nil {
		s.logger.WithError(err).Fatal("AWS error while running DescribeStack")
	}

	if exists {
		s.updateStack(runStackParameters)
	} else {
		ch := make(chan *CreateStackEntry, 1)

		go s.createStack(runStackParameters, ch)

		resp := <-ch
		if resp.Err != nil {
			s.logger.WithField("stackName", runStackParameters.StackName).WithError(resp.Err).Fatal("AWS error while running CreateStack")
		}

		if resp.Record != nil {
			s.logger.WithField("stackName", runStackParameters.StackName).Info(
				fmt.Sprintf("Stack creation has been initiated. %s", *resp.Record.StackId))
		}
	}
}

func (s *Stack) wait (stackName string) {
	ch := make(chan *StackEventsEntry, 1)
	go s.describeStackEvents(stackName, ch)

	resp := <- ch

	if resp.Err != nil {
		s.logger.WithError(resp.Err).Fatal("AWS error while running DescribeStackEventsPages")
	}

	s.logger.WithField("stackName", stackName).Debug(fmt.Sprintf("Stack has %d events", len(resp.Records)))
}

func (s *Stack) createStack(runStackParameters *RunStackParameters, ch chan<- *CreateStackEntry) {
	s.logger.WithField("stackName", runStackParameters.StackName).Debug("Creating new stack")

	resp, err := s.svc.CreateStack(&cloudformation.CreateStackInput{
		StackName: runStackParameters.StackName,
		TemplateBody: runStackParameters.TemplateBody,
		TemplateURL: runStackParameters.TemplateURL,
		Parameters: runStackParameters.Parameters,
		Tags: runStackParameters.Tags,
	})

	if err != nil {
		ch <- &CreateStackEntry{Err:err}
		return
	}

	ch <- &CreateStackEntry{Record: resp}

}

func (s *Stack) updateStack(runStackParameters *RunStackParameters) {

}