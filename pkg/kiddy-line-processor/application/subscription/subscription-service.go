package subscription

import (
	"github.com/col3name/lines/pkg/common/application/logger"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/util/array"
	"github.com/col3name/lines/pkg/common/util/times"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application/service"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application/sport-line"
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain/model"
	"sync"
)

type Service interface {
	Subscribe(responseSender service.ResponseSenderService, clientId int) bool
	PushMessage(dto *MessageToSubscribeDTO)
	Unsubscribe(clientId int)
}

type subscriptionServiceImpl struct {
	subscriptions    map[int]*model.ClientSubscription
	messageQueue     *MessageQueue
	sportLineService sport_line.SportLineService
	timesTicker      times.Ticker
	logger           logger.Logger
	mu               sync.Mutex
}

func NewSubscriptionManager(sportLineService sport_line.SportLineService, logger logger.Logger) *subscriptionServiceImpl {
	return &subscriptionServiceImpl{
		subscriptions:    make(map[int]*model.ClientSubscription, 0),
		messageQueue:     NewMessageQueue(),
		sportLineService: sportLineService,
		logger:           logger,
		timesTicker:      times.NewTimeTicker(),
	}
}

func (s *subscriptionServiceImpl) Subscribe(responseSender service.ResponseSenderService, clientId int) bool {
	if responseSender == nil {
		return false
	}
	subMsg := s.messageQueue.Peek()
	if s.isUserAuthorOfMessage(subMsg, clientId) {
		return false
	}
	return s.addNotifySubscriberTask(responseSender, subMsg)
}

func (s *subscriptionServiceImpl) PushMessage(dto *MessageToSubscribeDTO) {
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

func (s *subscriptionServiceImpl) isUserAuthorOfMessage(subMsg *MessageToSubscribeDTO, clientId int) bool {
	return subMsg == nil || (subMsg.ClientId != clientId)
}

func (s *subscriptionServiceImpl) isValidMessage(dto *MessageToSubscribeDTO) bool {
	return !array.EmptyST(dto.Sports) && dto.ClientId >= 0 && dto.UpdateIntervalSecond >= 1
}

func (s *subscriptionServiceImpl) isExistTask(ok bool, sub *model.ClientSubscription) bool {
	return !ok || (ok && sub.Task == nil)
}

func (s *subscriptionServiceImpl) addNotifySubscriberTask(responseSender service.ResponseSenderService, subMessage *MessageToSubscribeDTO) bool {
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

func (s *subscriptionServiceImpl) addNotifySubscriberPeriodically(sender service.ResponseSenderService, subMsg *MessageToSubscribeDTO) {
	clientSub := s.initClientSubscription(subMsg)
	fn := s.updateSportLineFn(sender, subMsg)
	fn(false)
	clientSub.Task = s.timesTicker.Handle(subMsg.UpdateIntervalSecond, func() {
		fn(true)
	})
	s.messageQueue.Pop()
}

func (s *subscriptionServiceImpl) updateSportLineFn(sender service.ResponseSenderService, subMsg *MessageToSubscribeDTO) func(bool) {
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

func (s *subscriptionServiceImpl) initClientSubscription(msg *MessageToSubscribeDTO) *model.ClientSubscription {
	subToSports := make(model.SportTypeMap, 0)

	for _, sportType := range msg.Sports {
		subToSports[sportType] = DefaultScore
	}

	sub := &model.ClientSubscription{Sports: subToSports, Task: nil}

	s.mu.Lock()
	s.subscriptions[msg.ClientId] = sub
	s.mu.Unlock()

	return sub
}
