package middlewares

import (
	"bytes"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
)

const (
	ErrCouldNotModify = "could not modify response body: %w"
)

var _ gin.ResponseWriter = (*BodyModifier)(nil)
var _ io.Writer = (*BodyModifier)(nil)

type BodyModifier struct {
	gin.ResponseWriter

	body *bytes.Buffer
}

func NewBodyModifier(w gin.ResponseWriter) *BodyModifier {
	return &BodyModifier{
		ResponseWriter: w,
		body:           new(bytes.Buffer),
	}
}

func (bm *BodyModifier) Flush() {
	bm.ResponseWriter.Write(bm.body.Bytes())
}

func (bm *BodyModifier) Write(p []byte) (int, error) {
	return bm.body.Write(p)
}

func (bm *BodyModifier) MustModify(modifier func(*bytes.Buffer) (*bytes.Buffer, error)) {
	var err error
	bm.body, err = modifier(bm.body)
	if err != nil {
		panic(fmt.Errorf(ErrCouldNotModify, err))
	}
}

func (bm *BodyModifier) Modify(modifier func(*bytes.Buffer) (*bytes.Buffer, error)) (ok bool, err error) {
	bm.body, err = modifier(bm.body)
	if err != nil {
		return false, fmt.Errorf(ErrCouldNotModify, err)
	}
	return true, nil
}
