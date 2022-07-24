package grpc

import (
	"fmt"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/util/times"
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain"
	pb "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/grpc/proto"
	log "github.com/sirupsen/logrus"
	"io"
	"math/rand"
	"sync"
	"time"
)

type ClientSubMsg struct {
	ClientId             int
	Sports               []commonDomain.SportType
	UpdateIntervalSecond int32
}

type ClientSub struct {
	Sports map[commonDomain.SportType]float32
	Task   *time.Ticker
}

type Server struct {
	pb.UnimplementedKiddyLineProcessorServer
	mu                sync.Mutex
	clientSubs        map[int]*ClientSub
	clientSubMsgQueue []*ClientSubMsg
	sportRepo         domain.SportRepo
}

func parseSportRequest(sports []string) []commonDomain.SportType {
	res := make([]commonDomain.SportType, 0)
	for _, sport := range sports {
		switch sport {
		case string(commonDomain.Football):
			res = append(res, commonDomain.Football)
		case string(commonDomain.Baseball):
			res = append(res, commonDomain.Baseball)
		case string(commonDomain.Soccer):
			res = append(res, commonDomain.Soccer)
		}
	}
	return res
}

func (s *Server) SubscribeOnSportsLines(stream pb.KiddyLineProcessor_SubscribeOnSportsLinesServer) error {
	errorsCh := make(chan error)
	clientUniqueCode := rand.Intn(1e6)

	go s.receiveSubscriptions(stream, clientUniqueCode, errorsCh)
	go s.sendSubData(stream, clientUniqueCode)

	return <-errorsCh
}

func (s *Server) receiveSubscriptions(stream pb.KiddyLineProcessor_SubscribeOnSportsLinesServer, clientId int, errCh chan error) {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			log.Println(err)
			s.unsubscribeClient(clientId)
			errCh <- err
		}
		if err != nil {
			log.Printf("Error in receiving message from client :: %v", err)
			errCh <- err
			s.unsubscribeClient(clientId)
			continue
		}
		sportsList := parseSportRequest(in.Sports)
		if in.IntervalInSecond < 1 {
			log.Printf("Error in receiving message from client. interval must be positive number :: %v", err)
			continue
		}
		s.mu.Lock()
		s.clientSubMsgQueue = append(s.clientSubMsgQueue, &ClientSubMsg{
			ClientId:             clientId,
			Sports:               sportsList,
			UpdateIntervalSecond: in.IntervalInSecond,
		})
		s.mu.Unlock()
	}
}

func (s *Server) sendSubData(stream pb.KiddyLineProcessor_SubscribeOnSportsLinesServer, clientId int) {
	for {
		for {
			s.mu.Lock()
			if len(s.clientSubMsgQueue) == 0 {
				s.mu.Unlock()
				break
			}

			subMsg := s.clientSubMsgQueue[0]
			s.mu.Unlock()
			if subMsg.ClientId == clientId {
				s.sendResponse(stream, subMsg)
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (s *Server) isSubChanged(clientId int, sports []commonDomain.SportType) bool {
	isSubChanged := false
	s.mu.Lock()
	sub, exist := s.clientSubs[clientId]
	if exist {
		if len(sub.Sports) != len(sports) {
			isSubChanged = true
		} else {
			for _, sportType := range sports {
				_, ok := sub.Sports[sportType]
				if !ok {
					isSubChanged = true
					break
				}
			}
		}
	} else {
		isSubChanged = true
	}
	s.mu.Unlock()

	return isSubChanged
}

func (s *Server) initClientSub(clientId int, msg *ClientSubMsg) *ClientSub {
	subToSports := make(map[commonDomain.SportType]float32, 0)
	for _, sportType := range msg.Sports {
		subToSports[sportType] = 1.0
	}

	sub := &ClientSub{
		Sports: subToSports,
		Task:   nil,
	}
	s.mu.Lock()
	s.clientSubs[clientId] = sub
	s.mu.Unlock()
	return sub
}

func (s *Server) calcSportsLine(lines []commonDomain.SportLine, isNeedDelta bool, clientId int) []*pb.Sport {
	var sportsResponse []*pb.Sport

	for _, line := range lines {
		sportType := line.Type
		newScore := line.Score
		s.mu.Lock()
		if isNeedDelta {
			newScore = s.clientSubs[clientId].Sports[sportType] - newScore
		}
		s.clientSubs[clientId].Sports[sportType] = line.Score
		s.mu.Unlock()
		resp := &pb.Sport{
			Type: sportType.String(),
			Line: newScore,
		}
		fmt.Println(resp.Type, resp.Line)
		sportsResponse = append(sportsResponse, resp)
	}
	return sportsResponse
}

func (s *Server) popSubMsgQueue() {
	s.mu.Lock()
	if len(s.clientSubMsgQueue) > 0 {
		s.clientSubMsgQueue = s.clientSubMsgQueue[1:]
	}
	s.mu.Unlock()
}

func (s *Server) unsubscribeClient(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.clientSubs[id]
	if !ok {
		return
	}
	if sub.Task == nil {
		return
	}
	sub.Task.Stop()
}

func (s *Server) sendResponse(stream pb.KiddyLineProcessor_SubscribeOnSportsLinesServer, subMsg *ClientSubMsg) {
	clientId := subMsg.ClientId
	isNeedDelta := false
	updateSportLineFn := func() {
		s.mu.Lock()
		sportLines, err := s.sportRepo.GetSportLines(subMsg.Sports)
		if err != nil {
			s.mu.Unlock()
			log.Println(err)
		} else {
			s.mu.Unlock()
			sportsResponse := s.calcSportsLine(sportLines, isNeedDelta, clientId)
			response := pb.SubscribeResponse{Sports: sportsResponse}
			if err = stream.Send(&response); err != nil {
				log.Println(err)
			}
		}
	}

	sports := subMsg.Sports
	s.mu.Lock()
	sub, isExistSubTask := s.clientSubs[subMsg.ClientId]
	s.mu.Unlock()
	if !isExistSubTask {
		fmt.Println("first sub")
		clientSub := s.initClientSub(clientId, subMsg)
		updateSportLineFn()
		isNeedDelta = true
		clientSub.Task = times.TickerHandle(subMsg.UpdateIntervalSecond, updateSportLineFn)
		s.popSubMsgQueue()
	} else {
		isSubChanged := s.isSubChanged(clientId, sports)
		if isSubChanged {
			fmt.Println("change sub")
			sub.Task.Stop()

			clientSub := s.initClientSub(clientId, subMsg)
			updateSportLineFn()
			isNeedDelta = true
			clientSub.Task = times.TickerHandle(subMsg.UpdateIntervalSecond, updateSportLineFn)
			s.popSubMsgQueue()
		}
	}
}

func NewServer(sportRepo domain.SportRepo) *Server {
	return &Server{
		clientSubs:        make(map[int]*ClientSub, 0),
		clientSubMsgQueue: make([]*ClientSubMsg, 0),
		sportRepo:         sportRepo,
	}
}
