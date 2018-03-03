package streamer

import (
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/sirupsen/logrus"
	"sync"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/aws"
	"time"
)

type Streamer struct {
	svc    cloudformationiface.CloudFormationAPI
	logger *logrus.Logger
}

type StreamerRecords struct {
	Records []*cloudformation.StackEvent
	Err error
}

//func (s *Streamer) StreamStackEvents(stackName string) (*SteamerRecord, error){
//	timer := time.NewTimer(60 * time.Minute)
//
//	ch := make(chan *SteamerRecord)
//	done := make(chan bool)
//	var wg sync.WaitGroup
//	wg.Add(2)
//	go s.describeStackEvents(stackName, &wg, ch, nil)
//	go s.describeStack(stackName, &wg, ch)
//
//	go func() {
//		defer close(done)
//		wg.Wait()
//	}()
//
//	select {
//		case <- done:
//			s.logger.WithField("stackName", stackName).Debug("Finished streaming stack information")
//		case <- timer.C:
//			//s.logger.WithField("stackName", stackName).Debug("Timeout streaming stack information")
//			return nil, errors.New("Timeout streaming stack information")
//	}
//}
func (s *Streamer) StartStreaming(done chan bool) {
	ch := make(chan *StreamerRecords)

	for {
		select {
			case <-done:
				return
			case <-ch:
				for r := range ch {
					for e := range r.Records {
						
					}
				}
			}
	}
}
func (s *Streamer) DescribeStackEvents(stackName string, seenEvents map[string]*cloudformation.StackEvent) {
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
