package entity

import "github.com/google/uuid"

type UrlPair struct {
	Uuid        uuid.UUID `json:"uuid"`
	ShortURL    string    `json:"short_url"`
	OriginalURL string    `json:"original_url"`
}
