package writer

import (
	"testing"
	//"github.com/stretchr/testify/require"
	"io"
)

type TestWriter struct {
	name string
}

func (TestWriter) Write(p []byte) (n int, err error) {
	return 1, nil
}

func TestWritePassesWriterAndMessageToFormatFunction(t *testing.T) {
	expectedMessage := "hello world"
	exptectedWriter := "TestWriter"
	testWriter := TestWriter{name:exptectedWriter}
	wrt := New(testWriter, func(wr io.Writer, message interface{}) {

		if wr.(TestWriter).name != exptectedWriter {
			t.Fatalf("Format function is expected to be callbed with %s writer but got %s", exptectedWriter, wr.(TestWriter).name)
		}

		if message != expectedMessage {
			t.Fatalf("Format function is expected to be callbed with %s message but got %s", expectedMessage, message)
		}
	})

	wrt.Write(expectedMessage)
}

func TestJsonFormatterWritesPrettyJsonString(t *testing.T) {

}