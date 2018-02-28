package uploader

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/aws/aws-sdk-go/aws"
	"io/ioutil"
	"crypto/x509/pkix"
	"os"
	"log"
	"io"
	"crypto/md5"
)

type Uploader struct {
	svc s3iface.S3API
	uploader s3manageriface.UploaderAPI
	bucketName *string
	prefix *string
	kmsKeyId *string
	forceUpload *bool
}

func New(svc s3iface.S3API, bucketName *string, prefix *string, kmsKeyId *string, forceUpload *bool) *Uploader {

	return &Uploader{
		svc: svc,
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
	m5hash, err := u.fileChecksum(filename)
	removePath := fmt.Sprintf("%s.%s", m5hash, extension)

	if err != nil {
		return "", err
	}

	return u.upload(filename, &removePath)

}

func (u *Uploader) upload(filename *string, remotePath *string) (string, error) {

	if *u.prefix != "" {
		*remotePath = fmt.Sprintf("%s/%s", u.prefix, remotePath)
	}

	if  u.FileExists(remotePath) && !*u.forceUpload {
		return "", errors.New(fmt.Sprintf("File with same data is already exists at %s. Skipping upload", u.makeUrl(remotePath)))
	}

	raw, err := os.Open(*filename)

	uploadInput := &s3manager.UploadInput{
		Bucket:u.bucketName,
		Key: remotePath,
		Body: raw,
	}

	if u.kmsKeyId != nil {
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
	return fmt.Sprintf("s3://%s/%s", u.bucketName, remotePath)
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