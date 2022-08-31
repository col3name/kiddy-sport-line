package application

import (
	"github.com/col3name/lines/pkg/common/application/logger"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/util/array"
	"github.com/col3name/lines/pkg/common/util/times"
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
	logger           logger.Logger
	mu               sync.Mutex
}

func NewSubscriptionManager(sportLineService SportLineService, logger logger.Logger) *subscriptionServiceImpl {
	return &subscriptionServiceImpl{
		subscriptions:    make(map[int]*ClientSubscription, 0),
		messageQueue:     NewMessageQueue(),
		sportLineService: sportLineService,
		logger:           logger,
		timesTicker:      times.NewTimeTicker(),
	}
}

func (s *subscriptionServiceImpl) Subscribe(responseSender responseSender, clientId int) bool {
	if responseSender == nil {
		return false
	}
	subMsg := s.messageQueue.Peek()
	if s.isUserAuthorOfMessage(subMsg, clientId) {
		return false
	}
	return s.addNotifySubscriberTask(responseSender, subMsg)
}

func (s *subscriptionServiceImpl) PushMessage(dto *SubscriptionMessageDTO) {
	if !s.isValidMessage(dto) {
		return
	}
	s.messageQueue.Push(dto)
}

func (s *subscriptionServiceImpl) Unsubscribe(clientId int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subscriptions[clientId]
	if s.isExistTask(ok, sub) {
		return
	}
	sub.Task.Stop()
	delete(s.subscriptions, clientId)
}

func (s *subscriptionServiceImpl) isUserAuthorOfMessage(subMsg *SubscriptionMessageDTO, clientId int) bool {
	return subMsg == nil || (subMsg.ClientId != clientId)
}

func (s *subscriptionServiceImpl) isValidMessage(dto *SubscriptionMessageDTO) bool {
	return !array.EmptyST(dto.Sports) && dto.ClientId >= 0 && dto.UpdateIntervalSecond >= 1
}

func (s *subscriptionServiceImpl) isExistTask(ok bool, sub *ClientSubscription) bool {
	return !ok || (ok && sub.Task == nil)
}

func (s *subscriptionServiceImpl) addNotifySubscriberTask(responseSender responseSender, subMessage *SubscriptionMessageDTO) bool {
	clientId := subMessage.ClientId
	sports := subMessage.Sports
	if array.EmptyST(sports) {
		s.messageQueue.Pop()
		return false
	}
	s.mu.Lock()
	sub, isExistSubTask := s.subscriptions[clientId]
	s.mu.Unlock()
	if !isExistSubTask {
		s.addNotifySubscriberPeriodically(responseSender, subMessage)
		return true
	}
	if s.isSubChanged(clientId, sports) {
		sub.Task.Stop()
		s.addNotifySubscriberPeriodically(responseSender, subMessage)
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
			s.logger.Println(err)
			return
		}
		if err = sender.Send(line); err != nil {
			s.logger.Println(err)
		}
	}
}

func (s *subscriptionServiceImpl) isSubChanged(clientId int, sports []commonDomain.SportType) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, exist := s.subscriptions[clientId]

	return !exist || exist && s.sportLineService.IsSubscriptionChanged(exist, sub.Sports, sports)
}

const DefaultScore = 1.0

func (s *subscriptionServiceImpl) initClientSubscription(msg *SubscriptionMessageDTO) *ClientSubscription {
	subToSports := make(SportTypeMap, 0)

	for _, sportType := range msg.Sports {
		subToSports[sportType] = DefaultScore
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
