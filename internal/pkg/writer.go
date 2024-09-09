package pkg

import (
	"bytes"
	"net/http"
)

type ResponseWriterWrapper struct {
	body   *bytes.Buffer
	header http.Header
	code   int
}

func NewWriter() *ResponseWriterWrapper {
	return &ResponseWriterWrapper{body: bytes.NewBufferString(""), header: make(http.Header)}
}

func (w *ResponseWriterWrapper) Header() http.Header {
	return w.header
}

func (w *ResponseWriterWrapper) WriteHeader(statusCode int) {
	w.code = statusCode
}

func (w *ResponseWriterWrapper) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *ResponseWriterWrapper) WriteString(s string) (int, error) {
	return w.body.WriteString(s)
}

func (w *ResponseWriterWrapper) Code() int {
	return w.code
}

func (w *ResponseWriterWrapper) Body() []byte {
	return w.body.Bytes()
}

func (w *ResponseWriterWrapper) SetCode(code int) {
	w.code = code
}

func (w *ResponseWriterWrapper) SetBody(body *bytes.Buffer) {
	w.body = body
}
