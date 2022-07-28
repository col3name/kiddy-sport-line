package infrastructure

import (
	"github.com/col3name/lines/pkg/common/application/errors"
	"github.com/col3name/lines/pkg/common/application/logger"
)

func InternalError(logger logger.Logger, err error) error {
	if err != nil {
		logger.Error(err)
	}
	return errors.ErrInternal
}

func ExternalError(logger logger.Logger, err error) error {
	if err != nil {
		logger.Error(err)
	}
	return errors.ErrExternal
}
