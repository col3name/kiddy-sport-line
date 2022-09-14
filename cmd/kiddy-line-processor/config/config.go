package config

import (
	loggerInterface "github.com/col3name/lines/pkg/common/application/logger"
	"github.com/col3name/lines/pkg/common/infrastructure/env"
)

type Config struct {
	UpdatePeriod     int
	HttpUrl          string
	GrpcUrl          string
	LinesProviderUrl string
	LogLevel         string
	DbUrl            string
}

func ParseConfig(logger loggerInterface.Logger) *Config {
	updatePeriod := env.GetEnvVariableInt("UPDATE_INTERVAL", 1, logger)
	linesProviderUrl := env.GetEnvVariable("LINES_PROVIDER_URL", "http://localhost:8000")
	dbURL := env.GetEnvVariable("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/lines")
	httpUrl := env.GetEnvVariable("HTTP_URL", ":3333")
	grpcUrl := env.GetEnvVariable("GRPC_URL", ":50051")

	return &Config{
		UpdatePeriod:     updatePeriod,
		HttpUrl:          httpUrl,
		GrpcUrl:          grpcUrl,
		LinesProviderUrl: linesProviderUrl,
		DbUrl:            dbURL,
		LogLevel:         "",
	}
}
