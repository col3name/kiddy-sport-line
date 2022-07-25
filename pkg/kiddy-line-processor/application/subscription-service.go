package application

import (
	"fmt"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/util/times"
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type SubscriptionService interface {
	Subscribe(responseSender responseSender, clientId int) bool
	PushMessage(clientId int, list []commonDomain.SportType, updateIntervalInSeconds int32)
	UnsubscribeClient(clientId int)
}

type ClientSubscription struct {
	Sports map[commonDomain.SportType]float32
	Task   *time.Ticker
}

type subscriptionServiceImpl struct {
	subscriptions    map[int]*ClientSubscription
	messageQueue     *MessageQueue
	sportLineService SportLineService
	timesTicker      times.Ticker
	mu               sync.Mutex
}

func NewSubscriptionManager(sportRepo domain.SportRepo) *subscriptionServiceImpl {
	return &subscriptionServiceImpl{
		subscriptions:    make(map[int]*ClientSubscription, 0),
		messageQueue:     NewMessageQueue(),
		sportLineService: NewSportLineService(sportRepo),
		timesTicker:      times.NewTimeTicker(),
	}
}

func (s *subscriptionServiceImpl) Subscribe(responseSender responseSender, clientId int) bool {
	subMsg := s.messageQueue.Peek()
	if subMsg == nil {
		return false
	}
	if subMsg.ClientId == clientId {
		return s.addNotifySubscriberTask(responseSender, subMsg)
	}
	return true
}

func (s *subscriptionServiceImpl) PushMessage(clientId int, list []commonDomain.SportType, updateIntervalInSeconds int32) {
	s.messageQueue.Push(clientId, list, updateIntervalInSeconds)
}

func (s *subscriptionServiceImpl) UnsubscribeClient(clientId int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subscriptions[clientId]
	if !ok || (ok && sub.Task == nil) {
		return
	}
	sub.Task.Stop()
}

func (s *subscriptionServiceImpl) addNotifySubscriberTask(responseSender responseSender, subMsg *SubscriptionMessageDTO) bool {
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

func (s *subscriptionServiceImpl) addNotifySubscriberPeriodically(sender responseSender, subMsg *SubscriptionMessageDTO) {
	clientSub := s.initClientSubscription(subMsg)
	fn := s.updateSportLineFn(sender, subMsg)
	fn(false)
	clientSub.Task = s.timesTicker.Handle(subMsg.UpdateIntervalSecond, func() {
		fn(true)
	})
	s.messageQueue.Pop()
}

func (s *subscriptionServiceImpl) updateSportLineFn(sender responseSender, subMsg *SubscriptionMessageDTO) func(bool) {
	return func(isNeedDelta bool) {
		s.mu.Lock()
		subscription := s.subscriptions[subMsg.ClientId]
		s.mu.Unlock()
		line, err := s.sportLineService.Calculate(subMsg.Sports, isNeedDelta, subscription)
		if err != nil {
			log.Println(err)
			return
		}
		if err = sender.Send(line); err != nil {
			log.Println(err)
		}
	}
}

func (s *subscriptionServiceImpl) isSubChanged(clientId int, sports []commonDomain.SportType) bool {
	s.mu.Lock()
	sub, exist := s.subscriptions[clientId]
	isSubChanged := true
	if exist {
		isSubChanged = s.sportLineService.IsChanged(exist, sub.Sports, sports)
	}
	s.mu.Unlock()

	return isSubChanged
}

func (s *subscriptionServiceImpl) initClientSubscription(msg *SubscriptionMessageDTO) *ClientSubscription {
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

func (s *SportLineServiceImpl) calculateLineOfSports(lines []commonDomain.SportLine, isNeedDelta bool, subs *ClientSubscription) []*domain.Sport {
	var sportsResponse []*domain.Sport

	for _, line := range lines {
		resp := s.calculateLine(&line, isNeedDelta, subs)
		fmt.Println(resp.Type, resp.Line)
		sportsResponse = append(sportsResponse, resp)
	}
	return sportsResponse
}

func (s *SportLineServiceImpl) calculateLine(line *commonDomain.SportLine, isNeedDelta bool, subs *ClientSubscription) *domain.Sport {
	sportType := line.Type
	newScore := line.Score
	if isNeedDelta {
		newScore = newScore - subs.Sports[sportType]
	}
	subs.Sports[sportType] = line.Score
	return &domain.Sport{
		Type: sportType.String(),
		Line: newScore,
	}
}
