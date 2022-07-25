package application

import (
	"fmt"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/util/times"
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain"
	pb "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/grpc/proto"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type ClientSubscription struct {
	Sports map[commonDomain.SportType]float32
	Task   *time.Ticker
	Stream pb.KiddyLineProcessor_SubscribeOnSportsLinesServer
}

type SubscriptionService struct {
	subscriptions map[int]*ClientSubscription
	messageQueue  *MessageQueue
	sportRepo     domain.SportRepo
	sender        responseSender
	mu            sync.Mutex
}

func NewSubscriptionManager(sportRepo domain.SportRepo) *SubscriptionService {
	return &SubscriptionService{
		subscriptions: make(map[int]*ClientSubscription, 0),
		sportRepo:     sportRepo,
		messageQueue:  NewMessageQueue(),
	}
}

func (s *SubscriptionService) Subscribe(responseSender responseSender, clientId int) bool {
	subMsg := s.messageQueue.Peek()
	if subMsg == nil {
		return false
	}
	if subMsg.ClientId == clientId {
		return s.addNotifySubscriberTask(responseSender, subMsg)
	}
	return true
}

func (s *SubscriptionService) PushMessage(clientId int, list []commonDomain.SportType, updateIntervalInSeconds int32) {
	s.messageQueue.Push(clientId, list, updateIntervalInSeconds)
}

func (s *SubscriptionService) UnsubscribeClient(clientId int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subscriptions[clientId]
	if !ok || (ok && sub.Task == nil) {
		return
	}
	sub.Task.Stop()
}

func (s *SubscriptionService) addNotifySubscriberTask(responseSender responseSender, subMsg *SubscriptionMessageDTO) bool {
	clientId := subMsg.ClientId
	sports := subMsg.Sports
	s.mu.Lock()
	sub, isExistSubTask := s.subscriptions[clientId]
	s.mu.Unlock()
	if !isExistSubTask {
		fmt.Println("first sub")
		s.addNotifySubscriberPeriodically(responseSender, subMsg)
		return true
	}
	if s.isSubChanged(clientId, sports) {
		fmt.Println("change sub")
		sub.Task.Stop()
		s.addNotifySubscriberPeriodically(responseSender, subMsg)
		return true
	}
	return false
}

func (s *SubscriptionService) addNotifySubscriberPeriodically(sender responseSender, subMsg *SubscriptionMessageDTO) {
	clientSub := s.initClientSubscription(subMsg)
	fn := s.updateSportLineFn(sender, subMsg)
	fn(false)
	clientSub.Task = times.TickerHandle(subMsg.UpdateIntervalSecond, func() {
		fn(true)
	})
	s.messageQueue.Pop()
}

func (s *SubscriptionService) updateSportLineFn(sender responseSender, subMsg *SubscriptionMessageDTO) func(bool) {
	return func(isNeedDelta bool) {
		s.mu.Lock()
		sportLines, err := s.sportRepo.GetSportLines(subMsg.Sports)
		if err != nil {
			s.mu.Unlock()
			log.Println(err)
			return
		}
		s.mu.Unlock()
		line := s.calculateLineOfSports(sportLines, isNeedDelta, subMsg.ClientId)

		if err = sender.Send(line); err != nil {
			log.Println(err)
		}
	}
}

func (s *SubscriptionService) isSubChanged(clientId int, sports []commonDomain.SportType) bool {
	isSubChanged := false
	s.mu.Lock()
	sub, exist := s.subscriptions[clientId]
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

func (s *SubscriptionService) initClientSubscription(msg *SubscriptionMessageDTO) *ClientSubscription {
	subToSports := make(map[commonDomain.SportType]float32, 0)

	for _, sportType := range msg.Sports {
		subToSports[sportType] = 1.0
	}

	sub := &ClientSubscription{
		Sports: subToSports,
		Task:   nil,
	}
	s.mu.Lock()
	s.subscriptions[msg.ClientId] = sub
	s.mu.Unlock()
	return sub
}

func (s *SubscriptionService) calculateLineOfSports(lines []commonDomain.SportLine, isNeedDelta bool, clientId int) []*Sport {
	var sportsResponse []*Sport

	for _, line := range lines {
		resp := s.calculateLine(&line, isNeedDelta, clientId)
		fmt.Println(resp.Type, resp.Line)
		sportsResponse = append(sportsResponse, resp)
	}
	return sportsResponse
}

func (s *SubscriptionService) calculateLine(line *commonDomain.SportLine, isNeedDelta bool, clientId int) *Sport {
	sportType := line.Type
	newScore := line.Score
	s.mu.Lock()
	if isNeedDelta {
		newScore = s.subscriptions[clientId].Sports[sportType] - newScore
	}
	s.subscriptions[clientId].Sports[sportType] = line.Score
	s.mu.Unlock()
	return &Sport{
		Type: sportType.String(),
		Line: newScore,
	}
}
