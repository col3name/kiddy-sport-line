package domain

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSportTypeFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected struct {
			res SportType
			err error
		}
	}{
		{
			name:  "unsupported sport type",
			input: "nand",
			expected: struct {
				res SportType
				err error
			}{res: "", err: ErrUnsupportedSportType},
		},
		{
			name:  "success from baseball",
			input: "baseball",
			expected: struct {
				res SportType
				err error
			}{res: Baseball, err: nil},
		},
		{
			name:  "success from soccer",
			input: "soccer",
			expected: struct {
				res SportType
				err error
			}{res: Soccer, err: nil},
		},
		{
			name:  "success from football",
			input: "football",
			expected: struct {
				res SportType
				err error
			}{res: Football, err: nil},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expected := test.expected
			res, err := NewSportType(test.input)
			assert.Equal(t, expected.err, err)
			assert.Equal(t, expected.res.String(), res.String())
		})
	}
}

type inputSportLine struct {
	in  SportLine
	val string
}
type expectedSportLine struct {
	err error
	res SportLine
}

func TestSportLine(t *testing.T) {
	tests := []struct {
		name     string
		input    inputSportLine
		expected expectedSportLine
	}{
		{
			name: "invalid score string",
			input: inputSportLine{
				in:  SportLine{Baseball, 0.744},
				val: "hello",
			},
			expected: expectedSportLine{
				err: ErrInvalidScore,
				res: SportLine{Baseball, 0.744},
			},
		},
		{
			name: "valid score string",
			input: inputSportLine{
				in:  SportLine{Baseball, 0.744},
				val: "1.0",
			},
			expected: expectedSportLine{
				err: nil,
				res: SportLine{Baseball, 1.0},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := test.input
			expected := test.expected
			err := input.in.SetScore(input.val)
			assert.Equal(t, expected.err, err)
			assert.Equal(t, expected.res.Type, input.in.Type)
			assert.Equal(t, expected.res.Score, input.in.Score)
		})
	}
}
