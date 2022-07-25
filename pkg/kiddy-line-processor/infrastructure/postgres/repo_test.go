package postgres

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestName(t *testing.T) {
	assert.True(t, strings.Contains("ERROR: relation \"sport_lines\" does not exist (SQLSTATE 42P01)", TableNotExistMessage))
}
