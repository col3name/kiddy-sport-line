package application

import (
	"errors"
	"github.com/col3name/lines/pkg/common/domain"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
	"time"
)

type expectedPushMessage struct {
	queueSize int
	msg       *SubscriptionMessageDTO
}

func getFieldValue(interf interface{}, fieldName string) *reflect.Value {
	val := reflect.ValueOf(interf).Elem()
	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)

		if typeField.Name == fieldName {
			valueField := val.Field(i)
			return &valueField
		}
	}
	return nil
}

func equalSubscriptionMessageDTO(t *testing.T, lhs, rhs *SubscriptionMessageDTO) {
	assert.Equal(t, lhs.UpdateIntervalSecond, rhs.UpdateIntervalSecond)
	assert.Equal(t, lhs.ClientId, rhs.ClientId)
	assert.Equal(t, len(lhs.Sports), len(rhs.Sports))
	for i, sport := range rhs.Sports {
		assert.Equal(t, lhs.Sports[i], sport)
	}
}

func TestPushMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    *SubscriptionMessageDTO
		expected *expectedPushMessage
	}{
		{
			name: "empty sports",
			input: &SubscriptionMessageDTO{
				ClientId:             1,
				Sports:               []domain.SportType{},
				UpdateIntervalSecond: 1,
			},
			expected: &expectedPushMessage{queueSize: 0},
		},
		{
			name: "invalid client id",
			input: &SubscriptionMessageDTO{
				ClientId:             0,
				Sports:               []domain.SportType{},
				UpdateIntervalSecond: 1,
			},
			expected: &expectedPushMessage{queueSize: 0},
		},
		{
			name: "negative client id",
			input: &SubscriptionMessageDTO{
				ClientId:             -1,
				Sports:               []domain.SportType{},
				UpdateIntervalSecond: 1,
			},
			expected: &expectedPushMessage{queueSize: 0},
		},
		{
			name: "update interval < 1",
			input: &SubscriptionMessageDTO{
				ClientId:             -1,
				Sports:               []domain.SportType{},
				UpdateIntervalSecond: 1,
			},
			expected: &expectedPushMessage{queueSize: 0},
		},
		{
			name: "update interval < 1",
			input: &SubscriptionMessageDTO{
				ClientId:             -1,
				Sports:               []domain.SportType{},
				UpdateIntervalSecond: 1,
			},
			expected: &expectedPushMessage{queueSize: 0},
		},
		{
			name: "valid sub message",
			input: &SubscriptionMessageDTO{
				ClientId:             1,
				Sports:               []domain.SportType{domain.Baseball},
				UpdateIntervalSecond: 1,
			},
			expected: &expectedPushMessage{queueSize: 1,
				msg: &SubscriptionMessageDTO{
					ClientId:             1,
					Sports:               []domain.SportType{domain.Baseball},
					UpdateIntervalSecond: 1,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := NewSubscriptionManager(&MockLinesService{
				FakeCalculate: nil,
				FakeIsChanged: nil,
			})
			manager.PushMessage(test.input)
			assert.Equal(t, test.expected.queueSize, manager.messageQueue.Size())
			if test.expected.queueSize > 0 {
				peek := manager.messageQueue.Peek()
				msg := test.expected.msg
				equalSubscriptionMessageDTO(t, msg, peek)
			}
		})
	}
}

type MockLinesService struct {
	FakeCalculate func(sports []domain.SportType, isNeedDelta bool, subs *ClientSubscription) ([]*domain.SportLine, error)
	FakeIsChanged func(exist bool, subscriptionMap SportTypeMap, newValue []domain.SportType) bool
}

func (m *MockLinesService) Calculate(sports []domain.SportType, isNeedDelta bool, subs *ClientSubscription) ([]*domain.SportLine, error) {
	if m.FakeCalculate == nil {
		return nil, nil
	}
	return m.FakeCalculate(sports, isNeedDelta, subs)
}

func (m *MockLinesService) IsChanged(exist bool, subscriptionMap SportTypeMap, subscribeToSports []domain.SportType) bool {
	if m.FakeIsChanged == nil {
		return false
	}
	return m.FakeIsChanged(exist, subscriptionMap, subscribeToSports)
}

type inputUnsubscribeClient struct {
	subscriptions map[int]*ClientSubscription
	clientId      int
}

