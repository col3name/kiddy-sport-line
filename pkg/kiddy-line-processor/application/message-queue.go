package application

import (
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"sync"
)

type MessageQueue struct {
	mu                sync.Mutex
	clientSubMsgQueue []*SubscriptionMessageDTO
}

func NewMessageQueue() *MessageQueue {
	return &MessageQueue{
		mu:                sync.Mutex{},
		clientSubMsgQueue: make([]*SubscriptionMessageDTO, 0),
	}
}

func (s *MessageQueue) Push(clientId int, sportsList []commonDomain.SportType, updateIntervalInSeconds int32) {
	s.mu.Lock()
	s.clientSubMsgQueue = append(s.clientSubMsgQueue, &SubscriptionMessageDTO{
		ClientId:             clientId,
		Sports:               sportsList,
		UpdateIntervalSecond: updateIntervalInSeconds,
	})
	s.mu.Unlock()
}

func (s *MessageQueue) Pop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Empty() {
		s.clientSubMsgQueue = s.clientSubMsgQueue[1:]
	}
}

func (s *MessageQueue) Peek() *SubscriptionMessageDTO {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Empty() {
		return s.clientSubMsgQueue[0]
	}
	return nil
}

func (s *MessageQueue) Empty() bool {
	return s.Size() > 0
}
func (s *MessageQueue) Size() int {
	return len(s.clientSubMsgQueue)
}
