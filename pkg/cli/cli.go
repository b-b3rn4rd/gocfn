package cli

import (
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"encoding/json"
	"github.com/alecthomas/kingpin"
	"os"
	"io/ioutil"
	"fmt"
	"regexp"
	"github.com/aws/aws-sdk-go/aws"
)

type CFNParametersValue []*cloudformation.Parameter
type CFNTagsValue []*cloudformation.Tag
type KeyValValue map[string]string

func (h *CFNParametersValue) Set(value string) (error) {
	if _, err := os.Stat(value); err == nil {
		raw, err := ioutil.ReadFile(value)

		if err != nil {
			return err
		}

		return json.Unmarshal(raw, &h)

	} else {
		return json.Unmarshal([]byte(value), &h)
	}

	return nil
}

func (h *CFNParametersValue) String() string {
	return ""
}

func CFNParameters(s kingpin.Settings) (target *CFNParametersValue) {
	target = &CFNParametersValue{}
	s.SetValue((*CFNParametersValue)(target))
	return
}

func (h *CFNTagsValue) Set(value string) (error) {
	var r = regexp.MustCompile("(\\w+)=(\\w+)")

	keysVars := r.FindAllStringSubmatch(value, -1)

	if len(keysVars) == 0 {
		return fmt.Errorf("expected KEY=VALUE got '%s'", value)
	}

	for _, kv := range keysVars {
		*h = append(*h, &cloudformation.Tag{
			Key: aws.String(kv[1]),
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
	s.SetValue((*CFNTagsValue)(target))
	return
}









func (m *KeyValValue) Set(value string) (error) {
	var r = regexp.MustCompile("(\\w+)=(\\w+)")
	keysVars := r.FindAllStringSubmatch(value, -1)
	if len(keysVars) == 0 {
		return fmt.Errorf("expected KEY=VALUE got '%s'", value)
	}

	for _, kv := range keysVars {
		(*m)[kv[1]] = kv[2]
	}

	return nil
}

func (h *KeyValValue) String() string {
	return ""
}

func KeyVal(s kingpin.Settings) (target *map[string]string) {
	target = &(map[string]string{})
	s.SetValue((*KeyValValue)(target))
	return
}