package adapter

import (
	"errors"
	appErr "github.com/col3name/lines/pkg/common/application/errors"
	"github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/infrastructure/transport"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application/fake"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"strings"
	"testing"
)

type MockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

// Do is the mock client's `Do` func
func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc == nil {
		return nil, nil
	}
	return m.DoFunc(req)
}

type inputTestCase struct {
	doFunc    func(req *http.Request) (*http.Response, error)
	sportType domain.SportType
}

type expectedTestCase struct {
	err       error
	sportLine *domain.SportLine
}

type errReader int

func (errReader) Read(_ []byte) (n int, err error) {
	return 0, errors.New("test error")
}

func TestGetLines(t *testing.T) {
	tests := []struct {
		name     string
		input    *inputTestCase
		expected *expectedTestCase
	}{
		{
			name: "http client return error",
			input: &inputTestCase{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("fake error")
				},
				sportType: domain.Soccer,
			},
			expected: &expectedTestCase{
				err:       appErr.ErrExternal,
				sportLine: nil,
			},
		},
		{
			name: "response http status != 200",
			input: &inputTestCase{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusNotFound,
					}, nil
				},
				sportType: domain.Soccer,
			},
			expected: &expectedTestCase{
				err:       appErr.ErrExternal,
				sportLine: nil,
			},
		},
		{
			name: "failed read response body",
			input: &inputTestCase{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("")),
					}, nil
				},
				sportType: domain.Soccer,
			},
			expected: &expectedTestCase{
				err:       appErr.ErrInternal,
				sportLine: nil,
			},
		},
		{
			name: "failed read response body",
			input: &inputTestCase{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(errReader(0)),
					}, nil
				},
				sportType: domain.Soccer,
			},
			expected: &expectedTestCase{
				err:       appErr.ErrInternal,
				sportLine: nil,
			},
		},
		{
			name: "failed unmarshal Baseball",
			input: &inputTestCase{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("")),
					}, nil
				},
				sportType: domain.Baseball,
			},
			expected: &expectedTestCase{
				err:       appErr.ErrInternal,
				sportLine: nil,
			},
		},
		{
			name: "failed unmarshal Soccer",
			input: &inputTestCase{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("")),
					}, nil
				},
				sportType: domain.Soccer,
			},
			expected: &expectedTestCase{
				err:       appErr.ErrInternal,
				sportLine: nil,
			},
		},
		{
			name: "failed unmarshal Football",
			input: &inputTestCase{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("")),
					}, nil
				},
				sportType: domain.Football,
			},
			expected: &expectedTestCase{
				err:       appErr.ErrInternal,
				sportLine: nil,
			},
		},
		{
			name: "invalid score",
			input: &inputTestCase{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("{\"rows\":{\"BASEBdALL\":\"0.774\"}}")),
					}, nil
				},
				sportType: domain.Soccer,
			},
			expected: &expectedTestCase{
				err:       appErr.ErrInternal,
				sportLine: nil,
			},
		},
		{
			name: "invalid response sport type",
			input: &inputTestCase{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("{\"lines\":{\"BASEBALL\":\"0.774\"}}")),
					}, nil
				},
				sportType: domain.Soccer,
			},
			expected: &expectedTestCase{
				err:       appErr.ErrInternal,
				sportLine: nil,
			},
		},
		{
			name: "valid baseball",
			input: &inputTestCase{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("{\"lines\":{\"BASEBALL\":\"0.774\"}}")),
					}, nil
				},
				sportType: domain.Baseball,
			},
			expected: &expectedTestCase{
				err: nil,
				sportLine: &domain.SportLine{
					Type:  domain.Baseball,
					Score: 0.774,
				},
			},
		},
		{
			name: "valid football",
			input: &inputTestCase{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("{\"lines\":{\"FOOTBALL\":\"0.774\"}}")),
					}, nil
				},
				sportType: domain.Football,
			},
			expected: &expectedTestCase{
				err: nil,
				sportLine: &domain.SportLine{
					Type:  domain.Football,
					Score: 0.774,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transport.Client = &MockClient{DoFunc: test.input.doFunc}
			adapter := NewLinesProviderAdapter("http://localhost:8000", fake.Logger{})
			line, err := adapter.GetLineBySport(test.input.sportType)
			expected := test.expected
			assert.Equal(t, expected.err, err)
			if expected.sportLine == nil {
				assert.Nil(t, line)
			} else {
				assert.Equal(t, expected.sportLine.Type, line.Type)
				assert.Equal(t, expected.sportLine.Score, line.Score)
			}
		})
	}
}
