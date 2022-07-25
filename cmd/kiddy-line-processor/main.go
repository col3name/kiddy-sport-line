package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	netHttp "github.com/col3name/lines/pkg/common/infrastructure/transport/net-http"
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain"
	"github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/postgres"
	grpcServer "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/transport/grpc"
	pb "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/transport/grpc/proto"
	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io/ioutil"
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

	var wg sync.WaitGroup
	wg.Add(2)
	go runHttpServer(&wg, conf.HttpUrl)
	go runGrpcServer(&wg, conf.GrpcUrl, sportLineRepo)
	go runSpotLineUpdateWorkers(sportLineRepo, conf.LinesProviderUrl, conf.UpdatePeriod)
	wg.Wait()
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

		_, err = conn.Exec(context.Background(), createSportLinesSql)
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

func runHttpServer(wg *sync.WaitGroup, serverUrl string) {
	defer wg.Done()
	http.HandleFunc("/ready", netHttp.ReadyCheckHandler)

	err := http.ListenAndServe(serverUrl, nil)
	if err != nil {
		log.Fatal(err)
		return
	}
}

func runGrpcServer(wg *sync.WaitGroup, serveUrl string, repo domain.SportRepo) {
	defer wg.Done()
	lis, err := net.Listen("tcp", serveUrl)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	server := grpcServer.NewServer(repo)
	pb.RegisterKiddyLineProcessorServer(s, server)
	log.Printf("server listening at %v", lis.Addr())
	if err = s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func runSpotLineUpdateWorkers(sportLineRepo domain.SportRepo, url string, updatePeriod int) {
	for _, sportType := range commonDomain.SupportSports {
		go pullLineWorker(sportLineRepo, url, sportType, updatePeriod)
	}
}

type BaseSport struct {
}

type BaseballResp struct {
	BaseSport
	Lines struct {
		Score string `json:"BASEBALL"`
	} `json:"lines"`
}

type FootballResp struct {
	BaseSport
	Lines struct {
		Score string `json:"FOOTBALL"`
	} `json:"lines"`
}

type SoccerResp struct {
	BaseSport
	Lines struct {
		Score string `json:"SOCCER"`
	} `json:"lines"`
}

func pullLineWorker(sportLineRepo domain.SportRepo, linesProviderUrl string, sportType commonDomain.SportType, period int) {
	for {
		url := fmt.Sprintf("%s/api/v1/lines/%s", linesProviderUrl, sportType)
		resp, err := http.Get(url)
		sleepDuration := time.Duration(period) * time.Second
		if err != nil {
			log.Error("failed get", sportType, "data", err)
			time.Sleep(sleepDuration)
			continue
		}
		sport, err := parseResp(resp, sportType)
		if err != nil {
			log.Error(err)
			time.Sleep(sleepDuration)
			continue
		}
		err = sportLineRepo.Store(sport)
		if err != nil {
			log.Error(err)
		}
		time.Sleep(sleepDuration)
	}
}

func failedGetSportError(sportType commonDomain.SportType, err error) error {
	text := "failed get " + string(sportType) + "data"
	if err != nil {
		text += err.Error()
	}
	return errors.New(text)
}

func parseResp(resp *http.Response, sportType commonDomain.SportType) (*commonDomain.SportLine, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, failedGetSportError(sportType, nil)
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return nil, failedGetSportError(sportType, err)
	}

	defer resp.Body.Close()

	return parseGetLinesResponse(bytes, sportType)
}

func parseGetLinesResponse(bytes []byte, sportType commonDomain.SportType) (*commonDomain.SportLine, error) {
	var sport commonDomain.SportLine
	var score string
	var err error
	switch sportType {
	case commonDomain.Baseball:
		var model BaseballResp
		err = json.Unmarshal(bytes, &model)
		if err != nil {
			return nil, err
		}
		sport.Type = commonDomain.Baseball
		score = model.Lines.Score
	case commonDomain.Soccer:
		var model SoccerResp
		err = json.Unmarshal(bytes, &model)
		if err != nil {
			return nil, err
		}
		sport.Type = commonDomain.Soccer
		score = model.Lines.Score
	case commonDomain.Football:
		var model FootballResp
		err = json.Unmarshal(bytes, &model)
		if err != nil {
			return nil, err
		}
		sport.Type = commonDomain.Football
		score = model.Lines.Score
	}

	if err = sport.SetScore(score); err != nil {
		return nil, err
	}
	return &sport, nil
}
