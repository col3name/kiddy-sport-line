package fake

import "github.com/col3name/lines/pkg/common/application/logger"

type Logger struct{}

func (Logger) With(_ logger.Fields) logger.Logger {
	return Logger{}
}

func (Logger) WithError(_ error) logger.Logger {
	return Logger{}
}

func (Logger) Debug(_ ...interface{}) {
}

func (Logger) Error(_ ...interface{}) {
}

func (Logger) Warn(_ ...interface{}) {
}

func (Logger) Info(_ ...interface{}) {
}

func (Logger) Fatal(_ ...interface{}) {
}

func (Logger) Fatalf(_ string, _ ...interface{}) {
}

func (Logger) Println(_ ...interface{}) {
}
