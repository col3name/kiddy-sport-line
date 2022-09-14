package array

import "github.com/col3name/lines/pkg/common/domain"

func Empty(array []string) bool {
	return len(array) == 0
}

func EmptyST(array []domain.SportType) bool {
	return len(array) == 0
}
