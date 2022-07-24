package domain

import (
	"errors"
	"strconv"
)

type SportType string

var (
	Baseball SportType = "baseball"
	Football SportType = "football"
	Soccer   SportType = "soccer"
)

func (s SportType) String() string {
	return string(s)
}

var SupportSports = map[string]SportType{
	"baseball": Baseball,
	"football": Football,
	"soccer":   Soccer,
}
var ErrInvalidScore = errors.New("invalid score")

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
