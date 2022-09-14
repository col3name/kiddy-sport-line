package main

import (
	"github.com/col3name/lines/cmd/kiddy-line-processor/config"
	"github.com/col3name/lines/data/migrations/pg"
	loggerInterface "github.com/col3name/lines/pkg/common/application/logger"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/infrastructure/logrusLogger"
	commonPostgres "github.com/col3name/lines/pkg/common/infrastructure/postgres"
	grpcUtil "github.com/col3name/lines/pkg/common/infrastructure/transport/grpc"
	httpUtil "github.com/col3name/lines/pkg/common/infrastructure/transport/http"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application/service/sport-line"
	domainQuery "github.com/col3name/lines/pkg/kiddy-line-processor/domain/query"
	"github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/adapter"
	"github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/postgres/query"
	"github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/postgres/repo"
	grpcServer "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/transport/grpc"
	pb "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/transport/grpc/proto"
	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"sync"
	"time"
)

func main() {
	logger := logrusLogger.New()
	conf := config.ParseConfig(logger)
	conn := commonPostgres.SetupDbConnection(conf.DbUrl, logger)

	unitOfWork := repo.NewUnitOfWork(conn, logger)
	sportLineQueryService := query.NewSportLineQueryService(conn, logger)
	linesProviderAdapter := adapter.NewLinesProviderAdapter(conf.LinesProviderUrl, logger)
	newSportLineUpdateService := sport_line.NewSportLinesUpdateService(conf.UpdatePeriod, linesProviderAdapter, unitOfWork)
	migrationService := pg.NewMigrationService(sportLineQueryService, unitOfWork)

	s := newMicroservice(conf, logger, migrationService, sportLineQueryService, newSportLineUpdateService)
	s.run()
}

type microservice struct {
	conf                    *config.Config
	logger                  loggerInterface.Logger
	migration               pg.MigrationService
	sportLineQueryService   domainQuery.SportLineQueryService
	sportLinesUpdateService sport_line.SportLinesUpdateService
}

func newMicroservice(
	conf *config.Config,
	logger loggerInterface.Logger,
	migration pg.MigrationService,
	sportLineQueryService domainQuery.SportLineQueryService,
	sportLineUpdateService sport_line.SportLinesUpdateService,
) *microservice {

	return &microservice{
		conf:                    conf,
		logger:                  logger,
		migration:               migration,
		sportLineQueryService:   sportLineQueryService,
		sportLinesUpdateService: sportLineUpdateService,
	}
}

func (s *microservice) run() {
	var wg sync.WaitGroup
	wg.Add(2)
	err := s.performDbMigrationIfNeeded()
	if err != nil {
		s.logger.Fatal(err)
	}
	go s.runHttpServer(&wg)
	go s.runGrpcServer(&wg)
	go s.runSpotLineUpdateWorkers()
	wg.Wait()
}

func (s *microservice) performDbMigrationIfNeeded() error {
	return s.migration.MigrateIfNeeded()
}

func (s *microservice) runHttpServer(wg *sync.WaitGroup) {
	defer wg.Done()
	router := mux.NewRouter()
	router.HandleFunc("/ready", httpUtil.ReadyCheckHandler)
	handler := httpUtil.LogMiddleware(router, s.logger)
	httpUtil.RunHttpServer(s.conf.HttpUrl, handler, s.logger)
}

func (s *microservice) runGrpcServer(wg *sync.WaitGroup) {
	defer wg.Done()
	sportLineService := sport_line.NewSportLineService(s.sportLineQueryService)

	server := grpcServer.NewServer(sportLineService, s.logger)

	grpcSrv := grpc.NewServer()
	pb.RegisterKiddyLineProcessorServer(grpcSrv, server)

	grpcUtil.RunGrpcServer(s.logger, s.conf.GrpcUrl, grpcSrv)
}

func (s *microservice) runSpotLineUpdateWorkers() {
	for _, sportType := range commonDomain.SupportSports {
		go s.runUpdateSportLineWorker(sportType)
	}
}

func (s *microservice) runUpdateSportLineWorker(sportType commonDomain.SportType) {
	sleepDuration := time.Duration(s.conf.UpdatePeriod) * time.Second

	for {
		err := s.sportLinesUpdateService.Update(sportType)
		if err != nil {
			s.logger.Error(err)
		}
		time.Sleep(sleepDuration)
	}
}
