package adapter

import commonDomain "github.com/col3name/lines/pkg/common/domain"

type LinesProviderAdapter interface {
	GetLineBySport(sportType commonDomain.SportType) (*commonDomain.SportLine, error)
}
