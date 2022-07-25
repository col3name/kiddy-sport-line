package errors

import "errors"

var ErrInternal = errors.New("internalServerError")
var ErrExternal = errors.New("externalServerError")
var ErrInvalidArgument = errors.New("invalidArgumentError")
