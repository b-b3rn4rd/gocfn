package writer

import (
	"io"
	"fmt"
	"encoding/json"
)

type FormatFunc func(wr io.Writer, message interface{})


type StringWriter struct {
	wr io.Writer
	format FormatFunc
}

func New(wr io.Writer, formatter FormatFunc) *StringWriter {
	return &StringWriter{
		wr:wr,
		format: formatter,
	}
}

func (w *StringWriter) Write(message interface{}) {
	w.format(w.wr, message)
}

func PlainFormatter(wr io.Writer, message interface{}) {
	fmt.Fprintln(wr, message)
}

func JsonFormatter(wr io.Writer, message interface{}) {
	raw, _ := json.MarshalIndent(message, "", "    ")
	fmt.Fprintln(wr, string(raw))
}
