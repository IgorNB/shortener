package repository

import "sync"

type URLRepository struct {
	mu          sync.RWMutex
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
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.origToShort[orig]
}

func (r *URLRepository) GetOrigByShort(short string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.shortToOrig[short]
}

// SaveIfNotTaken атомарно проверяет, что для orig ещё нет записи и short свободен,
// и сохраняет пару. Возвращает актуальный short и true при успехе.
// Если для orig уже есть запись — возвращает существующий short и true.
// Если short занят другим orig — возвращает "" и false (нужен новый кандидат).
func (r *URLRepository) SaveIfNotTaken(orig, short string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing := r.origToShort[orig]; existing != "" {
		return existing, true
	}
	if r.shortToOrig[short] != "" {
		return "", false
	}
	r.origToShort[orig] = short
	r.shortToOrig[short] = orig
	return short, true
}
