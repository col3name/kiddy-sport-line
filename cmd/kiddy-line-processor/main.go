package main

import (
	"context"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	netHttp "github.com/col3name/lines/pkg/common/infrastructure/transport/net-http"
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain"
	"github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/adapter"
	"github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/postgres"
	grpcServer "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/transport/grpc"
	pb "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/transport/grpc/proto"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

func main() {
	conf := setupConfig()
	conn := setupDbConnection(conf.DbUrl)
	sportLineRepo := postgres.NewSportLineRepository(conn)
	performDbMigrationIfNeeded(sportLineRepo, conn)

	s := &service{
		linesProviderAdapter: adapter.NewLinesProviderAdapter(conf.LinesProviderUrl),
		conf:                 conf,
		sportLineRepo:        sportLineRepo,
	}
	s.Run()
}

func performDbMigrationIfNeeded(sportLineRepo domain.SportRepo, conn *pgxpool.Pool) {
	_, err := sportLineRepo.GetSportLines([]commonDomain.SportType{commonDomain.Baseball})
	if err != nil {
		if err != postgres.ErrTableNotExist {
			log.Fatal(err)
		}

		createSportLinesSql := `BEGIN TRANSACTION;
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

		cancelFunc, err := postgres.WithTx(conn, func(tx pgx.Tx) error {
			_, err = tx.Exec(context.Background(), createSportLinesSql)
			return err
		})
		if cancelFunc != nil {
			defer cancelFunc()
		}
		if err != nil {
			log.Fatal(err)
		}
	}
}

type config struct {
	UpdatePeriod     int
	HttpUrl          string
	GrpcUrl          string
	LinesProviderUrl string
	LogLevel         string
	DbUrl            string
}

func setupConfig() *config {
	updatePeriod := 1
	nStr := os.Getenv("UPDATE_INTERVAL")
	if len(nStr) > 0 {
		val, err := strconv.Atoi(nStr)
		errPositive := "UPDATE_INTERVAL must be positive integer"
		if err != nil {
			log.Error(errPositive)
		}
		if val < 1 {
			log.Error(errPositive)
		}
	}
	linesProviderUrl := os.Getenv("LINES_PROVIDER_URL")
	if len(linesProviderUrl) == 0 {
		linesProviderUrl = "http://localhost:8000"
	}
	dbURL := os.Getenv("DATABASE_URL")
	if len(dbURL) == 0 {
		dbURL = "postgres://postgres:postgres@localhost:5432/lines"
	}
	httpUrl := os.Getenv("HTTP_URL")
	if len(httpUrl) == 0 {
		httpUrl = ":3333"
	}
	grpcUrl := os.Getenv("GRPC_URL")
	if len(grpcUrl) == 0 {
		grpcUrl = ":50051"
	}

	return &config{
		UpdatePeriod:     updatePeriod,
		HttpUrl:          httpUrl,
		GrpcUrl:          grpcUrl,
		LinesProviderUrl: linesProviderUrl,
		DbUrl:            dbURL,
		LogLevel:         "",
	}
}

func setupDbConnection(dbUrl string) *pgxpool.Pool {
	poolConfig, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		log.Fatal("Unable to parse DATABASE_URL", "error", err)
	}
	db, err := pgxpool.ConnectConfig(context.Background(), poolConfig)
	if err != nil {
		log.Fatal("Unable to create connection pool", "error", err)
	}
	return db
}

type service struct {
	linesProviderAdapter adapter.LinesProviderAdapter
	conf                 *config
	sportLineRepo        *postgres.SportRepoImpl
}

func (s *service) runHttpServer(wg *sync.WaitGroup) {
	defer wg.Done()
	http.HandleFunc("/ready", netHttp.ReadyCheckHandler)

	err := http.ListenAndServe(s.conf.HttpUrl, nil)
	if err != nil {
		log.Fatal(err)
		return
	}
}

func (s *service) runGrpcServer(wg *sync.WaitGroup) {
	defer wg.Done()
	lis, err := net.Listen("tcp", s.conf.GrpcUrl)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcSrv := grpc.NewServer()
	server := grpcServer.NewServer(s.sportLineRepo)
	pb.RegisterKiddyLineProcessorServer(grpcSrv, server)
	log.Printf("server listening at %v", lis.Addr())
	if err = grpcSrv.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
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
		sportLine, err := s.linesProviderAdapter.GetLines(sportType)
		if err != nil {
			log.Error(err)
			time.Sleep(sleepDuration)
			continue
		}
		err = s.sportLineRepo.Store(sportLine)
		if err != nil {
			log.Error(err)
		}
		time.Sleep(sleepDuration)
	}
}

func (s *service) Run() {
	var wg sync.WaitGroup
	wg.Add(2)
	go s.runHttpServer(&wg)
	go s.runGrpcServer(&wg)
	go s.runSpotLineUpdateWorkers()
	wg.Wait()
}
