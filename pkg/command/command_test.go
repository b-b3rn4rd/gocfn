package command_test

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/b-b3rn4rd/cfn/pkg/command"
	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	r, _ := json.MarshalIndent(command.DeployParams{
		StackName:    aws.String("hello"),
		TemplateFile: aws.String("example.yml"),
	}, "", "    ")

	tests := map[string]struct {
		expectedResp string
	}{
		"String returns pretty params": {
			expectedResp: string(r),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			res := command.DeployParams{
				StackName:    aws.String("hello"),
				TemplateFile: aws.String("example.yml"),
			}

			assert.Equal(t, test.expectedResp, res.String())
		})
	}
}
