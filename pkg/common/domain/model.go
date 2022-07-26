package domain

import (
	"errors"
	"strconv"
	"strings"
)

type SportType string

const (
	Baseball SportType = "baseball"
	Football SportType = "football"
	Soccer   SportType = "soccer"
)

var (
	ErrUnsupportedSportType = errors.New("unsupported sport type")
	ErrInvalidScore         = errors.New("invalid score")
)

func (s SportType) String() string {
	return string(s)
}

func NewSportType(sport string) (SportType, error) {
	switch strings.ToLower(sport) {
	case Baseball.String():
		return Baseball, nil
	case Soccer.String():
		return Soccer, nil
	case Football.String():
		return Football, nil
	default:
		return "", ErrUnsupportedSportType
	}
}

var SupportSports = map[string]SportType{
	"baseball": Baseball,
	"football": Football,
	"soccer":   Soccer,
}

type SportLine struct {
	Type  SportType
	Score float32
}

func (s *SportLine) SetScore(score string) error {
	value, err := strconv.ParseFloat(score, 32)
	if err != nil {
		return ErrInvalidScore
	}
	s.Score = float32(value)
	return nil
}
