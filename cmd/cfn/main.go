package main

import (
	"fmt"
	//"github.com/b-b3rn4rd/cfn/pkg/stringme"
	"github.com/alecthomas/kingpin"
)
var (
	version = "master"
	tracing = kingpin.Flag("trace", "Enable trace mode.").Short('t').Bool()
	debug   = kingpin.Flag("debug", "Enable debug logging.").Short('d').Bool()
	//cwlogsCommand = kingpin.Command("cwlogs", "Process cloudwatch logs data from kinesis.")
	name    = kingpin.Arg("name", "Name of user.").Required().String()
)

func main()  {
	kingpin.Version(version)
	subCommand := kingpin.Parse()

	fmt.Printf(subCommand)
}