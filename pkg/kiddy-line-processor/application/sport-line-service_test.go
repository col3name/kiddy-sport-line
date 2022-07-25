package application

import (
	"github.com/col3name/lines/pkg/common/application/errors"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/stretchr/testify/assert"
	"testing"
)

type mockDB struct {
	FakeGetSportLines func([]commonDomain.SportType) ([]*commonDomain.SportLine, error)
	FakeStore         func(model *commonDomain.SportLine) error
}

func (m *mockDB) GetSportLines(sportTypes []commonDomain.SportType) ([]*commonDomain.SportLine, error) {
	if m.FakeGetSportLines == nil {
		return []*commonDomain.SportLine{}, nil
	}
	return m.FakeGetSportLines(sportTypes)
}

func (m *mockDB) Store(model *commonDomain.SportLine) error {
	if m.FakeStore == nil {
		return nil
	}
	return m.FakeStore(model)
}

type isChangeInput struct {
	exist  bool
	subMap SportTypeMap
	sports []commonDomain.SportType
}

func TestIsChanged(t *testing.T) {
	db := &mockDB{}
	tests := []struct {
		name     string
		mockDB   *mockDB
		input    *isChangeInput
		expected bool
	}{
		{
			name:   "nil subMap",
			mockDB: db,
			input: &isChangeInput{
				exist:  false,
				subMap: nil,
				sports: nil,
			},
			expected: false,
		},
		{
			name:   "common domain empty",
			mockDB: db,
			input: &isChangeInput{
				exist: false,
				subMap: SportTypeMap(map[commonDomain.SportType]float32{
					commonDomain.Baseball: 1.0,
				}),
				sports: []commonDomain.SportType{},
			},
			expected: false,
		},
		{
			name:   "when new sport type not exist in empty old value",
			mockDB: db,
			input: &isChangeInput{
				exist:  false,
				subMap: SportTypeMap(map[commonDomain.SportType]float32{}),
				sports: []commonDomain.SportType{commonDomain.Baseball},
			},
			expected: true,
		},
		{
			name:   "when new length sportsType != subscription length",
			mockDB: db,
			input: &isChangeInput{
				exist: true,
				subMap: SportTypeMap(map[commonDomain.SportType]float32{
					commonDomain.Baseball: 1.0,
					commonDomain.Soccer:   1.5,
				}),
				sports: []commonDomain.SportType{commonDomain.Football},
			},
			expected: true,
		},
		{
			name:   "when new sport type equal old value",
			mockDB: db,
			input: &isChangeInput{
				exist: true,
				subMap: SportTypeMap(map[commonDomain.SportType]float32{
					commonDomain.Soccer: 1.0,
				}),
				sports: []commonDomain.SportType{commonDomain.Baseball},
			},
			expected: true,
		},
		{
			name:   "when new sport type equal old value",
			mockDB: db,
			input: &isChangeInput{
				exist: true,
				subMap: SportTypeMap(map[commonDomain.SportType]float32{
					commonDomain.Baseball: 1.0,
				}),
				sports: []commonDomain.SportType{commonDomain.Baseball},
			},
			expected: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewSportLineService(test.mockDB)
			input := test.input
			result := service.IsChanged(input.exist, input.subMap, input.sports)
			assert.Equal(t, test.expected, result)
		})
	}
}

type CalculateInput struct {
	isNeedDelta bool
	types       []commonDomain.SportType
	subs        *ClientSubscription
}

type CalculateExpected struct {
	err    error
	result []*commonDomain.SportLine
}

func TestCalculates(t *testing.T) {
	tests := []struct {
		name     string
		mockDB   *mockDB
		input    *CalculateInput
		expected *CalculateExpected
	}{
		{
			name: "subs nil",
			input: &CalculateInput{
				isNeedDelta: false,
				types:       nil,
				subs:        nil,
			},
			mockDB: &mockDB{
				FakeGetSportLines: func(types []commonDomain.SportType) ([]*commonDomain.SportLine, error) {
					return []*commonDomain.SportLine{}, nil
				},
			},
			expected: &CalculateExpected{
				err:    errors.ErrInvalidArgument,
				result: nil,
			},
		},
		{
			name: "table not exist",
			input: &CalculateInput{
				isNeedDelta: false,
				types:       nil,
				subs:        &ClientSubscription{Sports: nil},
			},
			mockDB: &mockDB{
				FakeGetSportLines: func(types []commonDomain.SportType) ([]*commonDomain.SportLine, error) {
					return []*commonDomain.SportLine{}, errors.ErrTableNotExist
				},
			},
			expected: &CalculateExpected{
				err:    errors.ErrTableNotExist,
				result: nil,
			},
		},
		{
			name: "internal server error",
			input: &CalculateInput{
				isNeedDelta: false,
				types:       nil,
				subs:        &ClientSubscription{Sports: nil},
			},
			mockDB: &mockDB{
				FakeGetSportLines: func(types []commonDomain.SportType) ([]*commonDomain.SportLine, error) {
					return []*commonDomain.SportLine{}, errors.ErrInternal
				},
			},
			expected: &CalculateExpected{
				err:    errors.ErrInternal,
				result: nil,
			},
		},
		{
			name: "does not need delta",
			input: &CalculateInput{
				isNeedDelta: false,
				types:       []commonDomain.SportType{commonDomain.Baseball},
				subs:        &ClientSubscription{Sports: make(SportTypeMap, 0)},
			},
			mockDB: &mockDB{
				FakeGetSportLines: func(types []commonDomain.SportType) ([]*commonDomain.SportLine, error) {
					return []*commonDomain.SportLine{{Type: commonDomain.Baseball, Score: 1.5}}, nil
				},
			},
			expected: &CalculateExpected{
				err:    nil,
				result: []*commonDomain.SportLine{{Type: commonDomain.Baseball, Score: 1.5}},
			},
		},
		{
			name: "need delta",
			input: &CalculateInput{
				isNeedDelta: true,
				types:       []commonDomain.SportType{commonDomain.Baseball},
				subs: &ClientSubscription{Sports: SportTypeMap{
					commonDomain.Baseball: 1.5,
				}},
			},
			mockDB: &mockDB{
				FakeGetSportLines: func(types []commonDomain.SportType) ([]*commonDomain.SportLine, error) {
					return []*commonDomain.SportLine{{Type: commonDomain.Baseball, Score: 1.0}}, nil
				},
			},
			expected: &CalculateExpected{
				err:    nil,
				result: []*commonDomain.SportLine{{Type: commonDomain.Baseball, Score: -0.5}},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewSportLineService(test.mockDB)
			input := test.input
			result, err := service.Calculate(input.types, input.isNeedDelta, input.subs)
			if err != nil {
				assert.Error(t, test.expected.err, err)
			} else {
				assert.Equal(t, test.expected.err, err)
			}
			if result == nil {
				assert.Equal(t, test.expected.result, result)
			} else {
				assert.Equal(t, len(test.expected.result), len(result))
				for i, line := range result {
					expectedLine := test.expected.result[i]
					assert.Equal(t, expectedLine.Score, line.Score)
					assert.Equal(t, expectedLine.Type, line.Type)
				}
			}
		})
	}
}

//db, mock, err := sqlmock.New()
//if err != nil {
//	t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
//}
//defer db.Close()
//mock.ExpectBegin()
//mock.ExpectExec("UPDATE products").WillReturnResult(sqlmock.NewResult(1, 1))
//mock.ExpectExec("INSERT INTO product_viewers").WithArgs(2, 3).WillReturnResult(sqlmock.NewResult(1, 1))
//mock.ExpectCommit()
