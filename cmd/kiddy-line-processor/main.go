package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	netHttp "github.com/col3name/lines/pkg/common/infrastructure/transport/net-http"
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain"
	grpcServer "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/grpc"
	pb "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/grpc/proto"
	"github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/postgres"
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
	config := newConfig()
	db := setupDb(config.DbUrl)
	sportLineRepo := postgres.NewSportLineRepository(db)
	var wg sync.WaitGroup
	wg.Add(3)
	go runHttpServer(&wg, config.HttpUrl)
	go runGrpcServer(&wg, config.GrpcUrl, sportLineRepo)
	//go runSpotLineUpdateWorkers(&wg, sportLineRepo, config.LinesProviderUrl, config.UpdatePeriod)
	wg.Wait()
}

type config struct {
	UpdatePeriod     int
	HttpUrl          string
	GrpcUrl          string
	LinesProviderUrl string
	LogLevel         string
	DbUrl            string
}

func newConfig() *config {
	updatePeriod := 1
	nStr := os.Getenv("N")
	if len(nStr) > 0 {
		val, err := strconv.Atoi(nStr)
		errPositive := "N must be positive integer"
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

	return &config{
		UpdatePeriod:     updatePeriod,
		HttpUrl:          ":3333",
		GrpcUrl:          ":50051",
		LinesProviderUrl: linesProviderUrl,
		DbUrl:            dbURL,
		LogLevel:         "",
	}
}

func setupDb(dbUrl string) *pgxpool.Pool {
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

func runSpotLineUpdateWorkers(wg *sync.WaitGroup, sportLineRepo domain.SportRepo, url string, updatePeriod int) {
	for _, sportType := range commonDomain.SupportSports {
		wg.Add(1)
		go pullLineWorker(wg, sportLineRepo, url, sportType, updatePeriod)
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
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
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

func pullLineWorker(wg *sync.WaitGroup, sportLineRepo domain.SportRepo, linesProviderUrl string, sportType commonDomain.SportType, period int) {
	for {
		url := fmt.Sprintf("%s/api/v1/lines/%s", linesProviderUrl, sportType)
		fmt.Println(url)
		resp, err := http.Get(url)
		if err != nil {
			log.Error("failed get", sportType, "data", err)
		} else {
			sport, err := parseResp(resp, sportType)
			if err != nil {
				log.Error(err)
			} else {
				err = sportLineRepo.Store(sport)
				if err != nil {
					log.Error(err)
				}
			}
		}
		time.Sleep(time.Duration(period) * time.Second)
	}
	wg.Done()
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
	fmt.Println(string(bytes))
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
