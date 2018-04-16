package main

import (
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/b-b3rn4rd/gocfn/pkg/writer"
	"github.com/sirupsen/logrus"
)

var (
	version       = "master"
	debug         = kingpin.Flag("debug", "Enable debug logging.").Short('d').Bool()
	logger        = logrus.New()
	jsonOutWriter = writer.New(os.Stdout, writer.JSONFormatter)
	strOutWriter  = writer.New(os.Stdout, writer.PlainFormatter)
	exiter        = os.Exit
)

func main() {
	kingpin.Version(version)
	runCommand := kingpin.Parse()

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
		logger.SetLevel(logrus.DebugLevel)
	}

	logger.Formatter = &logrus.JSONFormatter{}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	switch runCommand {
	case "deploy":
		deploy(sess)
	case "package":
		packaage(sess)
	}
}
