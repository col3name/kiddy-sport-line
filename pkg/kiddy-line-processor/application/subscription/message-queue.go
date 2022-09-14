package subscription

import (
	"sync"
)

type MessageQueue struct {
	mu   sync.Mutex
	data []*MessageToSubscribeDTO
}

func NewMessageQueue() *MessageQueue {
	return &MessageQueue{
		mu:   sync.Mutex{},
		data: make([]*MessageToSubscribeDTO, 0),
	}
}

func (s *MessageQueue) Push(dto *MessageToSubscribeDTO) {
	s.mu.Lock()
	s.data = append(s.data, dto)
	s.mu.Unlock()
}

func (s *MessageQueue) Pop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.Empty() {
		s.data = s.data[1:]
	}
}

func (s *MessageQueue) Peek() *MessageToSubscribeDTO {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Empty() {
		return nil
	}
	return s.data[0]
}

func (s *MessageQueue) Empty() bool {
	return s.Size() == 0
}
func (s *MessageQueue) Size() int {
	return len(s.data)
}
