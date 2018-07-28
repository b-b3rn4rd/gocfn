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

// stringToKeyVal converts string key1=val1 key2="val2 val3" into map
func stringToKeyVal(value string) (map[string]string, error) {
	keyVal := make(map[string]string)

	var r = regexp.MustCompile(`(\w+)=([^\s"']+|"([^"]*)"|'([^']*)')`)

	keysVars := r.FindAllStringSubmatch(value, -1)
	if len(keysVars) == 0 {
		return nil, fmt.Errorf("expected ParameterKey1=ParameterValue1 got '%s'", value)
	}

	for _, kv := range keysVars {
		var parameterValue string

		for i := 2; i < len(kv); i++ {
			if kv[i] != "" {
				parameterValue = kv[i]
			}
		}
		keyVal[kv[1]] = parameterValue
	}

	return keyVal, nil
}

func (h *CFNParametersValue) Set(value string) error {
	keyVars, err := stringToKeyVal(value)
	if err != nil {
		return err
	}

	for k, v := range keyVars {
		*h = append(*h, &cloudformation.Parameter{
			ParameterKey:   aws.String(k),
			ParameterValue: aws.String(v),
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
	keysVars, err := stringToKeyVal(value)
	if err != nil {
		return err
	}

	for k, v := range keysVars {
		*h = append(*h, &cloudformation.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
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
