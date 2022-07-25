package application

import (
	"fmt"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/util/times"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type SubscriptionService interface {
	Subscribe(responseSender responseSender, clientId int) bool
	PushMessage(dto *SubscriptionMessageDTO)
	Unsubscribe(clientId int)
}

type ClientSubscription struct {
	Sports SportTypeMap
	Task   *time.Ticker
}

type subscriptionServiceImpl struct {
	subscriptions    map[int]*ClientSubscription
	messageQueue     *MessageQueue
	sportLineService SportLineService
	timesTicker      times.Ticker
	mu               sync.Mutex
}

func NewSubscriptionManager(sportLineService SportLineService) *subscriptionServiceImpl {
	return &subscriptionServiceImpl{
		subscriptions:    make(map[int]*ClientSubscription, 0),
		messageQueue:     NewMessageQueue(),
		sportLineService: sportLineService,
		timesTicker:      times.NewTimeTicker(),
	}
}

func (s *subscriptionServiceImpl) Subscribe(responseSender responseSender, clientId int) bool {
	if responseSender == nil {
		return false
	}
	subMsg := s.messageQueue.Peek()
	if subMsg == nil || (subMsg.ClientId != clientId) {
		return false
	}
	return s.addNotifySubscriberTask(responseSender, subMsg)
}

func (s *subscriptionServiceImpl) PushMessage(dto *SubscriptionMessageDTO) {
	if len(dto.Sports) == 0 || dto.ClientId < 0 || dto.UpdateIntervalSecond < 1 {
		return
	}
	s.messageQueue.Push(dto)
}

func (s *subscriptionServiceImpl) Unsubscribe(clientId int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subscriptions[clientId]
	if !ok || (ok && sub.Task == nil) {
		return
	}
	sub.Task.Stop()
	delete(s.subscriptions, clientId)
}

func (s *subscriptionServiceImpl) addNotifySubscriberTask(responseSender responseSender, subMsg *SubscriptionMessageDTO) bool {
	clientId := subMsg.ClientId
	sports := subMsg.Sports
	if len(sports) == 0 {
		s.messageQueue.Pop()
		return false
	}
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
	s.messageQueue.Pop()
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
	subToSports := make(SportTypeMap, 0)

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

func (s *sportLineServiceImpl) calculateLineOfSports(lines []*commonDomain.SportLine, isNeedDelta bool, subs *ClientSubscription) []*commonDomain.SportLine {

	for i, line := range lines {
		s.calculateLine(line, isNeedDelta, subs)
		fmt.Println(line.Type, line.Score)
		lines[i] = line
	}

	return lines
}

func (s *sportLineServiceImpl) calculateLine(line *commonDomain.SportLine, isNeedDelta bool, subs *ClientSubscription) {
	sportType := line.Type
	if isNeedDelta {
		line.Score = line.Score - subs.Sports[sportType]
	}
	subs.Sports[sportType] = line.Score
}
