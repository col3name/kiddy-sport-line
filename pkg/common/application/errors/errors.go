package errors

import "errors"

const TableNotExistMessage = " does not exist (SQLSTATE 42P01)"

var (
	ErrInternal        = errors.New("internalServerError")
	ErrExternal        = errors.New("externalServerError")
	ErrInvalidArgument = errors.New("invalidArgumentError")
	ErrTableNotExist   = errors.New("table does not exist")
)
