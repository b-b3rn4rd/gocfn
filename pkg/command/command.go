package command

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/b-b3rn4rd/gocfn/pkg/uploader"
)

type Params interface {
	String() string
}

// DeployParams parameters required for deploy command
type DeployParams struct {
	S3Uploader           uploader.Uploaderiface
	StackName            *string
	TemplateFile         *string
	Parameters           []*cloudformation.Parameter
	Capabilities         []*string
	NoExecuteChangeset   *bool
	RoleArn              *string
	NotificationArns     []*string
	FailOnEmptyChangeset *bool
	Tags                 []*cloudformation.Tag
	ForceDeploy          *bool
}

// PackageParams parameters required for package params
type PackageParams struct {
	S3Uploader         uploader.Uploaderiface
	TemplateFile       *string
	OutputTemplateFile *string
}

func (p *DeployParams) String() string {
	raw, _ := json.MarshalIndent(*p, "", "    ")
	return string(raw)
}
