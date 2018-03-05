package streamer

import (
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/sirupsen/logrus"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"time"
	"github.com/b-b3rn4rd/cfn/pkg/writer"
	"github.com/pkg/errors"
	"fmt"
)

type StackEvents map[string]*cloudformation.StackEvent

type Streamer struct {
	svc    cloudformationiface.CloudFormationAPI
	logger *logrus.Logger
}

type StackEventsRecord struct {
	Records StackEvents
	Err error
}

func New(svc cloudformationiface.CloudFormationAPI, logger *logrus.Logger) *Streamer {
	return &Streamer{
		svc:svc,
		logger:logger,
	}
}

func (s *Streamer) StartStreaming(stackName *string, wr *writer.StringWriter, done <-chan bool) error {
	s.logger.WithField("stackName", *stackName).Debug("Start streaming stack events")

	ch := make(chan *StreamerRecords, 1)
	ticker := time.NewTicker(time.Second * 15)
	seenEvents := map[string]*cloudformation.StackEvent{}
	stackFinished := false

	s.logger.WithField("stackName", *stackName).Debug("Gather existing stack events")
	go s.DescribeStackEvents(stackName, ch, seenEvents)

	res := <- ch

	if res.Err != nil {
		return errors.Wrap(res.Err, "AWS error while running DescribeStackEvents")
	}

	seenEvents = res.Records
	s.logger.WithField("stackName", *stackName).Debug(fmt.Sprintf("Stack has %d existing events", len(seenEvents)))

	go s.DescribeStackEvents(stackName, ch, seenEvents)

	for {
		select {
			case <-done:
				stackFinished = true
				s.logger.WithField("stackName", *stackName).Debug("Stack creation/update has finished")
			case r := <-ch:
				s.logger.WithField("stackName", *stackName).Debug("Received stack events")

				for _, e := range r.Records {
					fmt.Println(e.GoString())
				}

				s.logger.WithField("stackName", *stackName).Debug("Stack events have been printed")
				if stackFinished {
					s.logger.WithField("stackName", *stackName).Debug("Stack events have been streamed and stack is ready")
					return nil
				}

			case <- ticker.C:
				s.logger.WithField("stackName", *stackName).Debug("Polling for new stack events")
				go s.DescribeStackEvents(stackName, ch, seenEvents)
			}
	}

	return nil
}

func (s *Streamer) DescribeStackEvents(stackName *string, seenEvents StackEvents) (stackEvents *StackEventsRecord) {
	stackEvents = &StackEventsRecord{Records:StackEvents{}}

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
