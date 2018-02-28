package streamer

//import (
//	"github.com/aws/aws-sdk-go/service/cloudformation"
//	"github.com/sirupsen/logrus"
//	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
//	"github.com/aws/aws-sdk-go/aws"
//	"strings"
//	"fmt"
//	"sync"
//	"time"
//	"github.com/pkg/errors"
//)
//
//type SteamerRecord struct {
//	stack *cloudformation.Stack
//	stackEvents []*cloudformation.StackEvent
//	Err error
//}
//
//type Streamer struct {
//	svc    cloudformationiface.CloudFormationAPI
//	logger *logrus.Logger
//}
//
//
//func (s *Streamer) streamStackEvents(stackName string) (*SteamerRecord, error){
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
//
//func (s *Streamer) describeStackEvents(stackName string, wg *sync.WaitGroup, ch chan<- *SteamerRecord, seenEvents map[string]*cloudformation.StackEvent) {
//	defer wg.Done()
//
//	stackEvents := map[string]*cloudformation.StackEvent{}
//
//	err := s.svc.DescribeStackEventsPages(&cloudformation.DescribeStackEventsInput{
//		StackName: aws.String(stackName),
//	}, func(page *cloudformation.DescribeStackEventsOutput, isLastPage bool) bool {
//		for _, stackEvent := range page.StackEvents {
//			if len(seenEvents) == 0 {
//				stackEvents[*stackEvent.EventId] = stackEvent
//			} else {
//				if _, exists := seenEvents[*stackEvent.EventId]; !exists {
//					stackEvents[*stackEvent.EventId] = stackEvent
//				}
//			}
//		}
//		return !isLastPage
//	})
//
//	if err != nil {
//		ch <- &StackEventsEntry{Err:err}
//		return
//	}
//
//	ch <- &StackEventsEntry{Records:stackEvents}
//}
//
//func (s *Streamer) describeStack(stackName string, wg *sync.WaitGroup, ch chan<-  *SteamerRecord) {
//	defer wg.Done()
//
//	resp, err := s.svc.DescribeStacks(&cloudformation.DescribeStacksInput{
//		StackName: &stackName,
//	})
//
//	if err != nil {
//		if strings.Contains(err.Error(), fmt.Sprintf("Stack with id %s does not exist", stackName)) {
//			ch <- &StackEntry{}
//			s.logger.WithField("stackName", stackName).Debug("Stack does not exist")
//		} else {
//			ch <- &StackEntry{Err: err}
//		}
//
//		return
//	}
//
//	if len(resp.Stacks) == 0 {
//		ch <- &StackEntry{}
//		s.logger.WithField("stackName", stackName).Debug("Stack does not exist")
//
//		return
//	}
//
//	ch <- &StackEntry{Record: resp.Stacks[0]}
//}