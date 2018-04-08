package uploader

import (
	"crypto/md5"
	"fmt"
	"io"

	"net/url"

	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

type Uploaderiface interface {
	UploadWithDedup(*string, string) (string, error)
	FileExists(*string) bool
	FileChecksum(*string) (string, error)
	MakeURL(*string) string
	UrlTos3Path(string) (string, error)
}

type Uploader struct {
	svc         s3iface.S3API
	logger      *logrus.Logger
	uploader    s3manageriface.UploaderAPI
	bucketName  *string
	prefix      *string
	kmsKeyID    *string
	forceUpload *bool
	appFs       afero.Fs
}

func New(svc s3iface.S3API, uSvc s3manageriface.UploaderAPI, logger *logrus.Logger, bucketName *string, prefix *string, kmsKeyID *string, forceUpload *bool, fs afero.Fs) *Uploader {

	return &Uploader{
		svc:         svc,
		logger:      logger,
		uploader:    uSvc,
		bucketName:  bucketName,
		prefix:      prefix,
		kmsKeyID:    kmsKeyID,
		forceUpload: forceUpload,
		appFs:       fs,
	}
}

func (u *Uploader) FileChecksum(filename *string) (string, error) {
	f, err := u.appFs.Open(*filename)

	defer f.Close()

	if err != nil {
		return "", errors.Wrap(err, "Error while opening file")
	}

	h := md5.New()

	if _, err := io.Copy(h, f); err != nil {
		return "", errors.Wrap(err, "Error while opening file")
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (u *Uploader) UrlTos3Path(s3Url string) (string, error) {
	url, err := url.Parse(s3Url)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("s3://%s", strings.Trim(url.Path, "/")), err
}

func (u *Uploader) UploadWithDedup(filename *string, extension string) (string, error) {
	f := logrus.Fields{
		"bucketName": *u.bucketName,
		"prefix":     *u.prefix,
		"filename":   *filename,
	}

	u.logger.WithFields(f).Debug("Calculating md5 of uploaded file")

	m5hash, err := u.FileChecksum(filename)
	removePath := fmt.Sprintf("%s.%s", m5hash, extension)

	u.logger.WithFields(f).WithField("Hash", m5hash).Debug(fmt.Sprintf("M5 of file content"))
	if err != nil {
		return "", err
	}

	return u.upload(filename, &removePath)

}

func (u *Uploader) upload(filename *string, remotePath *string) (string, error) {

	if *u.prefix != "" {
		*remotePath = fmt.Sprintf("%s/%s", *u.prefix, *remotePath)
	}

	u.logger.WithField("filename", *remotePath).Debug("Checking if file already exist")

	if u.FileExists(remotePath) && !*u.forceUpload {
		u.logger.WithField("filename", *remotePath).WithField("templateUrl", u.MakeURL(remotePath)).Debug("File with same data is already exists, skipping upload")
		return u.MakeURL(remotePath), nil
	}

	u.logger.WithField("filename", *remotePath).Debug("Uploading file")

	raw, err := u.appFs.Open(*filename)

	if err != nil {
		return "", errors.Wrap(err, "Error while opening filename")
	}

	uploadInput := &s3manager.UploadInput{
		Bucket: u.bucketName,
		Key:    remotePath,
		Body:   raw,
	}

	if *u.kmsKeyID != "" {
		u.logger.WithField("kmsKeyID", *u.kmsKeyID).Debug("KMS key is specified")
		uploadInput.ServerSideEncryption = aws.String("aws:kms")
		uploadInput.SSEKMSKeyId = u.kmsKeyID
	} else {
		uploadInput.ServerSideEncryption = aws.String("AES256")
	}

	resp, err := u.uploader.Upload(uploadInput)

	if err != nil {
		return "", errors.Wrap(err, "AWS error while uploading to s3")
	}

	logrus.WithField("filename", *remotePath).WithField("uploadId", resp.UploadID).Debug(fmt.Sprintf("File has been uploaded"))
	return resp.Location, nil
}

func (u *Uploader) MakeURL(remotePath *string) string {
	var region *string

	if s3, ok := u.svc.(*s3.S3); ok {
		region = s3.Config.Region
	} else {
		region = aws.String("us-west-2")
	}

	base := "https://s3.amazonaws.com"

	if *region != "us-east-1" {
		base = fmt.Sprintf("https://s3-%s.amazonaws.com", *region)
	}

	return fmt.Sprintf("%s/%s/%s", base, *u.bucketName, *remotePath)
}

func (u *Uploader) FileExists(remotePath *string) bool {
	_, err := u.svc.HeadObject(&s3.HeadObjectInput{
		Bucket: u.bucketName,
		Key:    remotePath,
	})

	return err == nil
}
