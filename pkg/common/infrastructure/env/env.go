package env

import (
	loggerInterface "github.com/col3name/lines/pkg/common/application/logger"
	str "github.com/col3name/lines/pkg/common/infrastructure/util/stringss"
	"os"
	"strconv"
)

func GetEnvVariableInt(key string, defaultValue int, logger loggerInterface.Logger) int {
	defaultVal := strconv.Itoa(defaultValue)
	valueString := GetEnvVariable(key, defaultVal)
	value, err := strconv.Atoi(valueString)
	msg := key + " must be positive integer. Set default value: " + defaultVal
	if err != nil {
		logger.Error(msg)
		return defaultValue
	}
	if value < 1 {
		logger.Error(msg)
		return defaultValue
	}

	return value
}

func GetEnvVariable(key, defaultVal string) string {
	value := os.Getenv(key)
	if str.Empty(value) {
		value = defaultVal
	}
	return value
}
