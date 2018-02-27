package streamer

import (
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/aws"
	"strings"
	"fmt"
	"sync"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/sirupsen/logrus"
	"github.com/pkg/errors"
	"time"
)

type Streamer struct {
	svc cloudformationiface.CloudFormationAPI
	logger *logrus.Logger
}

type describeStackEntry struct {
	stack *cloudformation.Stack
	err error
}

type describeStackEventsEntry struct {
	stackEvents map[string]*cloudformation.StackEvent
	err error
}

type StreamerEntry struct {
	Stack *cloudformation.Stack
	StackEvents map[string]*cloudformation.StackEvent
	Err error
}

func (s *Streamer) streamStackEvents(stackName string) *StreamerEntry {
	var wg sync.WaitGroup
	wg.Add(2)

	streamerEntry := &StreamerEntry{}
	describeStackEntry := &describeStackEntry{}
	describeStackEventsEntry := &describeStackEventsEntry{}
	done := make(chan bool, 1)
	timer := time.NewTimer(60 * time.Minute)
	go s.describeStack(stackName, &wg, describeStackEntry)
	go s.describeStackEvents(stackName, &wg, describeStackEventsEntry, nil)

	go func () {
		defer close(done)
		wg.Wait()
	}()

	if describeStackEntry.err != nil {
		streamerEntry.Err = errors.Wrap(describeStackEntry.err, "AWS error while running DescribeStack")
		return streamerEntry
	}

	if describeStackEventsEntry.err != nil {
		streamerEntry.Err = errors.Wrap(describeStackEntry.err, "AWS error while running DescribeStackEvents")
		return streamerEntry
	}

	streamerEntry.Stack = describeStackEntry.stack
	streamerEntry.StackEvents = describeStackEventsEntry.stackEvents

	select {
		case <-done:
			s.logger.WithField("stackName", stackName).Debug("Finished executing goroutines")
		case <-timer.C:
			s.logger.WithField("stackName", stackName).Debug("Timeout executing goroutines")
	}
	
	return streamerEntry
}

func (s *Streamer) describeStack(stackName string, wg *sync.WaitGroup, entry *describeStackEntry) {
	defer wg.Done()

	resp, err := s.svc.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: &stackName,
	})

	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("Stack with id %s does not exist", stackName)) {
			entry.stack = nil
			s.logger.WithField("stackName", stackName).Debug("Stack does not exist")
		} else {
			entry.err = err
		}

		return
	}

	if len(resp.Stacks) == 0 {
		entry.stack = &cloudformation.Stack{}
		s.logger.WithField("stackName", stackName).Debug("Stack does not exist")

		return
	}

	entry.stack = resp.Stacks[0]
}

func (s *Streamer) describeStackEvents(stackName string, wg *sync.WaitGroup, entry *describeStackEventsEntry, seenEvents map[string]*cloudformation.StackEvent) {
	defer wg.Done()

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
		entry.err = err
		return
	}

	entry.stackEvents = stackEvents
}