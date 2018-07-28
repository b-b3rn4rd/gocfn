package cli_test

import (
	"testing"

	"fmt"

	"github.com/alecthomas/kingpin"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/b-b3rn4rd/gocfn/pkg/cli"
	"github.com/stretchr/testify/assert"
)

func TestCFNParametersValue(t *testing.T) {
	tests := map[string]struct {
		passedParams         string
		expectedParsedParams cli.CFNParametersValue
	}{
		"cheeky params are parsed as expected": {
			passedParams: "Email=firstname.lastname@example.com.au Name='John Doh'",
			expectedParsedParams: []*cloudformation.Parameter{
				{
					ParameterKey:   aws.String("Email"),
					ParameterValue: aws.String("firstname.lastname@example.com.au"),
				},
				{
					ParameterKey:   aws.String("Name"),
					ParameterValue: aws.String("John Doh"),
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			a := kingpin.New("test", "").Terminate(nil)
			p := cli.CFNParameters(a.Flag("parameters", ""))
			a.Parse([]string{fmt.Sprintf("--parameters=%s", test.passedParams)})
			assert.Equal(t, &test.expectedParsedParams, p)
		})
	}
}

func TestCFNTagsValue(t *testing.T) {
	tests := map[string]struct {
		passedTags         string
		expectedParsedTags cli.CFNTagsValue
	}{
		"cheeky tags are also handled well": {
			passedTags: "ServerIP=127.0.0.1:80 Name='Web App'",
			expectedParsedTags: []*cloudformation.Tag{
				{
					Key:   aws.String("ServerIP"),
					Value: aws.String("127.0.0.1:80"),
				},
				{
					Key:   aws.String("Name"),
					Value: aws.String("Web App"),
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			a := kingpin.New("test", "").Terminate(nil)
			p := cli.CFNTags(a.Flag("tags", ""))
			a.Parse([]string{fmt.Sprintf("--tags=%s", test.passedTags)})
			assert.Equal(t, &test.expectedParsedTags, p)
		})
	}
}
