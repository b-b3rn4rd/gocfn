package packager

import (
	"os"

	"github.com/awslabs/goformation"
	"github.com/awslabs/goformation/cloudformation"
	"github.com/sirupsen/logrus"
)

type Packageriface interface {
	Export(templateFile *string) error
}

type Packager struct {
	logger *logrus.Logger
}

func New(logger *logrus.Logger) *Packager {
	return &Packager{
		logger: logger,
	}
}

func (p *Packager) Export(templateFile *string, outputTemplateFile *string) error {
	template, err := goformation.Open(*templateFile)

	if err != nil {
		return err
	}

	for _, function := range template.GetAllAWSServerlessFunctionResources() {
		if function.CodeUri.S3Location != nil {
			continue
		}
	}

	err = p.WriteOutput(outputTemplateFile, template)

	if err != nil {
		return err
	}

	return nil
}

func (p *Packager) isLocalFile(filepath string) bool {
	_, err := os.Stat(filepath)
	return err == nil
}

func (p *Packager) export(codeURI *string) {

}
func (p *Packager) WriteOutput(outputFemplateFile *string, template *cloudformation.Template) error {
	return nil
}
