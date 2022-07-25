package grpc

import (
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application"
	pb "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/transport/grpc/proto"
	log "github.com/sirupsen/logrus"
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
}

func NewServer(sportLineService application.SportLineService) *Server {
	return &Server{
		subscriptionManager: application.NewSubscriptionManager(sportLineService),
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
			log.Println(err)
			s.subscriptionManager.Unsubscribe(clientId)
			errCh <- err
		}
		if err != nil {
			log.Printf("Error in receiving message from client :: %v", err)
			errCh <- err
			s.subscriptionManager.Unsubscribe(clientId)
			continue
		}
		if in.IntervalInSecond < 1 || len(in.Sports) == 0 {
			log.Printf("Error in receiving message from client. interval must be positive number :: %v", err)
			errCh <- err
			continue
		}
		sportsList := parseSportRequest(in.Sports)
		if len(sportsList) == 0 {
			log.Println("Error in receiving message from client. :: ")
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
