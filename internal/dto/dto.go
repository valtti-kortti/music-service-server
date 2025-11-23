package dto

import (
	"github.com/google/uuid"
	"time"
)

type Video struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	Duration int64  `json:"duration"`
}

type ResponseRoom struct {
	ID uuid.UUID `json:"id"`
}

type Command struct {
	Type string `json:"type"` // "play", "pause", "next"
}

type ErrorResponse struct {
	Message string `json:"message"`
}

type Room struct {
	ID          uuid.UUID `json:"id"`
	Queue       []*Video  `json:"queue"`
	Current     *Video    `json:"current"`
	Playing     bool      `json:"playing"`
	Subscribers int       `json:"subscribers"`
}

type State struct {
	ID        uuid.UUID `json:"id"`
	Current   *Video    `json:"current"` // что играет (может быть nil)
	Queue     []*Video  `json:"queue"`   // копия очереди
	Playing   bool      `json:"playing"`
	Position  float64   `json:"position"`   // на какой секунде сейчас должен быть трек
	UpdatedAt time.Time `json:"updated_at"` // когда этот state посчитали
}
