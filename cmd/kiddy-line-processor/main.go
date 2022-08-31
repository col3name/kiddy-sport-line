package main

import (
	"context"
	"github.com/col3name/lines/pkg/common/application/errors"
	loggerInterface "github.com/col3name/lines/pkg/common/application/logger"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/infrastructure/logrusLogger"
	commonPostgres "github.com/col3name/lines/pkg/common/infrastructure/postgres"
	netHttp "github.com/col3name/lines/pkg/common/infrastructure/transport/net-http"
	str "github.com/col3name/lines/pkg/common/util/stringss"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application"
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain"
	"github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/adapter"
	"github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/postgres"
	grpcServer "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/transport/grpc"
	pb "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/transport/grpc/proto"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

func main() {
	logger := logrusLogger.New()
	conf := setupConfig(logger)
	conn := setupDbConnection(conf.DbUrl, logger)
	sportLineRepo := postgres.NewSportLineRepository(conn, logger)
	err := performDbMigrationIfNeeded(sportLineRepo, conn, logger)
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

func performDbMigrationIfNeeded(sportLineRepo domain.SportRepo, conn commonPostgres.PgxPoolIface, logger loggerInterface.Logger) error {
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

type config struct {
	UpdatePeriod     int
	HttpUrl          string
	GrpcUrl          string
	LinesProviderUrl string
	LogLevel         string
	DbUrl            string
}

func setupConfig(logger loggerInterface.Logger) *config {
	updatePeriod := getEnvVariableInt("UPDATE_INTERVAL", 1, logger)
	linesProviderUrl := getEnvVariable("LINES_PROVIDER_URL", "http://localhost:8000")
	dbURL := getEnvVariable("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/lines")
	httpUrl := getEnvVariable("HTTP_URL", ":3333")
	grpcUrl := getEnvVariable("GRPC_URL", ":50051")

	return &config{
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

func setupDbConnection(dbUrl string, logger loggerInterface.Logger) commonPostgres.PgxPoolIface {
	poolConfig, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		logger.Fatal("Unable to parse DATABASE_URL", "error", err)
	}
	db, err := pgxpool.ConnectConfig(context.Background(), poolConfig)
	if err != nil {
		logger.Fatal("Unable to create connection pool", "error", err)
	}
	return db
}

type service struct {
	linesProviderAdapter adapter.LinesProviderAdapter
	conf                 *config
	sportLineRepo        *postgres.SportRepoImpl
	logger               loggerInterface.Logger
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
	lis, err := net.Listen("tcp", s.conf.GrpcUrl)
	if err != nil {
		s.logger.Fatalf("failed to listen: %v", err)
	}
	grpcSrv := grpc.NewServer()
	sportLineService := application.NewSportLineService(s.sportLineRepo)
	server := grpcServer.NewServer(sportLineService, s.logger)
	pb.RegisterKiddyLineProcessorServer(grpcSrv, server)
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
