package streamer_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/b-b3rn4rd/cfn/pkg/streamer"
	"github.com/b-b3rn4rd/cfn/pkg/writer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type mockedCloudFormationAPI struct {
	Resp error
	cloudformationiface.CloudFormationAPI
}

func (m mockedCloudFormationAPI) DescribeStackEventsPages(input *cloudformation.DescribeStackEventsInput, fn func(*cloudformation.DescribeStackEventsOutput, bool) bool) error {
	fn(&cloudformation.DescribeStackEventsOutput{
		StackEvents: []*cloudformation.StackEvent{
			{
				EventId: aws.String("test1"),
			},
			{
				EventId: aws.String("test2"),
			}, {
				EventId: aws.String("test3"),
			},
		},
	}, true)
	return m.Resp
}

func TestDescribeStackEvents(t *testing.T) {
	stackName := aws.String("test")

	tests := map[string]struct {
		Svc        cloudformationiface.CloudFormationAPI
		Err        error
		SeenEvents streamer.StackEvents
		Records    streamer.StackEvents
	}{
		"Describe stack events return all events if seen is empty": {
			Svc: mockedCloudFormationAPI{},
			Err: nil,
			Records: streamer.StackEvents{
				"test1": &cloudformation.StackEvent{
					EventId: aws.String("test1"),
				},
				"test2": &cloudformation.StackEvent{
					EventId: aws.String("test2"),
				},
				"test3": &cloudformation.StackEvent{
					EventId: aws.String("test3"),
				},
			},
		},
		"Describe stack events return only unseen events": {
			Svc: mockedCloudFormationAPI{},
			Err: nil,
			SeenEvents: streamer.StackEvents{
				"test2": &cloudformation.StackEvent{
					EventId: aws.String("test2"),
				},
			},
			Records: streamer.StackEvents{
				"test1": &cloudformation.StackEvent{
					EventId: aws.String("test1"),
				},
				"test3": &cloudformation.StackEvent{
					EventId: aws.String("test3"),
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			s := streamer.New(test.Svc, logrus.New())
			res := s.DescribeStackEvents(stackName, test.SeenEvents)
			assert.Equal(t, test.Records, res.Records)
		})
	}
}

func TestStartStreaming(t *testing.T) {
	stackName := aws.String("test")
	wr := &bytes.Buffer{}
	sw := writer.New(wr, writer.JSONFormatter)
	done := make(chan bool, 1)
	tests := map[string]struct {
		Svc        cloudformationiface.CloudFormationAPI
		SeenEvents streamer.StackEvents
		Records    []*cloudformation.StackEvent
		Err        error
	}{
		"All unseen events are streamed": {
			Svc: mockedCloudFormationAPI{},
			SeenEvents: streamer.StackEvents{
				"test2": &cloudformation.StackEvent{
					EventId: aws.String("test2"),
				},
			},
			Records: []*cloudformation.StackEvent{
				{
					EventId: aws.String("test1"),
				},
				{
					EventId: aws.String("test3"),
				},
			},
		},
		"Returns error if cant describe events": {
			Svc: mockedCloudFormationAPI{
				Resp: errors.New("Error while describe"),
			},
			Err: errors.New("Error while describe"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			s := streamer.New(test.Svc, logrus.New())
			done <- true
			err := s.StartStreaming(stackName, test.SeenEvents, sw, done)

			if err != nil {
				assert.Error(t, test.Err, err)
			} else {
				actual := ""
				for _, r := range test.Records {
					raw, _ := json.MarshalIndent(r, "", "    ")
					actual += string(raw) + "\n"
				}

				assert.Equal(t, actual, wr.String())
			}

		})
	}
}
