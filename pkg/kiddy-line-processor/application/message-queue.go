package application

import (
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

func (s *MessageQueue) Push(dto *SubscriptionMessageDTO) {
	s.mu.Lock()
	s.clientSubMsgQueue = append(s.clientSubMsgQueue, dto)
	s.mu.Unlock()
}

func (s *MessageQueue) Pop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.Empty() {
		s.clientSubMsgQueue = s.clientSubMsgQueue[1:]
	}
}

func (s *MessageQueue) Peek() *SubscriptionMessageDTO {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Empty() {
		return nil
	}
	return s.clientSubMsgQueue[0]
}

func (s *MessageQueue) Empty() bool {
	return s.Size() == 0
}
func (s *MessageQueue) Size() int {
	return len(s.clientSubMsgQueue)
}
