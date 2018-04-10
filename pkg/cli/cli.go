package cli

import (
	"github.com/aws/aws-sdk-go/service/cloudformation"

	"github.com/alecthomas/kingpin"

	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
)

type CFNParametersValue []*cloudformation.Parameter
type CFNTagsValue []*cloudformation.Tag

func (h *CFNParametersValue) Set(value string) error {
	var r = regexp.MustCompile(`(\w+)=(\w+)`)

	keysVars := r.FindAllStringSubmatch(value, -1)

	if len(keysVars) == 0 {
		return fmt.Errorf("expected ParameterKey1=ParameterValue1 got '%s'", value)
	}

	for _, kv := range keysVars {
		*h = append(*h, &cloudformation.Parameter{
			ParameterKey:   aws.String(kv[1]),
			ParameterValue: aws.String(kv[2]),
		})
	}

	return nil
}

func (h *CFNParametersValue) String() string {
	return ""
}

func CFNParameters(s kingpin.Settings) (target *CFNParametersValue) {
	target = &CFNParametersValue{}
	s.SetValue(target)
	return
}

func (h *CFNTagsValue) Set(value string) error {
	var r = regexp.MustCompile(`(\w+)=(\w+)`)

	keysVars := r.FindAllStringSubmatch(value, -1)

	if len(keysVars) == 0 {
		return fmt.Errorf("expected KEY=VALUE got '%s'", value)
	}

	for _, kv := range keysVars {
		*h = append(*h, &cloudformation.Tag{
			Key:   aws.String(kv[1]),
			Value: aws.String(kv[2]),
		})
	}

	return nil
}

func (h *CFNTagsValue) String() string {
	return ""
}

func CFNTags(s kingpin.Settings) (target *CFNTagsValue) {
	target = &CFNTagsValue{}
	s.SetValue(target)
	return
}
