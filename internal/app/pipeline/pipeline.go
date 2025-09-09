package pipeline

import (
	"context"
	"fmt"
	"github.com/torys877/vectrain/internal/config"
	"github.com/torys877/vectrain/pkg/types"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// ----------------- Pipeline -----------------

type Pipeline struct {
	mode string
	//cfg      *config.Config
	cfg      *config.AppConfig
	source   types.Source
	embedder types.Embedder
	storage  types.Storage
	running  atomic.Bool
}

type EmbeddingItem struct {
	item   *types.Entity
	vector []float32
}

func NewPipeline(cfg *config.Config, src types.Source, emb types.Embedder, storage types.Storage) *Pipeline {
	p := &Pipeline{
		source:   src,
		embedder: emb,
		storage:  storage,
		cfg:      cfg.App,
	}
	p.running.Store(false)
	p.mode = cfg.App.Pipeline.Mode

	return p
}

func (p *Pipeline) Run(ctx context.Context) error {
	fmt.Println("Run pipeline")
	fmt.Println("Source Connecting...")
	err := p.source.Connect()
	if err != nil {
		return err
	}
	defer p.source.Close()
	fmt.Println("Source Connected")

	fmt.Println("Storage Connecting...")
	err = p.storage.Connect()
	if err != nil {
		return err
	}
	defer p.storage.Close()
	fmt.Println("Storage Connected")

	switch p.mode {
	case "performance":
		fmt.Println("Performance mode running...")
		return p.runWithPerformance(ctx)
	case "reliability":
		fmt.Println("Reliability mode running...")
		return p.runWithReliability(ctx)
	default:
		fmt.Println("No mode find :(")
		return nil
	}
}

// fast
func (p *Pipeline) runWithPerformance(ctx context.Context) error {
	messageCh := make(chan *types.Entity, p.cfg.Pipeline.SourceBatchSize*2)
	embeddingCh := make(chan *types.Entity, p.cfg.Pipeline.SourceBatchSize*2)

	wg := sync.WaitGroup{}

	// 1 - run retreive from source
	wg.Add(1)
	go p.retrieveMsgs(ctx, messageCh, &wg)

	// 2 - run embedders
	for i := 0; i < p.cfg.Pipeline.EmbedderWorkersCnt; i++ {
		wg.Add(1)
		go p.embedMsg(ctx, messageCh, embeddingCh, &wg)
	}

	// 3 - run storage save
	wg.Add(1)
	go p.saveMsgs(ctx, embeddingCh, &wg)

	wg.Wait()
	return nil
}

func (p *Pipeline) runWithReliability(ctx context.Context) error {
	messageCh := make(chan *types.Entity, p.cfg.Pipeline.SourceBatchSize*2)
	embeddingCh := make(chan *types.Entity, p.cfg.Pipeline.StorageBatchSize*2)

	var wg sync.WaitGroup
	storageErrCh := make(chan error, 1)

	// Embedder workers
	for i := 0; i < p.cfg.Pipeline.EmbedderWorkersCnt; i++ {
		wg.Add(1)
		go func() {
			p.embedMsg(ctx, messageCh, embeddingCh, &wg)
		}()
	}

	// Storage processor
	wg.Add(1)
	go func() {
		defer wg.Done()
		//defer close(embeddingCh) // Закрываем только после обработки всех данных

		vectors := make([]*types.Entity, 0, p.cfg.Pipeline.StorageBatchSize)

		for {
			select {
			case <-ctx.Done():
				// Сохраняем оставшиеся данные при shutdown
				if len(vectors) > 0 {
					if err := p.processStorageBatch(ctx, vectors); err != nil {
						storageErrCh <- err
					}
				}
				return

			case item, ok := <-embeddingCh:
				if !ok {
					// Канал закрыт, сохраняем последний батч
					if len(vectors) > 0 {
						if err := p.processStorageBatch(ctx, vectors); err != nil {
							storageErrCh <- err
						}
					}
					return
				}

				vectors = append(vectors, item)

				if len(vectors) >= p.cfg.Pipeline.StorageBatchSize {
					if err := p.processStorageBatch(ctx, vectors); err != nil {
						storageErrCh <- err
						// Не прерываем работу, только логируем ошибку
						log.Printf("Storage batch error: %v", err)
					}
					vectors = vectors[:0] // Очищаем slice
				}
			}
		}
	}()

	// Message producer
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(messageCh)

		for {
			select {
			case <-ctx.Done():
				return

			default:
				if !p.running.Load() {
					time.Sleep(200 * time.Millisecond)
					continue
				}

				batch, err := p.source.FetchBatch(ctx, p.cfg.Pipeline.SourceBatchSize)
				if err != nil {
					log.Printf("Fetch error: %v", err)
					continue
				}

				if len(batch) == 0 {
					continue // Ждем следующей итерации
				}

				log.Printf("Fetched %d messages", len(batch))

				for _, item := range batch {
					select {
					case <-ctx.Done():
						return
					case messageCh <- item:
					}
				}
			}
		}
	}()

	// Ожидаем завершения или ошибки
	select {
	case <-ctx.Done():
		log.Println("Context cancelled, waiting for workers to finish...")
		wg.Wait()
		return ctx.Err()

	case err := <-storageErrCh:
		log.Printf("Critical storage error: %v", err)
		return err
	}
}

