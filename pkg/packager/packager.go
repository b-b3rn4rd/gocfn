package packager

import "github.com/sirupsen/logrus"

type Packageriface interface {
}

type Packager struct {
	logger *logrus.Logger
}

func New() *Packager {
	return &Packager{}
}
