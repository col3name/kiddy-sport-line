package query

import (
	errBase "errors"
	"github.com/col3name/lines/pkg/common/application/errors"
	"github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/infrastructure/postgres"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application/fake"
	"github.com/pashagolub/pgxmock"
	"github.com/stretchr/testify/assert"
	"testing"
)

type status int

const (
	tableNotExist status = iota
	failedQuery
	multipleTypes
	rowsError

	failedRowScan
	skip
	ok
)

type inputGetLineBySport struct {
	sportTypes []domain.SportType
	status     status
	queryErr   error
}

type expectedGetLineBySport struct {
	lines []*domain.SportLine
	err   error
}

func TestGetSportLines(t *testing.T) {
	tests := []struct {
		name     string
		input    *inputGetLineBySport
		expected *expectedGetLineBySport
	}{
		{
			name:  "empty sport types list",
			input: &inputGetLineBySport{sportTypes: []domain.SportType{}, status: skip},
			expected: &expectedGetLineBySport{
				lines: nil,
				err:   errors.ErrInvalidArgument,
			},
		},
		{
			name: "table not exist error",
			input: &inputGetLineBySport{
				sportTypes: []domain.SportType{domain.Baseball},
				status:     tableNotExist,
				queryErr:   errBase.New("ERROR: relation \"sport_lines\" does not exist (SQLSTATE 42P01)"),
			},
			expected: &expectedGetLineBySport{
				lines: nil,
				err:   errors.ErrTableNotExist,
			},
		},
		{
			name: "failed query",
			input: &inputGetLineBySport{
				sportTypes: []domain.SportType{domain.Baseball},
				status:     failedQuery,
				queryErr:   errors.ErrInternal,
			},
			expected: &expectedGetLineBySport{
				lines: nil,
				err:   errors.ErrInternal,
			},
		},
		{
			name: "multiple query",
			input: &inputGetLineBySport{
				sportTypes: []domain.SportType{domain.Baseball, domain.Soccer},
				status:     multipleTypes,
				queryErr:   errors.ErrInternal,
			},
			expected: &expectedGetLineBySport{
				lines: nil,
				err:   errors.ErrInternal,
			},
		},
		{
			name: "return rows with error",
			input: &inputGetLineBySport{
				sportTypes: []domain.SportType{domain.Baseball},
				status:     rowsError,
				queryErr:   errors.ErrInternal,
			},
			expected: &expectedGetLineBySport{
				lines: nil,
				err:   errors.ErrInternal,
			},
		},
		{
			name: "failed row scan",
			input: &inputGetLineBySport{
				sportTypes: []domain.SportType{domain.Baseball},
				status:     failedRowScan,
				queryErr:   nil,
			},
			expected: &expectedGetLineBySport{
				lines: nil,
				err:   errors.ErrInternal,
			},
		},
		{
			name: "success get lines",
			input: &inputGetLineBySport{
				sportTypes: []domain.SportType{domain.Baseball},
				status:     ok,
				queryErr:   nil,
			},
			expected: &expectedGetLineBySport{
				lines: []*domain.SportLine{
					{Type: domain.Baseball, Score: 0.744},
				},
				err: nil,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mock, err := postgres.GetPgxMockPool(t)
			if err != nil {
				return
			}
			defer mock.Close()

			input := test.input
			expected := test.expected

			setupGetSportLinesUseCases(mock, input, expected)

			repo := NewSportLineQueryService(mock, fake.Logger{})
			types, err := repo.GetLinesBySportTypes(input.sportTypes)

			compareLines(t, expected, err, types)
			if input.status != skip {
				postgres.CheckExpectationsWereMet(t, mock)
			}
		})
	}
}

func fillData(input *inputGetLineBySport) []interface{} {
	var data []interface{}
	for _, sportType := range input.sportTypes {
		data = append(data, sportType)
	}
	return data
}

func setupGetSportLinesUseCases(mock pgxmock.PgxPoolIface, input *inputGetLineBySport, expected *expectedGetLineBySport) {
	data := fillData(input)

	switch input.status {
	case tableNotExist:
		mock.ExpectQuery("SELECT score,sport_type FROM sport_lines").WithArgs(data...).
			WillReturnError(input.queryErr)
	case failedQuery:
		mock.ExpectQuery("SELECT score,sport_type FROM sport_lines").WithArgs(data...).
			WillReturnError(input.queryErr)
	case rowsError:
		r := pgxmock.NewRows([]string{"exists"}).AddRow(&domain.SportLine{
			Type:  domain.Baseball,
			Score: 0.744,
		})
		r.RowError(0, errors.ErrInternal)
		mock.ExpectQuery("SELECT score,sport_type FROM sport_lines").
			WillReturnError(nil).
			WillReturnRows(r.CloseError(errors.ErrInternal))
	case failedRowScan:
		rs := pgxmock.NewRows([]string{"type"})
		mock.ExpectQuery("SELECT score,sport_type FROM sport_lines").
			WillReturnError(nil).
			WillReturnRows(rs.AddRow("line.Score"))
	case multipleTypes:
		var args []interface{}
		for _, sportType := range input.sportTypes {
			args = append(args, sportType)
		}
		sql := "SELECT score,sport_type FROM sport_lines WHERE sport_type = (.+) UNION ALL SELECT score,sport_type FROM sport_lines WHERE sport_type =(.+);"
		mock.ExpectQuery(sql).
			WithArgs(args...)
	case ok:
		rs := pgxmock.NewRows([]string{"score", "type"})
		for _, line := range expected.lines {
			rs.AddRow(line.Score, line.Type)
		}
		mock.ExpectQuery("SELECT score,sport_type FROM sport_lines").
			WillReturnError(nil).
			WillReturnRows(rs)
	}
}

func compareLines(t *testing.T, expected *expectedGetLineBySport, err error, actualLines []*domain.SportLine) {
	assert.Equal(t, expected.err, err)

	expectedLines := expected.lines
	if expectedLines != nil {
		assert.Equal(t, len(expectedLines), len(actualLines))
	} else {
		assert.Nil(t, actualLines)
	}

	for i, expectedLine := range expectedLines {
		actualLine := actualLines[i]
		compareSportLines(t, expectedLine, actualLine)
	}
}

func compareSportLines(t *testing.T, expected *domain.SportLine, actual *domain.SportLine) {
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.Score, actual.Score)
}
