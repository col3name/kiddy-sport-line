package subscription

import (
	"errors"
	"github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application/fake"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application/service"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application/sport-line"
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain/model"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
	"time"
)

type expectedPushMessage struct {
	queueSize int
	msg       *MessageToSubscribeDTO
}

func getFieldValue(object interface{}, fieldName string) *reflect.Value {
	val := reflect.ValueOf(object).Elem()
	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)

		if typeField.Name == fieldName {
			valueField := val.Field(i)
			return &valueField
		}
	}
	return nil
}

func compareSubscriptionMessageDTO(t *testing.T, lhs, rhs *MessageToSubscribeDTO) {
	assert.Equal(t, lhs.UpdateIntervalSecond, rhs.UpdateIntervalSecond)
	assert.Equal(t, lhs.ClientId, rhs.ClientId)
	assert.Equal(t, len(lhs.Sports), len(rhs.Sports))
	for i, sport := range rhs.Sports {
		assert.Equal(t, lhs.Sports[i], sport)
	}
}

func compareSports(t *testing.T, expected, actual model.SportTypeMap) {
	assert.Equal(t, len(expected), len(actual))
	for sportType, line := range actual {
		expectedLine, ok := expected[sportType]
		assert.True(t, ok)
		assert.Equal(t, expectedLine, line)
	}
}

var testsCaseForPushMessage = []struct {
	name     string
	input    *MessageToSubscribeDTO
	expected *expectedPushMessage
}{
	{
		name: "empty sports",
		input: &MessageToSubscribeDTO{
			ClientId:             1,
			Sports:               []domain.SportType{},
			UpdateIntervalSecond: 1,
		},
		expected: &expectedPushMessage{queueSize: 0},
	},
	{
		name: "invalid client id",
		input: &MessageToSubscribeDTO{
			ClientId:             0,
			Sports:               []domain.SportType{},
			UpdateIntervalSecond: 1,
		},
		expected: &expectedPushMessage{queueSize: 0},
	},
	{
		name: "negative client id",
		input: &MessageToSubscribeDTO{
			ClientId:             -1,
			Sports:               []domain.SportType{},
			UpdateIntervalSecond: 1,
		},
		expected: &expectedPushMessage{queueSize: 0},
	},
	{
		name: "update interval < 1",
		input: &MessageToSubscribeDTO{
			ClientId:             -1,
			Sports:               []domain.SportType{},
			UpdateIntervalSecond: 1,
		},
		expected: &expectedPushMessage{queueSize: 0},
	},
	{
		name: "update interval < 1",
		input: &MessageToSubscribeDTO{
			ClientId:             -1,
			Sports:               []domain.SportType{},
			UpdateIntervalSecond: 1,
		},
		expected: &expectedPushMessage{queueSize: 0},
	},
	{
		name: "valid sub message",
		input: &MessageToSubscribeDTO{
			ClientId:             1,
			Sports:               []domain.SportType{domain.Baseball},
			UpdateIntervalSecond: 1,
		},
		expected: &expectedPushMessage{queueSize: 1,
			msg: &MessageToSubscribeDTO{
				ClientId:             1,
				Sports:               []domain.SportType{domain.Baseball},
				UpdateIntervalSecond: 1,
			},
		},
	},
}

func TestPushMessage(t *testing.T) {
	for _, test := range testsCaseForPushMessage {
		t.Run(test.name, func(t *testing.T) {
			manager := NewSubscriptionManager(&MockLinesService{FakeCalculate: nil, FakeIsChanged: nil}, &fake.Logger{})
			manager.PushMessage(test.input)
			assert.Equal(t, test.expected.queueSize, manager.messageQueue.Size())
			if test.expected.queueSize > 0 {
				peek := manager.messageQueue.Peek()
				msg := test.expected.msg
				compareSubscriptionMessageDTO(t, msg, peek)
			}
		})
	}
}

type MockLinesService struct {
	FakeCalculate func(sports []domain.SportType, isNeedDelta bool, subs *model.ClientSubscription) ([]*domain.SportLine, error)
	FakeIsChanged func(exist bool, subscriptionMap model.SportTypeMap, newValue []domain.SportType) bool
}

func (m *MockLinesService) Calculate(sports []domain.SportType, isNeedDelta bool, subs *model.ClientSubscription) ([]*domain.SportLine, error) {
	if m.FakeCalculate == nil {
		return nil, nil
	}
	return m.FakeCalculate(sports, isNeedDelta, subs)
}

