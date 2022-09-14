package main

import (
	"context"
	"github.com/col3name/lines/cmd/kiddy-line-processor/config"
	"github.com/col3name/lines/pkg/common/application/errors"
	loggerInterface "github.com/col3name/lines/pkg/common/application/logger"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/infrastructure/logrusLogger"
	commonPostgres "github.com/col3name/lines/pkg/common/infrastructure/postgres"
	netHttp "github.com/col3name/lines/pkg/common/infrastructure/transport/net-http"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application/sport-line"
	domainQuery "github.com/col3name/lines/pkg/kiddy-line-processor/domain/query"
	domainRepo "github.com/col3name/lines/pkg/kiddy-line-processor/domain/repo"
	"github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/adapter"
	"github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/postgres"
	"github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/postgres/query"
	"github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/postgres/repo"
	grpcServer "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/transport/grpc"
	pb "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/transport/grpc/proto"
	"github.com/jackc/pgx/v4"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"sync"
	"time"
)

func main() {
	logger := logrusLogger.New()
	conf := config.SetupConfig(logger)
	conn := commonPostgres.SetupDbConnection(conf.DbUrl, logger)

	sportLineRepo := repo.NewSportLineRepository(conn, logger)
	sportLineQueryService := query.NewSportLineQueryService(conn, logger)

	err := performDbMigrationIfNeeded(sportLineQueryService, conn, logger)
	if err != nil {
		logger.Fatal(err)
	}

	s := &service{
		linesProviderAdapter: adapter.NewLinesProviderAdapter(conf.LinesProviderUrl, logger),
		conf:                 conf,
		sportLineRepo:        sportLineRepo,
		logger:               logger,
	}
	s.run()
}

const CreateSportLinesSql = `BEGIN TRANSACTION;
				CREATE TABLE sport_lines
				(
					id         UUID PRIMARY KEY UNIQUE NOT NULL,
					sport_type VARCHAR(255)            NOT NULL,
					score      REAL                     NOT NULL
				);
				
				INSERT INTO sport_lines (id, sport_type, score)
				VALUES ('ce267749-dec9-4d39-ad81-8b4cd8c381d2', 'baseball', 1.0),
					   ('ba9babe8-06d4-450e-8e9a-66b7512b5bd2', 'soccer', 1.0),
					   ('4b9d52e2-1473-4cdb-bba8-c1c1cac933f5', 'football', 1.0);
				END ;`

func performDbMigrationIfNeeded(sportLineRepo domainQuery.SportLineQueryService, conn commonPostgres.PgxPoolIface, logger loggerInterface.Logger) error {
	defaultSubscriptions := []commonDomain.SportType{commonDomain.Baseball}
	_, err := sportLineRepo.GetLinesBySportTypes(defaultSubscriptions)
	if err == nil {
		return nil
	}
	if err != errors.ErrTableNotExist {
		return err
	}

	cancelFunc, err := postgres.WithTx(conn, func(tx pgx.Tx) error {
		_, err = tx.Exec(context.Background(), CreateSportLinesSql)
		return err
	}, logger)
	if cancelFunc != nil {
		defer cancelFunc()
	}
	return err
}

type service struct {
	conf                  *config.Config
	logger                loggerInterface.Logger
	sportLineRepo         domainRepo.SportLineRepo
	linesProviderAdapter  adapter.LinesProviderAdapter
	sportLineQueryService domainQuery.SportLineQueryService
}

func (s *service) runHttpServer(wg *sync.WaitGroup) {
	defer wg.Done()
	http.HandleFunc("/ready", netHttp.ReadyCheckHandler)

	err := http.ListenAndServe(s.conf.HttpUrl, nil)
	if err != nil {
		s.logger.Fatal(err)
		return
	}
}

func (s *service) runGrpcServer(wg *sync.WaitGroup) {
	defer wg.Done()
	sportLineService := sport_line.NewSportLineService(s.sportLineQueryService)

	grpcSrv := grpc.NewServer()
	server := grpcServer.NewServer(sportLineService, s.logger)
	pb.RegisterKiddyLineProcessorServer(grpcSrv, server)

	lis, err := net.Listen("tcp", s.conf.GrpcUrl)
	if err != nil {
		s.logger.Fatalf("failed to listen: %v", err)
	}
	s.logger.Info("server listening at", lis.Addr().String())
	if err = grpcSrv.Serve(lis); err != nil {
		s.logger.Fatal("failed to serve: ", err)
	}
}

func (s *service) runSpotLineUpdateWorkers() {
	for _, sportType := range commonDomain.SupportSports {
		go s.updateSportLineWorker(sportType)
	}
}

func (s *service) updateSportLineWorker(sportType commonDomain.SportType) {
	for {
		sleepDuration := time.Duration(s.conf.UpdatePeriod) * time.Second
		sportLine, err := s.linesProviderAdapter.GetLineBySport(sportType)
		if err != nil {
			s.logger.Error(err)
			time.Sleep(sleepDuration)
			continue
		}
		err = s.sportLineRepo.Store(sportLine)
		if err != nil {
			s.logger.Error(err)
		}
		time.Sleep(sleepDuration)
	}
}

func (s *service) run() {
	var wg sync.WaitGroup
	wg.Add(2)
	go s.runHttpServer(&wg)
	go s.runGrpcServer(&wg)
	go s.runSpotLineUpdateWorkers()
	wg.Wait()
}
