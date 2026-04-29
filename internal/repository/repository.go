package repository

import (
	"errors"
	"sync"
)

type URLRepository struct {
	mu          sync.Mutex
	origToShort map[string]string
	shortToOrig map[string]string
}

func New() *URLRepository {
	return &URLRepository{
		origToShort: make(map[string]string),
		shortToOrig: make(map[string]string),
	}
}

func (r *URLRepository) GetShortByOrig(orig string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.origToShort[orig]
}

func (r *URLRepository) GetOrigByShort(short string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.shortToOrig[short]
}

func (r *URLRepository) SaveIfNotTaken(orig, short string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing := r.origToShort[orig]; existing != "" {
		return existing, nil
	}
	if r.shortToOrig[short] != "" {
		return "", errors.New("short is already taken")
	}
	r.origToShort[orig] = short
	r.shortToOrig[short] = orig
	return short, nil
}