func (m *MockLinesService) IsSubscriptionChanged(exist bool, subscriptionMap model.SportTypeMap, subscribeToSports []domain.SportType) bool {
	if m.FakeIsChanged == nil {
		return false
	}
	return m.FakeIsChanged(exist, subscriptionMap, subscribeToSports)
}

type inputUnsubscribeClient struct {
	subscriptions map[int]*model.ClientSubscription
	clientId      int
}

type expectedUnsubscribeClient struct {
	exist             bool
	subscriptionsSize int
	subscription      *model.ClientSubscription
}

func compareSubscriptionManager(t *testing.T, expected *expectedUnsubscribeClient, input *inputUnsubscribeClient, manager *subscriptionServiceImpl) {
	actualSubscription, ok := manager.subscriptions[input.clientId]

	assert.Equal(t, expected.exist, ok)
	assert.Equal(t, expected.subscriptionsSize, len(manager.subscriptions))

	compareClientSubscription(t, expected.subscription, actualSubscription)
}

func compareClientSubscription(t *testing.T, expectSubscription, actualSubscription *model.ClientSubscription) {
	if expectSubscription == nil {
		assert.True(t, actualSubscription == nil)
		return
	}

	assert.Equal(t, expectSubscription, actualSubscription)
	assert.Equal(t, expectSubscription.Task != nil, actualSubscription.Task != nil)

	compareSports(t, expectSubscription.Sports, actualSubscription.Sports)
}