type expectedUnsubscribeClient struct {
	exist             bool
	subscriptionsSize int
	subscription      *ClientSubscription
}

func TestUnsubscribeClient(t *testing.T) {
	tests := []struct {
		name     string
		input    *inputUnsubscribeClient
		expected *expectedUnsubscribeClient
	}{
		{
			name: "not exist client",
			input: &inputUnsubscribeClient{
				subscriptions: map[int]*ClientSubscription{},
				clientId:      1,
			},
			expected: &expectedUnsubscribeClient{
				exist:             false,
				subscriptionsSize: 0,
				subscription:      nil,
			},
		},
		{
			name: "not exist client",
			input: &inputUnsubscribeClient{
				subscriptions: map[int]*ClientSubscription{
					1: {
						Sports: map[domain.SportType]float32{domain.Baseball: 1.0},
						Task:   time.NewTicker(1),
					},
					2: {
						Sports: map[domain.SportType]float32{domain.Baseball: 1.0},
						Task:   time.NewTicker(1),
					},
				},
				clientId: 1,
			},
			expected: &expectedUnsubscribeClient{
				exist:             false,
				subscriptionsSize: 1,
				subscription:      nil,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := NewSubscriptionManager(&MockLinesService{
				FakeCalculate: nil,
				FakeIsChanged: nil,
			})
			input := test.input
			expected := test.expected

			manager.subscriptions = input.subscriptions
			manager.Unsubscribe(input.clientId)
			subscription, ok := manager.subscriptions[input.clientId]

			assert.Equal(t, expected.exist, ok)
			assert.Equal(t, expected.subscriptionsSize, len(manager.subscriptions))
			expectSub := expected.subscription
			if expectSub == nil {
				assert.True(t, subscription == nil)
			} else {
				assert.Equal(t, expectSub, subscription)
				if expectSub.Task != nil {
					assert.True(t, subscription.Task != nil)
				} else {
					assert.True(t, subscription.Task == nil)
				}

				assert.Equal(t, len(expectSub.Sports), len(subscription.Sports))
				for sportType, line := range subscription.Sports {
					expectedLine, ok := expectSub.Sports[sportType]
					assert.True(t, ok)
					assert.Equal(t, expectedLine, line)
				}
			}
		})
	}
}

type mockResponseSender struct {
	Called    bool
	FakeSend  func(sports []*domain.SportLine) error
	CountCall int
}

func (m *mockResponseSender) Send(sports []*domain.SportLine) error {
	if m.FakeSend == nil {
		return nil
	}
	m.Called = true
	m.CountCall++
	return m.FakeSend(sports)
}

type inputSubscribe struct {
	clientId         int
	responseSender   responseSender
	messageQueue     *MessageQueue
	sportLineService SportLineService
	subscriptions    map[int]*ClientSubscription
}

type expectedSubscribe struct {
	messageQueueSize        int
	success                 bool
	messageQueue            *MessageQueue
	responseSenderCalled    bool
	subscriptions           map[int]*ClientSubscription
	responseSenderCountCall int64
}

