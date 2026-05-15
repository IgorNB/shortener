package repository

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/IgorNB/shortener/internal/middleware/logger"
	"github.com/IgorNB/shortener/internal/model/entity"
	"github.com/google/uuid"
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
// URLRepository
/*
URLRepository — append-only хранилище коротких ссылок с RAM-кэшем.

Чтение всегда быстрое из RAM под mu и не блокируется записью в файл.

Запись выполняется строго под persistentMu:
 1. выполняется проверка уникальности orig/short по текущему состоянию RAM (на этом этапе дополнительно ставим и сразу снимаем mu lock)
 2. выполняется запись в append-only файл (источник истины)
 3. выполняется обновление RAM-кэша (на этом этапе дополнительно ставим и сразу снимаем mu lock)

в момент между обновлением файла и обновлением кэша возможно кратковременное
отставание кэша (пока хотя бы один вызов SaveIfNotTaken не завершится без ошибок), что допустимо и не влияет на корректность, т.к.
 1. параллельные вызовы GetShortByOrig / GetOrigByShort ещё не знают short ссылку (еще ни один SaveIfNotTaken не завершился) => не обязаны получить по ней из кэша оригинальную ссылку
 2. параллельные вызовы SaveIfNotTaken будут защищены от дублей `orig` /присвоения одинакового `short` через persistentMu

Модель оптимизирована под append-only storage и высокий RPS чтений.
*/
type URLRepository struct {
	//"Кэш". mu - быстрый Lock. Под ним только ЧИТАЕМ кэш
	mu          sync.Mutex
	origToShort map[string]string
	shortToOrig map[string]string

	//Персистентность. persistentMu - медленный лок. Под ним ходим на диск и ТОЛЬКО под ним ПИШЕМ кэш (дополнительно ставим mu при обращении к кэшу)
	persistentMu      sync.Mutex
	persistentStorage *os.File
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
		origToShort:       make(map[string]string),
		shortToOrig:       make(map[string]string),
		persistentStorage: file,
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
	r.persistentMu.Lock()
	defer r.persistentMu.Unlock()
	if err := r.check(orig, short); err != nil {
		return err
	}
	if err := r.appendFile(orig, short); err != nil {
		return errors.New("save failed")
	}
	r.appendCache(orig, short)
	return nil
}

func (r *URLRepository) check(orig, short string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing := r.origToShort[orig]; existing != "" {
		return errors.New("orig already added")
	}
	if r.shortToOrig[short] != "" {
		return errors.New("short is already taken")
	}
	return nil
}

func (r *URLRepository) appendCache(orig, short string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.origToShort[orig] = short
	r.shortToOrig[short] = orig
}

func (r *URLRepository) appendFile(orig string, short string) error {
	uuidV7, err := uuid.NewV7()
	if err != nil {
		logger.Log.Error().Err(err).Msg("uuid generation err")
	}
	rec := entity.UrlPair{
		Uuid:        uuidV7,
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
