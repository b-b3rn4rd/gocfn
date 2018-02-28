package stack

import (
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/sirupsen/logrus"
	"strings"
	"fmt"
	"io/ioutil"
	"github.com/aws/aws-sdk-go/aws"
	"time"
	"github.com/pkg/errors"
	//"github.com/b-b3rn4rd/cfn/pkg/streamer"
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

type UpdateStackEntry struct {
	Record *cloudformation.UpdateStackOutput
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

func (s *Stack) describeStackEvents(stackName string, ch chan<- *StackEventsEntry, seenEvents map[string]*cloudformation.StackEvent) {
	stackEvents := map[string]*cloudformation.StackEvent{}

	err := s.svc.DescribeStackEventsPages(&cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stackName),
	}, func(page *cloudformation.DescribeStackEventsOutput, isLastPage bool) bool {
			for _, stackEvent := range page.StackEvents {
				if len(seenEvents) == 0 {
					stackEvents[*stackEvent.EventId] = stackEvent
				} else {
					if _, exists := seenEvents[*stackEvent.EventId]; !exists {
						stackEvents[*stackEvent.EventId] = stackEvent
					}
				}
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

func (s *Stack) isStackExists(stackName string) (bool, error, *cloudformation.Stack) {

	s.logger.WithField("stackName", stackName).Debug("Checking if stack exists")
	stackExists := false

	ch := make(chan *StackEntry, 1)
	go s.describeStack(stackName, ch)

	resp := <- ch

	if resp.Err != nil {
		return false, resp.Err, nil
	}

	if resp.Record != nil {
		stackExists = true
	}

	return stackExists, nil, resp.Record
}

func (s *Stack) deleteStack(stackName string) (bool, error) {

	_, err := s.svc.DeleteStack(&cloudformation.DeleteStackInput{
		StackName:&stackName,
	})

	if err != nil {
		return false, err
	}

	ticker := time.NewTicker(time.Second * 10)
	timer := time.NewTimer(60 * time.Minute)
	ch := make(chan *StackEntry, 1)

	for {
		select {
			case res := <- ch:
				if res.Err != nil {
					return false, errors.Wrap(res.Err, "AWS error while running DescribeStack")
				}

				if res.Record == nil {
					s.logger.WithField("stackName", stackName).Debug("Stack has been successfully deleted")
					return true, nil
				}

				switch *res.Record.StackStatus {
					case cloudformation.StackStatusDeleteInProgress:
						s.logger.WithField("stackName", stackName).Debug(fmt.Sprintf("Stack is in %s continue polling", *res.Record.StackStatus))
					default:
						if strings.HasSuffix(*res.Record.StackStatus, "_FAILED") {
							return false, errors.New("AWS error while running DeleteStack")
						}
				}

			case <-ticker.C:
				go s.describeStack(stackName, ch)
			case <-timer.C:
				return false, errors.New("Timeout while polling stack deletion status")
		}
	}

	return true, nil
}


func (s *Stack) RunStack(runStackParameters *RunStackParameters, wait bool, force bool) (bool, error){
	s.logger.WithField("stackName", *runStackParameters.StackName).Debug("Running stack")

	exists, err, stack := s.isStackExists(*runStackParameters.StackName)

	if err != nil {
		return false, errors.Wrap(err, "AWS error while running DescribeStack")
	}

	requireUpdate := false
	seenEvents := map[string]*cloudformation.StackEvent{}

	if exists {
		if *stack.StackStatus == cloudformation.StackStatusCreateFailed ||
			*stack.StackStatus == cloudformation.StackStatusRollbackComplete {
			if force {
				s.logger.WithField("stackName", *runStackParameters.StackName).Debug(fmt.Sprintf("Stack is in %s, and force is specified, deleting stack", *stack.StackStatus))
				_, err := s.deleteStack(*runStackParameters.StackName)

				if err != nil {
					return false, errors.Wrap(err, "Error while running DeleteStack")
				}
			} else {
				return false, errors.New(fmt.Sprintf("Stack is in %s and can't be updated, unless --force is specified", *stack.StackStatus))
			}
		} else {
			requireUpdate = true
		}
	}

	if requireUpdate {
		stackEventsChannel := make(chan *StackEventsEntry, 1)
		go s.describeStackEvents(*runStackParameters.StackName, stackEventsChannel, nil)

		res := <- stackEventsChannel

		if res.Err != nil {
			return false, errors.Wrap(res.Err, "AWS error while running DescribeStackEventsPages")
		}

		s.logger.WithField("stackName", *runStackParameters.StackName).Debug(fmt.Sprintf("Stack has %d existing events", len(res.Records)))

		seenEvents = res.Records

		ch := make(chan *UpdateStackEntry, 1)
		s.updateStack(runStackParameters, stack, ch)

		resp := <-ch

		if resp.Err != nil {
			return false, errors.Wrap(resp.Err, "AWS error while running UpdateStack")
		}

	} else {
		ch := make(chan *CreateStackEntry, 1)
		go s.createStack(runStackParameters, ch)

		resp := <-ch

		if resp.Err != nil {
			return false, errors.Wrap(resp.Err, "AWS error while running CreateStack")
		}

		if resp.Record != nil {
			s.logger.WithField("stackName", *runStackParameters.StackName).Info(
				fmt.Sprintf("Stack creation has been initiated. %s", *resp.Record.StackId))
		}
	}

	if wait {
		return s.wait(*runStackParameters.StackName, seenEvents)
	}

	return true, nil
}

func (s *Stack) wait (stackName string, seenEvents map[string]*cloudformation.StackEvent) (bool, error) {
	ch := make(chan *StackEventsEntry, 1)
	ticker := time.NewTicker(time.Second * 10)
	timer := time.NewTimer(60 * time.Minute)

	ch2 := make(chan *StackEntry, 1)
	done := make(chan bool, 2)

	var (
		ok bool
		err error
	)

LOOP:
	for {
		select {
			case <- done:
				s.logger.WithField("stackName", stackName).Debug("Finished polling stack")

				break LOOP
			case res:= <-ch:

				if res.Err != nil {
					return false, errors.Wrap(res.Err, "AWS error while running DescribeStackEvents")
				}

				for _, stackEvent := range res.Records {

					e := s.logger.WithField("Date", *stackEvent.Timestamp)

					if stackEvent.ResourceStatus != nil {
						e = e.WithField("Status", *stackEvent.ResourceStatus)
					}

					if stackEvent.ResourceType != nil {
						e = e.WithField("Type", *stackEvent.ResourceType)
					}

					if stackEvent.LogicalResourceId != nil {
						e = e.WithField("LogicalID", *stackEvent.LogicalResourceId)
					}

					if stackEvent.ResourceStatusReason != nil {
						e.Info(*stackEvent.ResourceStatusReason)
					} else {
						e.Info("")
					}

					seenEvents[*stackEvent.EventId] = stackEvent
				}

				if len(done) == 1 {
					done <- true
				}

			case res:= <-ch2:

				if res.Err != nil {
					return false, errors.Wrap(res.Err, "AWS error while running DescribeStack")
				}

				switch *res.Record.StackStatus {
					case cloudformation.StackStatusCreateFailed,
						cloudformation.StackStatusRollbackFailed,
						cloudformation.StackStatusUpdateRollbackFailed,
						cloudformation.StackStatusRollbackComplete,
						cloudformation.StackStatusUpdateRollbackComplete:
						err = errors.New(fmt.Sprintf("Stack deployment has failed with state: %s", *res.Record.StackStatus))
						done <- true
						break
					case cloudformation.StackStatusCreateComplete,
						cloudformation.StackStatusUpdateComplete:
						s.logger.WithField("stackName", stackName).Debug(fmt.Sprintf("Stack deployment has been finished with state %s", *res.Record.StackStatus))
						ok = true
						done <- true
						break
					default:
						s.logger.WithField("stackName", stackName).Debug(fmt.Sprintf("Stack status is %s, continue polling", *res.Record.StackStatus))
				}
			case <-ticker.C:
				s.logger.WithField("stackName", stackName).Debug("Polling stack")

				go s.describeStack(stackName, ch2)
				go s.describeStackEvents(stackName, ch, seenEvents)
			case <-timer.C:
				return false, errors.New("Timeout while polling stack status")
		}
	}

	return ok, err
}

func (s *Stack) isStackInProgressOrFailedCreation(stackStatus string) bool {
	if stackStatus == cloudformation.StackStatusCreateFailed {
		return true
	}

	if stackStatus == cloudformation.StackStatusRollbackComplete {
		return true
	}

	return strings.HasSuffix(stackStatus, "_IN_PROGRESS")
}

func (s *Stack) createStack(runStackParameters *RunStackParameters, ch chan<- *CreateStackEntry) {
	s.logger.WithField("stackName", *runStackParameters.StackName).Debug("Creating new stack")

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

func (s *Stack) mergeParameters(runStackParameters *RunStackParameters, stack *cloudformation.Stack) []*cloudformation.Parameter {
	parameters := runStackParameters.Parameters

	isParameterSpecified := func(parameterKey string) bool {
		for _, p := range runStackParameters.Parameters {
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

func (s *Stack) updateStack(runStackParameters *RunStackParameters, stack *cloudformation.Stack, ch chan<- *UpdateStackEntry) {
	s.logger.WithField("stackName", *runStackParameters.StackName).Debug("Updating existing stack")

	parameters := s.mergeParameters(runStackParameters, stack)
	resp, err := s.svc.UpdateStack(&cloudformation.UpdateStackInput{
		StackName: runStackParameters.StackName,
		TemplateBody: runStackParameters.TemplateBody,
		TemplateURL: runStackParameters.TemplateURL,
		Parameters: parameters,
		Tags: runStackParameters.Tags,
	})

	if err != nil {
		ch <- &UpdateStackEntry{Err:err}
		return
	}

	ch <- &UpdateStackEntry{Record: resp}
}