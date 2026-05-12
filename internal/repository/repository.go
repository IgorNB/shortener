package repository

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/IgorNB/shortener/internal/config/logger"
	"github.com/IgorNB/shortener/internal/model/entity"
)

// URLRepository
/*
* URLRepository
Чтение всегда быстрое из RAM под Lock mu и не блокируется записью в файл

запись выполняется в 2 шага:
 1. под persistentMu выполняется проверка на дубликаты и занятость short,

запись в файл и обновление persistent-структур (это единственный источник истины)
 2. после успешной записи под mu обновляется кэш отдельным атомарным шагом

в момент между обновлением persistent и обновлением кэш возможно кратковременное
отставание кэша (пока хотя бы один вызов SaveIfNotTaken не завершится без ошибок), что допустимо и не влияет на корректность, т.к.
 1. параллельные вызовы GetShortByOrig / GetOrigByShort ещё не знают short ссылку (еще ни один SaveIfNotTaken не завершился) => не обязаны получить по ней из кэша оригинальную ссылку
 2. параллельные вызовы SaveIfNotTaken будут защищены от дублей `orig` /присвоения одинакового `short` через persistentMu

архитектура сознательно использует 2x структуру хранения для обеспечения быстрых чтений при гарантированной корректности через persistent слой
*/
type URLRepository struct {
	//"Кэш". mu - быстрый Lock. Под ним только операции в RAM
	mu          sync.Mutex
	origToShort map[string]string
	shortToOrig map[string]string

	//Персистентность. persistentMu - медленный лок. Под ним ходим на диск и меняем "копию" кэша в persistent мапах
	persistentMu          sync.Mutex
	persistentOrigToShort map[string]string
	persistentShortToOrig map[string]string
	persistentStorage     *os.File
}

func New(fileStoragePath string) *URLRepository {
	persistent, err := newPersistent(fileStoragePath, 1024*1024)
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("App init failed - cannot create persistent storage")
	}
	return persistent
}

func newPersistent(path string, scannerBufferSize int) (*URLRepository, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	repo := &URLRepository{
		origToShort: make(map[string]string),
		shortToOrig: make(map[string]string),

		persistentOrigToShort: make(map[string]string),
		persistentShortToOrig: make(map[string]string),
		persistentStorage:     file,
	}

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, scannerBufferSize)
	scanner.Buffer(buf, scannerBufferSize)

	for scanner.Scan() {
		var rec entity.UrlPair
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}

		orig := rec.OriginalURL
		short := rec.ShortURL

		repo.origToShort[orig] = short
		repo.shortToOrig[short] = orig
		repo.persistentOrigToShort[orig] = short
		repo.persistentShortToOrig[short] = orig
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return repo, nil
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

func (r *URLRepository) SaveIfNotTaken(orig, short string) error {
	if err := r.persistWithDuplicateCheck(orig, short); err != nil {
		return err
	}

	r.appendCache(orig, short)
	return nil
}

func (r *URLRepository) persistWithDuplicateCheck(orig, short string) error {
	r.persistentMu.Lock()
	defer r.persistentMu.Unlock()
	if existing := r.persistentOrigToShort[orig]; existing != "" {
		return nil
	}
	if r.persistentShortToOrig[short] != "" {
		return errors.New("short is already taken")
	}
	if err := r.appendFile(orig, short); err != nil {
		return errors.New("save failed")
	}
	r.persistentOrigToShort[orig] = short
	r.persistentShortToOrig[short] = orig
	return nil
}

func (r *URLRepository) appendFile(orig string, short string) error {
	rec := entity.UrlPair{
		ShortURL:    short,
		OriginalURL: orig,
	}

	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	if _, err := r.persistentStorage.WriteString(string(data) + "\n"); err != nil {
		logger.Log.Error().Err(err).Msg("file write failed")
		return err
	}
	if err := r.persistentStorage.Sync(); err != nil {
		logger.Log.Error().Err(err).Msg("file flush failed")
		return err
	}
	return nil
}

func (r *URLRepository) appendCache(orig, short string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.origToShort[orig] = short
	r.shortToOrig[short] = orig
}