// Вспомогательная функция для обработки батча хранилища
func (p *Pipeline) processStorageBatch(ctx context.Context, batch []*types.Entity) error {
	if err := p.storage.StoreBatch(ctx, batch); err != nil {
		return fmt.Errorf("storage error: %w", err)
	}

	// Фильтруем успешно обработанные элементы (без ошибок эмбеддинга)
	successItems := make([]*types.Entity, 0, len(batch))
	for _, item := range batch {
		if item.Err == nil {
			successItems = append(successItems, item)
		}
	}

	if len(successItems) > 0 {
		//if err := p.source.AfterProcessHook(ctx, successItems); err != nil {
		//	return fmt.Errorf("after process hook error: %w", err)
		//}
	}

	return nil
}

func (p *Pipeline) retrieveMsgs(ctx context.Context, messageCh chan *types.Entity, wg *sync.WaitGroup) error {
	defer close(messageCh)
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			batch, err := p.source.FetchBatch(ctx, p.cfg.Pipeline.SourceBatchSize)
			if err != nil {
				log.Printf("Error fetching batch: %v", err)
				return nil
			}
			if len(batch) == 0 {
				// Нет данных - делаем паузу перед следующей попыткой
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(1 * time.Second): // Настраиваемый интервал
				}
				continue
			}

			// Отправляем все элементы батча с проверкой контекста
			for _, item := range batch {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case messageCh <- item:
				}
			}
		}
	}
}
func (p *Pipeline) embedMsg(ctx context.Context, messageChIn <-chan *types.Entity, embeddingChOut chan<- *types.Entity, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case item, ok := <-messageChIn:
			if !ok {
				return
			}

			vec, err := p.embedder.Embed(ctx, item.Text)
			if err != nil {
				//log.Printf("Embedding error: %v", err)
				// Решаем что делать с ошибкой:
				// 1. Пропустить: continue
				// 2. Отправить с ошибкой:
				item.Err = err
			} else {
				item.Vector = vec
			}

			// Пытаемся отправить с таймаутом
			select {
			case <-ctx.Done():
				return
			case embeddingChOut <- item:
			}
		}
	}
}
func (p *Pipeline) saveMsgs(ctx context.Context, embeddingCh <-chan *types.Entity, wg *sync.WaitGroup) error {
	defer wg.Done()

	var batchItems []*types.Entity
	var batchOffsets []*types.Entity // для commit

	// Вспомогательная функция для обработки батча
	processBatch := func() error {
		if len(batchItems) == 0 {
			return nil
		}

		if err := p.storage.StoreBatch(ctx, batchItems); err != nil {
			log.Printf("VectorDB error: %v", err)
			return err
		}

		// Правильно создаем slice для sourceEntities
		sourceEntities := make([]*types.Entity, 0, len(batchItems))
		for _, item := range batchItems {
			if item.Err == nil { // Отправляем только успешные элементы
				sourceEntities = append(sourceEntities, item)
			}
		}

		if len(sourceEntities) > 0 {
			//if err := p.source.AfterProcessHook(ctx, sourceEntities); err != nil {
			//	log.Printf("AfterProcessHook error: %v", err)
			//	return err
			//}
		}

		batchItems = batchItems[:0]
		batchOffsets = batchOffsets[:0]
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			// Обрабатываем оставшиеся элементы при отмене
			if err := processBatch(); err != nil {
				log.Printf("Error processing final batch on shutdown: %v", err)
			}
			return ctx.Err()

		case res, ok := <-embeddingCh:
			if !ok {
				// Канал закрыт, обрабатываем оставшийся батч
				if err := processBatch(); err != nil {
					return err
				}
				return nil
			}

			// Добавляем в батч, даже если есть ошибка (для отслеживания)
			batchItems = append(batchItems, res)

			if len(batchItems) >= p.cfg.Pipeline.StorageBatchSize {
				if err := processBatch(); err != nil {
					return err
				}
			}
		}
	}
}

func (p *Pipeline) Start() {
	fmt.Println("Pipeline Starting...")
	p.running.Store(true)
}

func (p *Pipeline) Stop() {
	fmt.Println("Pipeline Stopping...")
	p.running.Store(false)
}

func (p *Pipeline) Configuration() *config.AppConfig {
	return p.cfg
}
