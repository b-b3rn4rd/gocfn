package uploader_test

import (
	"testing"
	//"io"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/b-b3rn4rd/gocfn/pkg/uploader"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type mockedS3API struct {
	s3iface.S3API
	headObjectResp s3.HeadObjectOutput
	err            error
}

type mockedUploaderAPI struct {
	s3manageriface.UploaderAPI
	uploadResp s3manager.UploadOutput
	err        error
}

func (m mockedS3API) HeadObject(*s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	return &m.headObjectResp, m.err
}

func (m mockedUploaderAPI) Upload(*s3manager.UploadInput, ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
	return &m.uploadResp, m.err
}

func TestUrlTos3Path(t *testing.T) {
	tests := map[string]struct {
		url      string
		expected string
	}{
		"url to s3": {
			url:      "http://example.com/bucketname/sam/filename.zip",
			expected: "s3://bucketname/sam/filename.zip",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tmpUploader := uploader.New(nil, nil, nil, nil, nil, nil, nil, nil)
			res, _ := tmpUploader.URLTos3Path(test.url)

			assert.Equal(t, test.expected, res)
		})
	}
}

func TestUploader(t *testing.T) {
	ext := "template"
	filename := "example-stack.yml"
	bucket := aws.String("test")
	prefix := aws.String("sam")
	kmsKeyId := aws.String("")
	forceUpload := aws.Bool(false)

	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, filename, []byte(""), 0644)

	tmpUploader := uploader.New(nil, nil, nil, bucket, prefix, nil, nil, fs)
	fileHash, _ := tmpUploader.FileChecksum(&filename)
	removePath := fmt.Sprintf("%s/%s.%s", *prefix, fileHash, ext)

	url := tmpUploader.MakeURL(&removePath)

	tests := map[string]struct {
		Svc   s3iface.S3API
		Usvc  s3manageriface.UploaderAPI
		Res   string
		Err   error
		Setup func()
	}{
		"New object upload": {
			Svc: mockedS3API{
				err: errors.New("file does not exist"),
			},
			Usvc: mockedUploaderAPI{
				uploadResp: s3manager.UploadOutput{
					Location: filename,
				},
				err: nil,
			},
			Res: url,
			Err: nil,
		},
		"Existing object without force is not uploaded": {
			Svc: mockedS3API{
				headObjectResp: s3.HeadObjectOutput{},
			},
			Res: url,
			Err: nil,
		},
		"Existing object with force is uploaded": {
			Setup: func() {
				*forceUpload = true
			},
			Svc: mockedS3API{
				headObjectResp: s3.HeadObjectOutput{},
			},
			Usvc: mockedUploaderAPI{
				uploadResp: s3manager.UploadOutput{
					Location: filename,
				},
				err: nil,
			},
			Res: url,
			Err: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.Setup != nil {
				test.Setup()
			}
			upldr := uploader.New(test.Svc, test.Usvc, logrus.New(), bucket, prefix, kmsKeyId, forceUpload, fs)

			resp, err := upldr.UploadWithDedup(&filename, ext)
			assert.Equal(t, test.Res, resp)

			if err != nil {
				assert.Equal(t, test.Err, err.Error())
			}

		})
	}

}
