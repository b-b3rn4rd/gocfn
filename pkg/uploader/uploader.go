package uploader

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/aws/aws-sdk-go/aws"
	"os"
	"io"
	"crypto/md5"
	"github.com/sirupsen/logrus"
)

type Uploader struct {
	svc s3iface.S3API
	logger *logrus.Logger
	uploader s3manageriface.UploaderAPI
	bucketName *string
	prefix *string
	kmsKeyId *string
	forceUpload *bool
}

func New(svc s3iface.S3API, logger *logrus.Logger, bucketName *string, prefix *string, kmsKeyId *string, forceUpload *bool) *Uploader {

	return &Uploader{
		svc: svc,
		logger: logger,
		uploader: s3manager.NewUploaderWithClient(svc),
		bucketName: bucketName,
		prefix: prefix,
		kmsKeyId: kmsKeyId,
		forceUpload:forceUpload,
	}
}

func (u *Uploader) fileChecksum(filename *string) (string, error) {
	f, err := os.Open(*filename)

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

func (u *Uploader) UploadWithDedup(filename *string, extension string) (string, error) {
	f := logrus.Fields{
		"bucketName": *u.bucketName,
		"prefix": *u.prefix,
		"filename": *filename,
	}

	u.logger.WithFields(f).Debug("Calculating md5 of uploaded file")

	m5hash, err := u.fileChecksum(filename)
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

	if  u.FileExists(remotePath) && !*u.forceUpload {
		u.logger.WithField("filename", *remotePath).WithField("TemplateUrl", u.makeUrl(remotePath)).Debug("File with same data is already exists, skipping upload")
		return u.makeUrl(remotePath), nil
	}

	raw, err := os.Open(*filename)

	uploadInput := &s3manager.UploadInput{
		Bucket:u.bucketName,
		Key: remotePath,
		Body: raw,
	}

	if *u.kmsKeyId != "" {
		u.logger.WithField("kmsKeyId", *u.kmsKeyId).Debug("KMS key is specified")
		uploadInput.ServerSideEncryption = aws.String("aws:kms")
		uploadInput.SSEKMSKeyId = u.kmsKeyId
	} else {
		uploadInput.ServerSideEncryption = aws.String("AES256")
	}

	resp, err := u.uploader.Upload(uploadInput)

	if err != nil {
		return "", errors.Wrap(err, "AWS error while uploading to s3")
	}

	return resp.Location, nil
}

func (u *Uploader) makeUrl(remotePath *string) string {
	region := u.svc.(*s3.S3).Config.Region

	base := "https://s3.amazonaws.com"

	if *region != "us-east-1" {
		base = fmt.Sprintf("https://s3-%s.amazonaws.com", *region)
	}

	return fmt.Sprintf("%s/%s/%s", base, *u.bucketName, *remotePath)
}


func (u *Uploader) FileExists(remotePath *string) bool {
	_, err := u.svc.HeadObject(&s3.HeadObjectInput{
		Bucket: u.bucketName,
		Key: remotePath,
	})

	if err != nil {
		return false
	}

	return true
}