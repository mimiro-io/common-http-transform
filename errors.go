package common_http_transform

import (
	"fmt"

	"github.com/labstack/echo/v4"
)

type LayerErrorType int

const (
	LayerErrorBadParameter LayerErrorType = iota
	LayerErrorInternal
	LayerNotSupported
)

type TransformError interface {
	error
	toHTTPError() *echo.HTTPError
	Underlying() error
}

type transformError struct {
	err     error
	errType LayerErrorType
}

func (l transformError) Underlying() error {
	return l.err
}

func (l transformError) toHTTPError() *echo.HTTPError {
	// TODO: map LayerErrorType to HTTP status code and message
	return echo.NewHTTPError(500, l.err.Error())
}

func (l transformError) Error() string {
	return l.err.Error()
}

func Err(err error, errType LayerErrorType) TransformError {
	if err == nil {
		return nil
	}
	return &transformError{err, errType}
}

func Errorf(errType LayerErrorType, format string, args ...any) TransformError {
	return &transformError{err: fmt.Errorf(format, args...), errType: errType}
}
