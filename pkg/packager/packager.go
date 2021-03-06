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

	"io/ioutil"

	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/awslabs/goformation/cloudformation"
	"github.com/b-b3rn4rd/gocfn/pkg/uploader"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

type Packageriface interface {
	Export(*PackageParams) (*Template, error)
	WriteOutput(*string, []byte) error
	Marshall(string, *Template) ([]byte, error)
	Open(string) (*Template, error)
}

// PackageParams parameters required for package params
type PackageParams struct {
	S3Uploader         uploader.Uploaderiface
	TemplateFile       string
	OutputTemplateFile string
}

// Template struct
type Template struct {
	Transform string `json:"Transform,omitempty"`
	cloudformation.Template
}

// Packager struct
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
func (p *Packager) Export(packageParams *PackageParams) (*Template, error) {
	template, err := p.Open(packageParams.TemplateFile)
	if err != nil {
		return nil, err
	}

	for resourceID, resourceType := range template.Resources {

		switch resourceType.AWSCloudFormationType() {
		case "AWS::Serverless::Function":
			resource, err := template.GetAWSServerlessFunctionWithName(resourceID)

			if err != nil {
				return nil, errors.Wrap(err, "error while searching for serverless func")
			}

			s3URL, err := p.exportAWSServerlessFunction(packageParams.S3Uploader, *resource)
			if err != nil {
				return nil, errors.Wrap(err, "error while exporting code")
			}

			if s3URL != "" {
				p.logger.WithField("s3URL", s3URL).Debug("new code URL")
				resource.CodeUri.String = aws.String(s3URL)
				template.Resources[resourceID] = resource
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

func (p *Packager) escapeTags(data []byte) []byte {
	r := string(data)
	r = strings.Replace(r, "!", "\\!", -1)

	return []byte(r)
}

func (p *Packager) normaliseTags(data []byte) []byte {
	r := string(data)
	r = strings.Replace(r, "\\!", "!", -1)

	return []byte(r)
}

func (p *Packager) isZipFile(filepath string) bool {
	p.logger.WithField("filepath", filepath).Debug("checking if file is zip")

	f, err := p.fs.Open(filepath)
	if err != nil {
		return false
	}

	defer f.Close()

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

func (p *Packager) Marshall(filename string, template *Template) ([]byte, error) {
	var raw []byte
	var err error

	isYaml := p.isYAML(filename)

	if isYaml {
		p.logger.WithField("filename", filename).Debug("file is yaml, converting to yaml")
		raw, err = yaml.Marshal(template)
	} else {
		p.logger.WithField("filename", filename).Debug("file is json, converting to json")
		raw, err = json.MarshalIndent(template, "", " ")
	}
	if err != nil {
		return nil, err
	}

	if isYaml {
		p.logger.WithField("filename", filename).Debug("file is yaml, normalise tags")
		raw = p.normaliseTags(raw)
	}

	return raw, nil
}

func (p *Packager) Open(filename string) (*Template, error) {
	p.logger.WithField("templateFile", filename).Debug("opening cfn template")

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrap(err, "error while opening cfn")
	}

	if p.isYAML(filename) {
		data, err = yaml.YAMLToJSON(p.escapeTags(data))

		if err != nil {
			return nil, errors.Wrap(err, "error while converting yaml to json")
		}
	}

	template := &Template{}

	if err := json.Unmarshal(data, template); err != nil {
		return nil, errors.Wrap(err, "error while unmarshalling cfn")
	}

	return template, nil
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

		s3Url, _ = s3uploader.URLTos3Path(s3Url)

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

func (p *Packager) isYAML(filename string) bool {
	ext := filepath.Ext(filename)

	return strings.ToLower(ext) == ".yaml" || strings.ToLower(ext) == ".yml"
}
