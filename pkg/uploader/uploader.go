package uploader

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Uploader struct {
	svc s3iface.S3API
	bucketName *string
	region *string
	prefix *string
	kmsKeyId *string
	forceUpload bool
}

func New(svc s3iface.S3API, bucketName *string, region *string, prefix *string, kmsKeyId *string, forceUpload bool) *Uploader {
	return &Uploader{
		svc: svc,
		bucketName: bucketName,
		region: region,
		prefix: prefix,
		kmsKeyId: kmsKeyId,
		forceUpload:forceUpload,
	}
}

func (u *Uploader) Upload(filename *string, remotePath *string) (string, error){
	if *u.prefix != "" {
		*remotePath = fmt.Sprintf("%s/%s", u.prefix, remotePath)
	}

	if  u.FileExists(remotePath) && !u.forceUpload {

	}
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