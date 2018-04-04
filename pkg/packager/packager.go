package packager

import (
	"os"

	"net/http"

	"net/url"

	"strings"

	"crypto/rand"
	"fmt"

	"archive/zip"

	"io"

	"encoding/hex"

	"path/filepath"

	"encoding/json"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/awslabs/goformation/cloudformation"
	"github.com/b-b3rn4rd/cfn/pkg/command"
	"github.com/b-b3rn4rd/cfn/pkg/uploader"
	"github.com/pkg/errors"
	yamlwrapper "github.com/sanathkr/yaml"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

type Packageriface interface {
	Export(*command.PackageParams) (*cloudformation.Template, error)
	WriteOutput(*string, []byte) error
}

type Packager struct {
	logger *logrus.Logger
	fs     afero.Fs
}

// New creates a new Packager struct
func New(logger *logrus.Logger, fs afero.Fs) *Packager {
	return &Packager{
		logger: logger,
		fs:     fs,
	}
}

// Export upload code for specific resources and modify template
func (p *Packager) Export(packageParams *command.PackageParams) (*cloudformation.Template, error) {
	p.logger.WithField("templateFile", *packageParams.TemplateFile).Debug("opening cfn template")

	template := &cloudformation.Template{}
	data, err := ioutil.ReadFile(*packageParams.TemplateFile)
	yamlwrapper.
		data, err = yamlwrapper.YAMLToJSON(data)
	json.Unmarshal(data, template)

	//template, err := goformation.OpenWithOptions(*packageParams.TemplateFile, &intrinsics.ProcessorOptions{
	//	IntrinsicHandlerOverrides: map[string]intrinsics.IntrinsicHandler{
	//		"Fn::GetAtt": func(name string, input interface{}, template interface{}) interface{} {
	//			return fmt.Sprintf("!%s \"%s\"", strings.Replace(name, "Fn::", "", -1), input)
	//		},
	//	},
	//})

	if err != nil {
		return nil, errors.Wrap(err, "error while opening cfn")
	}

	for resourceID, raw := range template.Resources {
		untyped := raw.(map[string]interface{})

		switch untyped["Type"] {
		case "AWS::Serverless::Function":
			resource, err := template.GetAWSServerlessFunctionWithName(resourceID)

			s3URL, err := p.exportAWSServerlessFunction(packageParams.S3Uploader, resource)

			if err != nil {
				return nil, errors.Wrap(err, "error while exporting code")
			}

			if s3URL != "" {
				p.logger.WithField("s3URL", s3URL).Debug("new code URL")
				resource.CodeUri.String = aws.String(s3URL)
				template.Resources[resourceID] = &resource
			}
		}
	}

	return template, nil
}

func (p *Packager) isLocalFile(filepath string) bool {
	p.logger.WithField("filepath", filepath).Debug("checking if file exists locally")

	_, err := p.fs.Stat(filepath)
	return err == nil
}

func (p *Packager) isZipFile(filepath string) bool {
	p.logger.WithField("filepath", filepath).Debug("checking if file is zip")

	f, err := p.fs.Open(filepath)
	defer f.Close()

	if err != nil {
		return false
	}

	buffer := make([]byte, 512)
	n, err := f.Read(buffer)

	if err != nil {
		return false
	}

	contentType := http.DetectContentType(buffer[:n])

	return contentType == "application/zip" || contentType == "application/x-gzip"
}

func (p *Packager) isS3URL(rawURL string) bool {
	p.logger.WithField("url", rawURL).Debug("checking if file is s3 url")

	url, err := url.Parse(rawURL)

	if err != nil {
		return false
	}

	return strings.ToLower(url.Scheme) == "s3"
}

func (p *Packager) exportAWSServerlessFunction(s3uploader uploader.Uploaderiface, resource cloudformation.AWSServerlessFunction) (string, error) {

	if resource.CodeUri.String == nil {
		p.logger.Debug("lambda CodeUri is not a URL, no upload required")
		return "", nil
	}

	if p.isS3URL(*resource.CodeUri.String) {
		p.logger.WithField("CodeUri", *resource.CodeUri.String).Debug("lambda CodeUri is already S3 URL, no upload required")
		return "", nil
	}

	if p.isLocalFile(*resource.CodeUri.String) {
		var zipname string
		var err error

		if p.isZipFile(*resource.CodeUri.String) {
			p.logger.WithField("zip", *resource.CodeUri.String).Debug("code is already zip")
			zipname = *resource.CodeUri.String
		} else {
			zipname, err = p.zip(*resource.CodeUri.String)

			if err != nil {
				return "", errors.Wrap(err, "error while zipping code")
			}

			defer p.fs.Remove(zipname)

			p.logger.WithField("zip", zipname).Debug("code was archived into zip")
		}

		s3Url, err := s3uploader.UploadWithDedup(aws.String(zipname), "zip")

		if err != nil {
			return "", errors.Wrap(err, "error while uploading code")
		}

		p.logger.WithField("s3url", s3Url).Debug("zip was uploaded to s3")

		return s3Url, nil
	}

	return "", nil
}

func (p *Packager) zip(source string) (string, error) {
	random := make([]byte, 16)
	_, err := rand.Read(random)

	if err != nil {
		return "", err
	}

	target := fmt.Sprintf("data-%s.zip", hex.EncodeToString(random))

	zipfile, err := p.fs.Create(target)

	if err != nil {
		return "", err
	}

	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	info, err := p.fs.Stat(source)
	if err != nil {
		return "", err
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		if baseDir != "" {
			header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := p.fs.Open(path)

		if err != nil {
			return err
		}

		defer file.Close()

		_, err = io.Copy(writer, file)

		return err
	})

	if err != nil {
		return "", err
	}

	return target, nil
}

// WriteOutput write template info specified file
func (p *Packager) WriteOutput(outputTemplateFile *string, data []byte) error {
	f, err := p.fs.OpenFile(*outputTemplateFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(0644))

	if err != nil {
		return err
	}

	n, err := f.Write(data)

	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}

	if err1 := f.Close(); err == nil {
		err = err1
	}

	return err
}
