package postgres

import (
	"github.com/col3name/lines/pkg/common/application/errors"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestName(t *testing.T) {
	assert.True(t, strings.Contains("ERROR: relation \"sport_lines\" does not exist (SQLSTATE 42P01)", errors.TableNotExistMessage))
}