var testsCaseForUnsubscribeClient = []struct {
	name     string
	input    *inputUnsubscribeClient
	expected *expectedUnsubscribeClient
}{
	{
		name: "not exist client",
		input: &inputUnsubscribeClient{
			subscriptions: map[int]*model.ClientSubscription{},
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
			subscriptions: map[int]*model.ClientSubscription{
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

func TestUnsubscribeClient(t *testing.T) {
	for _, test := range testsCaseForUnsubscribeClient {
		t.Run(test.name, func(t *testing.T) {
			manager := NewSubscriptionManager(&MockLinesService{FakeCalculate: nil, FakeIsChanged: nil}, &fake.Logger{})
			input := test.input
			expected := test.expected

			manager.subscriptions = input.subscriptions
			manager.Unsubscribe(input.clientId)

			compareSubscriptionManager(t, expected, input, manager)
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
	responseSender   service.ResponseSenderService
	messageQueue     *MessageQueue
	sportLineService sport_line.SportLineService
	subscriptions    map[int]*model.ClientSubscription
}

type expectedSubscribe struct {
	messageQueueSize        int
	subscribedOk            bool
	messageQueue            *MessageQueue
	responseSenderCalled    bool
	subscriptions           map[int]*model.ClientSubscription
	responseSenderCountCall int64
}

func compareDeepSubscriptionManager(t *testing.T, input *inputSubscribe, expected *expectedSubscribe, manager *subscriptionServiceImpl) {
	compareSubscribeResult(t, manager, input, expected)

	expectedQueue := expected.messageQueue
	actualQueue := manager.messageQueue
	compareMessageQueue(t, expected.messageQueueSize, expectedQueue, actualQueue)
	compareResponseSenderCalled(t, expected, input)
	compareSubscriptions(t, expected.subscriptions, manager.subscriptions)
}

func compareSubscribeResult(t *testing.T, manager *subscriptionServiceImpl, input *inputSubscribe, expected *expectedSubscribe) {
	respSender := input.responseSender

	actualSubscribedOk := manager.Subscribe(respSender, input.clientId)
	if expected.responseSenderCountCall <= 1 {
		assert.Equal(t, expected.subscribedOk, actualSubscribedOk)
		return
	}
	actualSubscribedOk = manager.Subscribe(respSender, input.clientId)
	fieldValue := getFieldValue(input.responseSender, "CountCall")
	if fieldValue != nil {
		assert.Equal(t, expected.responseSenderCountCall, fieldValue.Int())
	}

	assert.Equal(t, expected.subscribedOk, actualSubscribedOk)
}

func compareMessageQueue(t *testing.T, expectedSize int, expectedQueue, actualQueue *MessageQueue) {
	assert.Equal(t, expectedSize, actualQueue.Size())
	if expectedQueue == nil {
		return
	}
	assert.Equal(t, expectedQueue.Size(), actualQueue.Size())
	for i := 0; i < expectedQueue.Size(); i++ {
		expectedQueue.Pop()
		compareDeepQueueData(t, expectedQueue.data, actualQueue.data)
	}
}

func compareDeepQueueData(t *testing.T, expected, actual []*MessageToSubscribeDTO) {
	for j, expectedDto := range expected {
		compareSubscriptionMessageDTO(t, expectedDto, actual[j])
	}
}

func compareResponseSenderCalled(t *testing.T, expected *expectedSubscribe, input *inputSubscribe) {
	if expected.responseSenderCalled {
		fieldValue := getFieldValue(input.responseSender, "Called")
		if fieldValue != nil {
			assert.Equal(t, expected.responseSenderCalled, fieldValue.Bool())
		}
	}
}

func compareSubscriptions(t *testing.T, expectedSubsMap, actualSubsMap map[int]*model.ClientSubscription) {
	if expectedSubsMap == nil {
		return
	}
	assert.Equal(t, len(expectedSubsMap), len(actualSubsMap))
	for i, expectedSubs := range expectedSubsMap {
		actualSubscription := (actualSubsMap)[i]
		compareSports(t, expectedSubs.Sports, actualSubscription.Sports)
	}
}

var testsCaseForSubscribe = []struct {
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
			subscribedOk:         false,
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
			subscribedOk:         false,
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
				data: []*MessageToSubscribeDTO{
					{ClientId: 1, Sports: []domain.SportType{}, UpdateIntervalSecond: 1},
				},
			},
		},
		expected: &expectedSubscribe{
			messageQueueSize:     0,
			subscribedOk:         false,
			responseSenderCalled: false,
			messageQueue: &MessageQueue{
				data: []*MessageToSubscribeDTO{},
			},
			subscriptions: map[int]*model.ClientSubscription{},
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
				data: []*MessageToSubscribeDTO{
					{ClientId: 2, Sports: []domain.SportType{domain.Baseball}, UpdateIntervalSecond: 1},
				},
			},
		},
		expected: &expectedSubscribe{
			messageQueueSize: 1,
			subscribedOk:     false,
			messageQueue: &MessageQueue{
				data: []*MessageToSubscribeDTO{
					{ClientId: 2, Sports: []domain.SportType{domain.Baseball}, UpdateIntervalSecond: 1},
				},
			},
			responseSenderCalled: false,
			subscriptions:        map[int]*model.ClientSubscription{},
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
				FakeIsChanged: func(exist bool, r model.SportTypeMap, newValue []domain.SportType) bool {
					return false
				},
			},

			messageQueue: &MessageQueue{
				data: []*MessageToSubscribeDTO{
					{ClientId: 1, Sports: []domain.SportType{domain.Baseball}, UpdateIntervalSecond: 1},
					{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
				},
			},
			subscriptions: map[int]*model.ClientSubscription{
				1: {Sports: map[domain.SportType]float32{domain.Baseball: 1.0}, Task: time.NewTicker(1)},
				2: {Sports: map[domain.SportType]float32{domain.Soccer: 1.0}, Task: time.NewTicker(1)},
			},
		},
		expected: &expectedSubscribe{
			messageQueueSize: 1,
			subscribedOk:     false,
			messageQueue: &MessageQueue{
				data: []*MessageToSubscribeDTO{
					{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
				},
			},
			subscriptions: map[int]*model.ClientSubscription{
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
				data: []*MessageToSubscribeDTO{
					{ClientId: 1, Sports: []domain.SportType{domain.Baseball}, UpdateIntervalSecond: 1},
					{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
				},
			},
		},
		expected: &expectedSubscribe{
			messageQueueSize: 1,
			subscribedOk:     true,
			messageQueue: &MessageQueue{
				data: []*MessageToSubscribeDTO{
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
			subscribedOk:         false,
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
				FakeIsChanged: func(exist bool, r model.SportTypeMap, newValue []domain.SportType) bool {
					return false
				},
			},
			messageQueue: &MessageQueue{
				data: []*MessageToSubscribeDTO{
					{ClientId: 1, Sports: []domain.SportType{domain.Baseball}, UpdateIntervalSecond: 1},
					{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
				},
			},
			subscriptions: map[int]*model.ClientSubscription{
				1: {Sports: map[domain.SportType]float32{domain.Baseball: 1.0}, Task: time.NewTicker(1)},
			},
		},
		expected: &expectedSubscribe{
			messageQueueSize: 1,
			subscribedOk:     false,
			messageQueue: &MessageQueue{
				data: []*MessageToSubscribeDTO{
					{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
				},
			},
			responseSenderCalled: false,
			subscriptions: map[int]*model.ClientSubscription{
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
				FakeCalculate: func(sports []domain.SportType, isNeedDelta bool, subs *model.ClientSubscription) ([]*domain.SportLine, error) {
					return []*domain.SportLine{
						{Score: 1.0, Type: domain.Baseball},
						{Score: 1.5, Type: domain.Soccer},
					}, nil
				},
				FakeIsChanged: func(exist bool, r model.SportTypeMap, newValue []domain.SportType) bool {
					return true
				},
			},
			messageQueue: &MessageQueue{
				data: []*MessageToSubscribeDTO{
					{ClientId: 1, Sports: []domain.SportType{domain.Baseball}, UpdateIntervalSecond: 1},
					{ClientId: 1, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
					{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
				},
			},
			subscriptions: map[int]*model.ClientSubscription{
				1: {Sports: map[domain.SportType]float32{domain.Baseball: 1.0}, Task: time.NewTicker(1)},
			},
		},
		expected: &expectedSubscribe{
			messageQueueSize: 1,
			subscribedOk:     true,
			messageQueue: &MessageQueue{
				data: []*MessageToSubscribeDTO{
					{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
				},
			},
			responseSenderCountCall: 2,
			responseSenderCalled:    true,
			subscriptions: map[int]*model.ClientSubscription{
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
				FakeCalculate: func(sports []domain.SportType, isNeedDelta bool, subs *model.ClientSubscription) ([]*domain.SportLine, error) {
					return nil, errors.New("fake error")
				},
				FakeIsChanged: func(exist bool, r model.SportTypeMap, newValue []domain.SportType) bool {
					return true
				},
			},

			messageQueue: &MessageQueue{
				data: []*MessageToSubscribeDTO{
					{ClientId: 1, Sports: []domain.SportType{domain.Baseball}, UpdateIntervalSecond: 1},
					{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
				},
			},
			subscriptions: map[int]*model.ClientSubscription{
				1: {Sports: map[domain.SportType]float32{domain.Soccer: 1.0}, Task: time.NewTicker(1)},
			},
		},
		expected: &expectedSubscribe{
			messageQueueSize: 1,
			subscribedOk:     true,
			messageQueue: &MessageQueue{
				data: []*MessageToSubscribeDTO{
					{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
				},
			},
			responseSenderCalled: false,
			subscriptions: map[int]*model.ClientSubscription{
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
				FakeCalculate: func(sports []domain.SportType, isNeedDelta bool, subs *model.ClientSubscription) ([]*domain.SportLine, error) {
					return []*domain.SportLine{
						{Score: 1.0, Type: domain.Baseball},
						{Score: 1.5, Type: domain.Soccer},
					}, nil
				},
				FakeIsChanged: func(exist bool, r model.SportTypeMap, newValue []domain.SportType) bool {
					return true
				},
			},

			messageQueue: &MessageQueue{
				data: []*MessageToSubscribeDTO{
					{ClientId: 1, Sports: []domain.SportType{domain.Baseball}, UpdateIntervalSecond: 1},
					{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
				},
			},
			subscriptions: map[int]*model.ClientSubscription{
				1: {Sports: map[domain.SportType]float32{domain.Soccer: 1.0}, Task: time.NewTicker(1)},
			},
		},
		expected: &expectedSubscribe{
			messageQueueSize: 1,
			subscribedOk:     true,
			messageQueue: &MessageQueue{
				data: []*MessageToSubscribeDTO{
					{ClientId: 2, Sports: []domain.SportType{domain.Soccer}, UpdateIntervalSecond: 1},
				},
			},
			responseSenderCalled: false,
			subscriptions: map[int]*model.ClientSubscription{
				1: {Sports: map[domain.SportType]float32{domain.Baseball: 1.0}, Task: time.NewTicker(1)},
			},
		},
	},
}

func TestSubscribe(t *testing.T) {
	for _, test := range testsCaseForSubscribe {
		t.Run(test.name, func(t *testing.T) {
			input := test.input
			expected := test.expected

			manager := NewSubscriptionManager(input.sportLineService, &fake.Logger{})
			if input.messageQueue != nil {
				manager.messageQueue = input.messageQueue
			}
			if input.subscriptions != nil {
				manager.subscriptions = input.subscriptions
			}

			compareDeepSubscriptionManager(t, input, expected, manager)

			manager.Unsubscribe(input.clientId)
		})
	}
}
