package packager_test

import (
	"testing"

	"github.com/b-b3rn4rd/gocfn/pkg/command"

	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/b-b3rn4rd/gocfn/pkg/packager"
	"github.com/b-b3rn4rd/gocfn/pkg/uploader"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	test2 "github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type mockedS3Uploader struct {
	uploader.Uploaderiface
	uploadWithDedupResp string
	uploadWithDedupErr  error
	urlTos3PathResp     string
	urlTos3PathErr      error
}

func (u *mockedS3Uploader) UploadWithDedup(filename *string, extension string) (string, error) {
	return u.uploadWithDedupResp, u.uploadWithDedupErr
}

func (u *mockedS3Uploader) URLTos3Path(url string) (string, error) {
	return u.urlTos3PathResp, u.urlTos3PathErr
}

func TestExport(t *testing.T) {
	tests := map[string]struct {
		packageParams       *command.PackageParams
		exportResp          *packager.Template
		exportErr           error
		uploadWithDedupResp string
		uploadWithDedupErr  error
		urlTos3PathResp     string
		urlTos3PathErr      error
	}{
		"export uploads local file to s3 and modify CodeUri": {
			packageParams: &command.PackageParams{
				TemplateFile: aws.String("testdata/stack_with_local_file.yml"),
			},
			uploadWithDedupResp: "http://example.com/hello/abc.zip",
			urlTos3PathResp:     "s3://hello/abc.zip",
			exportResp: func() *packager.Template {
				logger, _ := test2.NewNullLogger()

				pkgr := packager.New(logger, afero.NewOsFs())
				template, _ := pkgr.Open("testdata/stack_with_local_file.yml")
				resource, _ := template.GetAWSServerlessFunctionWithName("Function")
				resource.CodeUri.String = aws.String("s3://hello/abc.zip")
				template.Resources["Function"] = &resource

				return template
			}(),
		},
		"export dont upload if file is already s3 url": {
			packageParams: &command.PackageParams{
				TemplateFile: aws.String("testdata/stack_with_s3_url.yml"),
			},
			uploadWithDedupResp: "not called",
			urlTos3PathResp:     "not called",
			exportResp: func() *packager.Template {
				logger, _ := test2.NewNullLogger()

				pkgr := packager.New(logger, afero.NewOsFs())
				template, _ := pkgr.Open("testdata/stack_with_s3_url.yml")
				return template
			}(),
		},
		"export dont upload codeUri is not string": {
			packageParams: &command.PackageParams{
				TemplateFile: aws.String("testdata/stack_invalid.yml"),
			},
			uploadWithDedupResp: "not called",
			urlTos3PathResp:     "not called",
			exportResp: func() *packager.Template {
				logger, _ := test2.NewNullLogger()

				pkgr := packager.New(logger, afero.NewOsFs())
				template, _ := pkgr.Open("testdata/stack_invalid.yml")

				return template
			}(),
		},
		"export returns error if upload failed": {
			packageParams: &command.PackageParams{
				TemplateFile: aws.String("testdata/stack_with_local_file.yml"),
			},
			uploadWithDedupErr: errors.New("error"),
			exportErr:          errors.New("error while exporting code: error while uploading code: error"),
		},
		"export upload already zipped file": {
			packageParams: &command.PackageParams{
				TemplateFile: aws.String("testdata/stack_with_zip.yml"),
			},
			uploadWithDedupResp: "http://example.com/hello/zipped.zip",
			urlTos3PathResp:     "s3://hello/zipped.zip",
			exportResp: func() *packager.Template {
				logger, _ := test2.NewNullLogger()

				pkgr := packager.New(logger, afero.NewOsFs())

				template, _ := pkgr.Open("testdata/stack_with_local_file.yml")
				resource, _ := template.GetAWSServerlessFunctionWithName("Function")
				resource.CodeUri.String = aws.String("s3://hello/zipped.zip")
				template.Resources["Function"] = &resource

				return template
			}(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			logger, _ := test2.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)
			fs := afero.NewOsFs()
			pkgr := packager.New(logger, fs)
			params := test.packageParams
			params.S3Uploader = &mockedS3Uploader{
				uploadWithDedupResp: test.uploadWithDedupResp,
				uploadWithDedupErr:  test.uploadWithDedupErr,
				urlTos3PathResp:     test.urlTos3PathResp,
			}

			res, err := pkgr.Export(params)

			assert.Equal(t, test.exportResp, res)

			if err != nil {
				assert.EqualError(t, test.exportErr, err.Error())
			}

		})
	}
}

func TestWriteOutput(t *testing.T) {
	tests := map[string]struct {
		writeOutputErr     error
		outputTemplateFile string
		data               []byte
	}{
		"WriteOutput writes data into specified file": {
			outputTemplateFile: "test.yaml",
			data:               []byte("hello world"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			logger, _ := test2.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)
			fs := afero.NewMemMapFs()
			pkgr := packager.New(logger, fs)

			err := pkgr.WriteOutput(aws.String(test.outputTemplateFile), test.data)

			if err != nil {
				assert.EqualError(t, test.writeOutputErr, err.Error())
			}

			f, _ := fs.Open(test.outputTemplateFile)
			defer f.Close()

			raw, _ := ioutil.ReadAll(f)

			assert.Equal(t, test.data, raw)
		})
	}
}