func TestSubscribe(t *testing.T) {
	tests := []struct {
		name     string
		input    *inputSubscribe
		expected *expectedSubscribe
	}{
		{
			name: "response sender nil",
			input: &inputSubscribe{
				clientId:         1,
				responseSender:   nil,
				sportLineService: nil,
				messageQueue:     nil,
			},
			expected: &expectedSubscribe{
				messageQueueSize:     0,
				success:              false,
				responseSenderCalled: false,
			},
		},
		{
			name: "empty message queue",
			input: &inputSubscribe{
				clientId:         1,
				responseSender:   nil,
				sportLineService: nil,
				messageQueue:     nil,
			},
			expected: &expectedSubscribe{
				messageQueueSize:     0,
				success:              false,
				responseSenderCalled: false,
			},
		},
		{
			name: "empty sport list for subscribe",
			input: &inputSubscribe{
				clientId: 1,
				responseSender: &mockResponseSender{FakeSend: func(sports []*domain.SportLine) error {
					return nil
				}},
				sportLineService: &MockLinesService{
					FakeCalculate: nil,
					FakeIsChanged: nil,
				},
				messageQueue: &MessageQueue{
					clientSubMsgQueue: []*SubscriptionMessageDTO{
						{ClientId: 1, Sports: []domain.SportType{}, UpdateIntervalSecond: 1},
					},
				},
			},
			expected: &expectedSubscribe{
				messageQueueSize:     0,
				success:              false,
				responseSenderCalled: false,
				messageQueue: &MessageQueue{
					clientSubMsgQueue: []*SubscriptionMessageDTO{},
				},
				subscriptions: map[int]*ClientSubscription{},
			},
		},
		{
			name: "subscription client id != parameter clientId",
			input: &inputSubscribe{
				clientId: 1,
				responseSender: &mockResponseSender{FakeSend: func(sports []*domain.SportLine) error {
					return nil
				}},
				sportLineService: &MockLinesService{
					FakeCalculate: nil,
					FakeIsChanged: nil,
				},
				messageQueue: &MessageQueue{
					clientSubMsgQueue: []*SubscriptionMessageDTO{
						{ClientId: 2, Sports: []domain.SportType{domain.Baseball}, UpdateIntervalSecond: 1},
					},
				},
			},
			expected: &expectedSubscribe{
				messageQueueSize: 1,
				success:          false,
				messageQueue: &MessageQueue{
					clientSubMsgQueue: []*SubscriptionMessageDTO{
						{ClientId: 2, Sports: []domain.SportType{domain.Baseball}, UpdateIntervalSecond: 1},
					},
				},
				responseSenderCalled: false,
				subscriptions:        map[int]*ClientSubscription{},
			},
		},
		{
			name: "subscription exist and not changed",
			input: &inputSubscribe{
				clientId: 1,
				responseSender: &mockResponseSender{
					Called: false,
					FakeSend: func(sports []*domain.SportLine) error {
						return nil
					}},
				sportLineService: &MockLinesService{
					FakeCalculate: nil,
					FakeIsChanged: func(exist bool, r SportTypeMap, newValue []domain.SportType) bool {
						return false
					},
				},

				messageQueue: &MessageQueue{
					clientSubMsgQueue: []*SubscriptionMessageDTO{
						{ClientId: 1, Sports: []domain.SportType{domain.Baseball}, UpdateIntervalSecond: 1},
						{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
					},
				},
				subscriptions: map[int]*ClientSubscription{
					1: {Sports: map[domain.SportType]float32{domain.Baseball: 1.0}, Task: time.NewTicker(1)},
					2: {Sports: map[domain.SportType]float32{domain.Soccer: 1.0}, Task: time.NewTicker(1)},
				},
			},
			expected: &expectedSubscribe{
				messageQueueSize: 1,
				success:          false,
				messageQueue: &MessageQueue{
					clientSubMsgQueue: []*SubscriptionMessageDTO{
						{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
					},
				},
				subscriptions: map[int]*ClientSubscription{
					1: {Sports: map[domain.SportType]float32{domain.Baseball: 1.0}, Task: time.NewTicker(1)},
					2: {Sports: map[domain.SportType]float32{domain.Soccer: 1.0}, Task: time.NewTicker(1)},
				},
				responseSenderCalled: false,
			},
		},
		{
			name: "subscription client id == parameter clientId and client not exist on subscription map",
			input: &inputSubscribe{
				clientId: 1,
				responseSender: &mockResponseSender{
					Called: false,
					FakeSend: func(sports []*domain.SportLine) error {
						return nil
					}},
				sportLineService: &MockLinesService{
					FakeCalculate: nil,
					FakeIsChanged: nil,
				},
				messageQueue: &MessageQueue{
					clientSubMsgQueue: []*SubscriptionMessageDTO{
						{ClientId: 1, Sports: []domain.SportType{domain.Baseball}, UpdateIntervalSecond: 1},
						{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
					},
				},
			},
			expected: &expectedSubscribe{
				messageQueueSize: 1,
				success:          true,
				messageQueue: &MessageQueue{
					clientSubMsgQueue: []*SubscriptionMessageDTO{
						{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
					},
				},
				responseSenderCalled: true,
			},
		},
		{
			name: "not have message for process",
			input: &inputSubscribe{
				clientId: 1,
				responseSender: &mockResponseSender{
					Called: false,
					FakeSend: func(sports []*domain.SportLine) error {
						return nil
					}},
				sportLineService: &MockLinesService{
					FakeCalculate: nil,
					FakeIsChanged: nil,
				},
				messageQueue: &MessageQueue{},
			},
			expected: &expectedSubscribe{
				messageQueueSize:     0,
				success:              false,
				messageQueue:         &MessageQueue{},
				responseSenderCalled: false,
			},
		},
		{
			name: "subscription client id == parameter clientId and exist in subscription map but sub not changed",
			input: &inputSubscribe{
				clientId: 1,
				responseSender: &mockResponseSender{
					Called: false,
					FakeSend: func(sports []*domain.SportLine) error {
						return nil
					}},
				sportLineService: &MockLinesService{
					FakeIsChanged: func(exist bool, r SportTypeMap, newValue []domain.SportType) bool {
						return false
					},
				},
				messageQueue: &MessageQueue{
					clientSubMsgQueue: []*SubscriptionMessageDTO{
						{ClientId: 1, Sports: []domain.SportType{domain.Baseball}, UpdateIntervalSecond: 1},
						{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
					},
				},
				subscriptions: map[int]*ClientSubscription{
					1: {Sports: map[domain.SportType]float32{domain.Baseball: 1.0}, Task: time.NewTicker(1)},
				},
			},
			expected: &expectedSubscribe{
				messageQueueSize: 1,
				success:          false,
				messageQueue: &MessageQueue{
					clientSubMsgQueue: []*SubscriptionMessageDTO{
						{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
					},
				},
				responseSenderCalled: false,
				subscriptions: map[int]*ClientSubscription{
					1: {Sports: map[domain.SportType]float32{domain.Baseball: 1.0}, Task: time.NewTicker(1)},
				},
			},
		},
		{
			name: "change subscription",
			input: &inputSubscribe{
				clientId: 1,
				responseSender: &mockResponseSender{
					Called:    false,
					CountCall: 0,
					FakeSend: func(sports []*domain.SportLine) error {
						return nil
					}},
				sportLineService: &MockLinesService{
					FakeCalculate: func(sports []domain.SportType, isNeedDelta bool, subs *ClientSubscription) ([]*domain.SportLine, error) {
						return []*domain.SportLine{
							{Score: 1.0, Type: domain.Baseball},
							{Score: 1.5, Type: domain.Soccer},
						}, nil
					},
					FakeIsChanged: func(exist bool, r SportTypeMap, newValue []domain.SportType) bool {
						return true
					},
				},
				messageQueue: &MessageQueue{
					clientSubMsgQueue: []*SubscriptionMessageDTO{
						{ClientId: 1, Sports: []domain.SportType{domain.Baseball}, UpdateIntervalSecond: 1},
						{ClientId: 1, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
						{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
					},
				},
				subscriptions: map[int]*ClientSubscription{
					1: {Sports: map[domain.SportType]float32{domain.Baseball: 1.0}, Task: time.NewTicker(1)},
				},
			},
			expected: &expectedSubscribe{
				messageQueueSize: 1,
				success:          true,
				messageQueue: &MessageQueue{
					clientSubMsgQueue: []*SubscriptionMessageDTO{
						{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
					},
				},
				responseSenderCountCall: 2,
				responseSenderCalled:    true,
				subscriptions: map[int]*ClientSubscription{
					1: {Sports: map[domain.SportType]float32{domain.Soccer: 1.0}, Task: time.NewTicker(1)},
				},
			},
		},
		{
			name: "failed get data from database",
			input: &inputSubscribe{
				clientId: 1,
				responseSender: &mockResponseSender{
					Called: false,
					FakeSend: func(sports []*domain.SportLine) error {
						return nil
					}},
				sportLineService: &MockLinesService{
					FakeCalculate: func(sports []domain.SportType, isNeedDelta bool, subs *ClientSubscription) ([]*domain.SportLine, error) {
						return nil, errors.New("fake error")
					},
					FakeIsChanged: func(exist bool, r SportTypeMap, newValue []domain.SportType) bool {
						return true
					},
				},

				messageQueue: &MessageQueue{
					clientSubMsgQueue: []*SubscriptionMessageDTO{
						{ClientId: 1, Sports: []domain.SportType{domain.Baseball}, UpdateIntervalSecond: 1},
						{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
					},
				},
				subscriptions: map[int]*ClientSubscription{
					1: {Sports: map[domain.SportType]float32{domain.Soccer: 1.0}, Task: time.NewTicker(1)},
				},
			},
			expected: &expectedSubscribe{
				messageQueueSize: 1,
				success:          true,
				messageQueue: &MessageQueue{
					clientSubMsgQueue: []*SubscriptionMessageDTO{
						{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
					},
				},
				responseSenderCalled: false,
				subscriptions: map[int]*ClientSubscription{
					1: {Sports: map[domain.SportType]float32{domain.Baseball: 1.0}, Task: time.NewTicker(1)},
				},
			},
		},
		{
			name: "failed send data to subscriber",
			input: &inputSubscribe{
				clientId: 1,
				responseSender: &mockResponseSender{
					Called: false,
					FakeSend: func(sports []*domain.SportLine) error {
						return errors.New("fake error")
					}},
				sportLineService: &MockLinesService{
					FakeCalculate: func(sports []domain.SportType, isNeedDelta bool, subs *ClientSubscription) ([]*domain.SportLine, error) {
						return []*domain.SportLine{
							{Score: 1.0, Type: domain.Baseball},
							{Score: 1.5, Type: domain.Soccer},
						}, nil
					},
					FakeIsChanged: func(exist bool, r SportTypeMap, newValue []domain.SportType) bool {
						return true
					},
				},

				messageQueue: &MessageQueue{
					clientSubMsgQueue: []*SubscriptionMessageDTO{
						{ClientId: 1, Sports: []domain.SportType{domain.Baseball}, UpdateIntervalSecond: 1},
						{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
					},
				},
				subscriptions: map[int]*ClientSubscription{
					1: {Sports: map[domain.SportType]float32{domain.Soccer: 1.0}, Task: time.NewTicker(1)},
				},
			},
			expected: &expectedSubscribe{
				messageQueueSize: 1,
				success:          true,
				messageQueue: &MessageQueue{
					clientSubMsgQueue: []*SubscriptionMessageDTO{
						{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
					},
				},
				responseSenderCalled: false,
				subscriptions: map[int]*ClientSubscription{
					1: {Sports: map[domain.SportType]float32{domain.Baseball: 1.0}, Task: time.NewTicker(1)},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := test.input
			expected := test.expected
			manager := NewSubscriptionManager(input.sportLineService)
			if input.messageQueue != nil {
				manager.messageQueue = input.messageQueue
			}
			if input.subscriptions != nil {
				manager.subscriptions = input.subscriptions
			}
			respSender := input.responseSender
			success := manager.Subscribe(respSender, input.clientId)
			if expected.responseSenderCountCall > 1 {
				success = manager.Subscribe(respSender, input.clientId)
				fieldValue := getFieldValue(input.responseSender, "CountCall")
				if fieldValue != nil {
					assert.Equal(t, expected.responseSenderCountCall, fieldValue.Int())
				}
			}
			queue := manager.messageQueue
			assert.Equal(t, expected.success, success)
			assert.Equal(t, expected.messageQueueSize, queue.Size())
			expectedQueue := expected.messageQueue
			if expectedQueue != nil {
				assert.Equal(t, expectedQueue.Size(), queue.Size())
				for i := 0; i < expectedQueue.Size(); i++ {
					expectedQueue.Pop()
					for i, expectedDto := range expectedQueue.clientSubMsgQueue {
						dto := queue.clientSubMsgQueue[i]
						equalSubscriptionMessageDTO(t, expectedDto, dto)
					}
				}
			}
			if expected.responseSenderCalled {
				fieldValue := getFieldValue(input.responseSender, "Called")
				if fieldValue != nil {
					assert.Equal(t, expected.responseSenderCalled, fieldValue.Bool())
				}
			}
			if expected.subscriptions != nil {
				assert.Equal(t, len(expected.subscriptions), len(manager.subscriptions))
				for i, expectedSubs := range expected.subscriptions {
					subs := manager.subscriptions[i]
					assert.Equal(t, len(expectedSubs.Sports), len(subs.Sports))
					for sportType, line := range expectedSubs.Sports {
						f, ok := subs.Sports[sportType]
						assert.True(t, ok)
						assert.Equal(t, line, f)
					}
				}
			}
			manager.Unsubscribe(input.clientId)
		})
	}
}
