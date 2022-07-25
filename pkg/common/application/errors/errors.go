package errors

import "errors"

var ErrInternal = errors.New("internalServerError")
var ErrInvalidArgument = errors.New("invalidArgumentError")
