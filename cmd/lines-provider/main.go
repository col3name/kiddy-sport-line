package main

import (
	"github.com/col3name/lines/pkg/common/infrastructure/env"
	"github.com/col3name/lines/pkg/common/infrastructure/logrusLogger"
	httpUtil "github.com/col3name/lines/pkg/common/infrastructure/transport/http"
	"github.com/col3name/lines/pkg/lines-provider/infrastructure/transport/http/router"
)

func main() {
	logger := logrusLogger.New()

	port := env.GetEnvVariable("PORT", "8000")

	serverUrl := ":" + port
	logger.Info("listen and serve at", serverUrl)
	defer logger.Info("stop")

	httpUtil.RunHttpServer(serverUrl, router.Router(), logger)
}
