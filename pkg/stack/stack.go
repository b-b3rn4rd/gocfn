package stack

import (
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/sirupsen/logrus"
	"strings"
	"fmt"
)

type Stack struct {
	svc    cloudformation.CloudFormation
	logger *logrus.Logger
}

type StackEntry struct {
	Record *cloudformation.Stack
	Err     error
}

func New(logger *logrus.Logger, svc cloudformation.CloudFormation) *Stack {
	return &Stack{
		logger:logger,
		svc: svc,
	}
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
			s.logger.WithError(err).Fatal("AWS error while running DescribeStack")
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

func (s *Stack) RunStack(stackName string) {
	s.logger.WithField("stackName", stackName).Debug("Running stack")

	exists, err := s.isStackExists(stackName)

	if err != nil {

	}

	if exists {
		s.updateStack(stackName)
	} else {
		s.createStack(stackName)
	}

}

func (s *Stack) createStack(stackName string) {
	s.svc.CreateStack(&cloudformation.CreateStackInput{
		StackName: &stackName,
	})
}

func (s *Stack) updateStack(stackName string) {

}