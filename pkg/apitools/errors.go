package apitools

import "errors"

var (
	ErrFailedToBind     = errors.New("failed to bind parameters")
	ErrInvalidParameter = errors.New("invalid parameter for type")
)
