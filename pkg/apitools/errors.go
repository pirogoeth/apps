package apitools

import (
	"github.com/gin-gonic/gin"
)

const (
	// MsgInvalidParameter is used when the parameter value can't be parsed or is otherwise invalid
	MsgInvalidParameter = "invalid parameter value"
	// ErrNoQueryProvided is used when no query parameter is provided but is expected
	MsgNoQueryProvided = "no query parameter provided"
	// ErrFailedToBind is used when request body failed to bind to the destination object
	MsgFailedToBind   = "failed to bind parameters"
	MsgNotImplemented = "not yet implemented <3 c:"
)

// ErrorPayload builds a "standardized" error payload for return to the user.
func ErrorPayload(message string, err error) *gin.H {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	} else {
		errMsg = "<no error>"
	}

	return &gin.H{
		"message": message,
		"error":   errMsg,
	}
}
