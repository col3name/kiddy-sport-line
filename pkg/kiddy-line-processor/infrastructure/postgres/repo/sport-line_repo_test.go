package repo

import (
	"github.com/col3name/lines/pkg/common/application/errors"
	"github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/infrastructure/postgres"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application/fake"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application/service"
	"github.com/jackc/pgconn"
	"github.com/pashagolub/pgxmock"
	"github.com/stretchr/testify/assert"
	"testing"
)

type status int

const (
	failedStartTransaction status = iota
	failedDoRollback
	successDoRollback
	failedDoCommitUserDoesNotExist
	failedDoCommit
	successDoCommit

	ok
)

type inputStore struct {
	sport       *domain.SportLine
	errorCommit error
}

type expectedStore struct {
	err    error
	result pgconn.CommandTag
	status status
}

type storeTest struct {
	name     string
	input    *inputStore
	expected *expectedStore
}

func TestStore(t *testing.T) {
	tests := []storeTest{
		{
			name: "failed start transaction",
			input: &inputStore{
				sport: &domain.SportLine{Type: domain.Baseball, Score: 0.744},
			},
			expected: &expectedStore{
				status: failedStartTransaction,
				err:    errors.ErrInternal,
				result: nil,
			},
		},
		{
			name: "failed rollback transaction",
			input: &inputStore{
				sport: &domain.SportLine{Type: domain.Baseball, Score: 0.744},
			},
			expected: &expectedStore{
				status: failedDoRollback,
				err:    errors.ErrInternal,
				result: nil,
			},
		},
		{
			name: "success rollback transaction",
			input: &inputStore{
				sport: &domain.SportLine{Type: domain.Baseball, Score: 0.744},
			},
			expected: &expectedStore{
				status: successDoRollback,
				err:    errors.ErrInternal,
				result: nil,
			},
		},
		{
			name: "failed commit transaction sport line doesn't exist",
			input: &inputStore{
				sport:       &domain.SportLine{Type: domain.Baseball, Score: 0.744},
				errorCommit: domain.ErrSportLinesDoesNotExist,
			},
			expected: &expectedStore{
				status: failedDoCommitUserDoesNotExist,
				err:    errors.ErrInternal,
				result: pgconn.CommandTag{},
			},
		},
		{
			name: "failed commit transaction",
			input: &inputStore{
				sport: &domain.SportLine{Type: domain.Baseball, Score: 0.744},
			},
			expected: &expectedStore{
				status: failedDoCommit,
				err:    errors.ErrInternal,
				result: pgconn.CommandTag("UPDATE 1"),
			},
		},
		{
			name: "success commit transaction",
			input: &inputStore{
				sport: &domain.SportLine{Type: domain.Baseball, Score: 0.744},
			},
			expected: &expectedStore{
				status: successDoCommit,
				err:    nil,
				result: pgconn.CommandTag("UPDATE 1"),
			},
		},
		{
			name: "failed save",
			input: &inputStore{
				sport: &domain.SportLine{Type: domain.Baseball, Score: 0.744},
			},
			expected: &expectedStore{
				status: ok,
				err:    errors.ErrInternal,
				result: nil,
			},
		},
		{
			name: "success save",
			input: &inputStore{
				sport: &domain.SportLine{Type: domain.Baseball, Score: 0.744},
			},
			expected: &expectedStore{
				status: ok,
				err:    nil,
				result: pgxmock.NewResult("UPDATE", 1),
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

			setupStoreUseCases(mock, &test)

			uow := NewUnitOfWork(mock, fake.Logger{})
			err = uow.Execute(func(rp service.RepositoryProvider) error {
				repo := rp.SportLineRepo()
				return repo.Store(test.input.sport)
			})
			assert.Equal(t, test.expected.err, err)
			postgres.CheckExpectationsWereMet(t, mock)
		})
	}
}

func setupStoreUseCases(mock pgxmock.PgxPoolIface, test *storeTest) {
	input := test.input
	expected := test.expected
	expectedErr := expected.err
	inputSportLine := input.sport
	inputScore := inputSportLine.Score
	inputType := inputSportLine.Type

	switch expected.status {
	case ok:
		mock.ExpectBegin()
		exec := mock.ExpectExec("UPDATE sport_lines")
		if expectedErr != nil {
			exec.WillReturnError(expectedErr)
			mock.ExpectRollback()
		} else {
			exec.WillReturnResult(expected.result)
			mock.ExpectCommit().WillReturnError(nil)
		}
	case failedStartTransaction:
		mock.ExpectBegin().WillReturnError(expectedErr)
	case failedDoRollback:
		mock.ExpectBegin().WillReturnError(nil)
		mock.ExpectExec("UPDATE sport_lines").WillReturnError(expectedErr)
		mock.ExpectRollback().WillReturnError(expectedErr)
	case successDoRollback:
		mock.ExpectBegin().WillReturnError(nil)
		mock.ExpectExec("UPDATE sport_lines").
			WithArgs(inputScore, inputType).
			WillReturnError(expectedErr)
		mock.ExpectRollback().WillReturnError(nil)
	case failedDoCommitUserDoesNotExist:
		mock.ExpectBegin().WillReturnError(nil)
		mock.ExpectExec("UPDATE sport_lines").
			WithArgs(inputScore, inputType).
			WillReturnResult(expected.result).
			WillReturnError(input.errorCommit)
		mock.ExpectRollback().WillReturnError(expected.err)
	case failedDoCommit:
		mock.ExpectBegin().WillReturnError(nil)
		mock.ExpectExec("UPDATE sport_lines").
			WithArgs(inputScore, inputType).
			WillReturnResult(expected.result).
			WillReturnError(nil)
		mock.ExpectCommit().WillReturnError(errors.ErrInternal)
	case successDoCommit:
		mock.ExpectBegin().WillReturnError(nil)
		mock.ExpectExec("UPDATE sport_lines").
			WillReturnResult(expected.result).
			WillReturnError(nil)
		mock.ExpectCommit().WillReturnError(nil)
	}
}
