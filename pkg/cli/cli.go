package cli

import (
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"encoding/json"
	"github.com/alecthomas/kingpin"
	"os"
	"io/ioutil"
)

type CFNParametersValue []*cloudformation.Parameter
type CFNTagsValue []*cloudformation.Tag


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

func (h *CFNTagsValue) String() string {
	return ""
}

func CFNTags(s kingpin.Settings) (target *CFNTagsValue) {
	target = &CFNTagsValue{}
	s.SetValue((*CFNTagsValue)(target))
	return
}