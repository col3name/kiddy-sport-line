package grpc

import (
	"github.com/col3name/lines/pkg/common/application/logger"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application"
	pb "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/transport/grpc/proto"
	"io"
	"math/rand"
	"time"
)

func parseSportRequest(sports []string) []commonDomain.SportType {
	res := make([]commonDomain.SportType, 0)

	for _, sportType := range sports {
		if val, err := commonDomain.NewSportType(sportType); err == nil {
			res = append(res, val)
		}
	}

	return res
}

type Server struct {
	pb.UnimplementedKiddyLineProcessorServer
	subscriptionManager application.SubscriptionService
	logger              logger.Logger
}

func NewServer(sportLineService application.SportLineService, logger logger.Logger) *Server {
	return &Server{
		subscriptionManager: application.NewSubscriptionManager(sportLineService, logger),
		logger:              logger,
	}
}

func (s *Server) SubscribeOnSportsLines(stream pb.KiddyLineProcessor_SubscribeOnSportsLinesServer) error {
	errorsCh := make(chan error)
	clientUniqueCode := rand.Intn(1e6)

	go s.receiveSubscriptions(stream, clientUniqueCode, errorsCh)
	go s.sendDataToSubscribers(stream, clientUniqueCode)

	return <-errorsCh
}

func (s *Server) receiveSubscriptions(stream pb.KiddyLineProcessor_SubscribeOnSportsLinesServer, clientId int, errCh chan error) {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			s.logger.Println(err)
			s.subscriptionManager.Unsubscribe(clientId)
			errCh <- err
		}
		if err != nil {
			s.logger.Println("Error in receiving message from client :: ", err)
			errCh <- err
			s.subscriptionManager.Unsubscribe(clientId)
			continue
		}
		if in.IntervalInSecond < 1 || len(in.Sports) == 0 {
			s.logger.Println("Error in receiving message from client. interval must be positive number :: ", err)
			errCh <- err
			continue
		}
		sportsList := parseSportRequest(in.Sports)
		if len(sportsList) == 0 {
			s.logger.Println("Error in receiving message from client. :: ")
			errCh <- err
			continue
		}
		s.subscriptionManager.PushMessage(&application.SubscriptionMessageDTO{
			ClientId:             clientId,
			Sports:               sportsList,
			UpdateIntervalSecond: in.IntervalInSecond,
		})
	}
}

func (s *Server) sendDataToSubscribers(stream pb.KiddyLineProcessor_SubscribeOnSportsLinesServer, clientId int) {
	for {
		for {
			sender := &ResponseSenderGrpc{Stream: stream}
			if ok := s.subscriptionManager.Subscribe(sender, clientId); !ok {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}
