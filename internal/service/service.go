package service

import (
	"crypto/rand"
	"encoding/hex"
)

//go:generate mockery --name Repository --output ./mocks --outpkg mocks
type Repository interface {
	GetShortByOrig(orig string) string
	GetOrigByShort(short string) string
	SaveIfNotTaken(orig, short string) (string, error)
}

type URLService struct {
	repo Repository
}

func New(repo Repository) *URLService {
	return &URLService{repo: repo}
}

func (s *URLService) GetOrCreate(origURL string) string {
	for range 10 {
		short, err := s.repo.SaveIfNotTaken(origURL, randomString(8))
		if err != nil {
			continue
		}
		return short
	}
	return ""
}

func (s *URLService) GetOrigURL(shortID string) string {
	return s.repo.GetOrigByShort(shortID)
}

func randomString(length int) string {
	numBytes := (length + 1) / 2
	b := make([]byte, numBytes)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)[:length]
}
