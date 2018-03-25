package writer_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/b-b3rn4rd/cfn/pkg/writer"
	"github.com/stretchr/testify/assert"
)

func TestWritePassesWriterAndMessageToFormatFunction(t *testing.T) {
	expectedMessage := "hello world"

	out := &bytes.Buffer{}
	wrt := writer.New(out, func(wr io.Writer, message interface{}) {

		assert.Equal(t, expectedMessage, message)
		assert.Equal(t, out, wr)
	})

	wrt.Write(expectedMessage)
}

func TestJsonFormatter(t *testing.T) {

	out := &bytes.Buffer{}

	message := struct {
		Text string
	}{"hello world"}

	expectedMessage, _ := json.MarshalIndent(message, "", "    ")
	writer.JSONFormatter(out, message)

	assert.Equal(t, out.String(), fmt.Sprintf("%s\n", string(expectedMessage)))
}

func TestPlainFormatter(t *testing.T) {
	out := &bytes.Buffer{}
	expectedMessage := "hello world"
	writer.PlainFormatter(out, expectedMessage)
	assert.Equal(t, out.String(), fmt.Sprintf("%s\n", expectedMessage))
}
