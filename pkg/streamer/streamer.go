package streamer

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/b-b3rn4rd/cfn/pkg/writer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type StackEvents map[string]*cloudformation.StackEvent

type Streamer struct {
	svc    cloudformationiface.CloudFormationAPI
	logger *logrus.Logger
}

type StackEventsRecord struct {
	Records StackEvents
	Err     error
}

type Streameriface interface {
	StartStreaming(*string, StackEvents, *writer.StringWriter, <-chan bool) error
	DescribeStackEvents(*string, StackEvents) (stackEvents *StackEventsRecord)
}

func New(svc cloudformationiface.CloudFormationAPI, logger *logrus.Logger) *Streamer {
	return &Streamer{
		svc:    svc,
		logger: logger,
	}
}

func (s *Streamer) StartStreaming(stackName *string, seenEvents StackEvents, wr *writer.StringWriter, done <-chan bool) error {
	s.logger.WithField("stackName", *stackName).Debug("Start streaming stack events")
	s.logger.WithField("stackName", *stackName).Debug(fmt.Sprintf("Stack has %d existing events", len(seenEvents)))

	ch := make(chan *StackEventsRecord, 1)

	ticker := time.NewTicker(time.Second * 1)
	isStackReady := false
	isLastPoll := false

	for {
		select {
		case <-done:
			isStackReady = true
			s.logger.WithField("stackName", *stackName).Debug("Stack creation/update has finished")
		case r := <-ch:
			if r.Err != nil {
				return errors.Wrap(r.Err, "AWS error while running DescribeStackEvents")
			}

			for _, e := range r.Records {
				wr.Write(e)
				seenEvents[*e.EventId] = e
			}

			if isLastPoll {
				s.logger.WithField("stackName", *stackName).Debug("Stack is ready and the last poll has finished")
				return nil
			}

			if isStackReady {
				isLastPoll = true
				s.logger.WithField("stackName", *stackName).Debug("Stack is ready, doing the last poll")
			}

		case <-ticker.C:
			s.logger.WithField("stackName", *stackName).Debug("Polling for new stack events")
			go func() {
				ch <- s.DescribeStackEvents(stackName, seenEvents)
			}()
			ticker = time.NewTicker(time.Second * 15)
		}
	}
}

func (s *Streamer) DescribeStackEvents(stackName *string, seenEvents StackEvents) (stackEvents *StackEventsRecord) {
	stackEvents = &StackEventsRecord{Records: StackEvents{}}

	err := s.svc.DescribeStackEventsPages(&cloudformation.DescribeStackEventsInput{
		StackName: stackName,
	}, func(page *cloudformation.DescribeStackEventsOutput, isLastPage bool) bool {
		for _, stackEvent := range page.StackEvents {
			if len(seenEvents) == 0 {
				stackEvents.Records[*stackEvent.EventId] = stackEvent
			} else {
				if _, exists := seenEvents[*stackEvent.EventId]; !exists {
					stackEvents.Records[*stackEvent.EventId] = stackEvent
				}
			}
		}
		return !isLastPage
	})

	if err != nil {
		stackEvents.Err = errors.Wrap(err, "AWS error while running DescribeStackEvents")
		return
	}

	return
}
