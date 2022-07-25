package infrastructure

import (
	"github.com/col3name/lines/pkg/common/application/errors"
	log "github.com/sirupsen/logrus"
)

func InternalError(err error) error {
	if err != nil {
		log.Error(err)
	}
	return errors.ErrInternal
}

func ExternalError(err error) error {
	if err != nil {
		log.Error(err)
	}
	return errors.ErrExternal
}
