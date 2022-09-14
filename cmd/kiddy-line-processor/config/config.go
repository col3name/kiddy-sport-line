package config

import (
	loggerInterface "github.com/col3name/lines/pkg/common/application/logger"
	str "github.com/col3name/lines/pkg/common/util/stringss"
	"os"
	"strconv"
)

type Config struct {
	UpdatePeriod     int
	HttpUrl          string
	GrpcUrl          string
	LinesProviderUrl string
	LogLevel         string
	DbUrl            string
}

func SetupConfig(logger loggerInterface.Logger) *Config {
	updatePeriod := getEnvVariableInt("UPDATE_INTERVAL", 1, logger)
	linesProviderUrl := getEnvVariable("LINES_PROVIDER_URL", "http://localhost:8000")
	dbURL := getEnvVariable("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/lines")
	httpUrl := getEnvVariable("HTTP_URL", ":3333")
	grpcUrl := getEnvVariable("GRPC_URL", ":50051")

	return &Config{
		UpdatePeriod:     updatePeriod,
		HttpUrl:          httpUrl,
		GrpcUrl:          grpcUrl,
		LinesProviderUrl: linesProviderUrl,
		DbUrl:            dbURL,
		LogLevel:         "",
	}
}

func getEnvVariableInt(key string, defaultValue int, logger loggerInterface.Logger) int {
	defaultVal := strconv.Itoa(defaultValue)
	valueString := getEnvVariable(key, defaultVal)
	value, err := strconv.Atoi(valueString)
	msg := key + " must be positive integer. Set default value: " + defaultVal
	if err != nil {
		logger.Error(msg)
		return defaultValue
	} else if value < 1 {
		logger.Error(msg)
		return defaultValue
	}

	return value
}

func getEnvVariable(key, defaultVal string) string {
	value := os.Getenv(key)
	if str.Empty(value) {
		value = defaultVal
	}
	return value
}
