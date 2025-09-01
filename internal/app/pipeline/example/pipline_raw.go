package pipeline

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// ------------------- Интерфейсы -------------------

type Source interface {
	// FetchBatch возвращает до size элементов
	FetchBatch(ctx context.Context, size int) ([]any, error)
	Name() string
	Close() error
}

type Embedder interface {
	// Embed принимает единичное сообщение и возвращает вектор или ошибку
	Embed(ctx context.Context, item any) ([]float32, error)
}

type VectorDB interface {
	// UpsertBatch принимает батч объектов с эмбединговыми векторами
	UpsertBatch(ctx context.Context, items []any, embeddings [][]float32) error
}

// ------------------- Pipeline -------------------

type PipelineConfig struct {
	BatchSize    int // сколько сообщений вытаскивать за раз
	EmbedWorkers int // сколько горутин эмбедера
}

type Pipeline struct {
	src   Source
	emb   Embedder
	vecDB VectorDB
	cfg   PipelineConfig
}

// NewPipeline конструктор
func NewPipeline(src Source, emb Embedder, vecDB VectorDB, cfg PipelineConfig) *Pipeline {
	return &Pipeline{
		src:   src,
		emb:   emb,
		vecDB: vecDB,
		cfg:   cfg,
	}
}

// Run запускает цикл обработки сообщений
func (p *Pipeline) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// 1) Получаем batch сообщений
			batch, err := p.src.FetchBatch(ctx, p.cfg.BatchSize)
			if err != nil {
				log.Printf("Error fetching batch: %v", err)
				continue
			}
			if len(batch) == 0 {
				// Источник пуст, можно завершить
				return nil
			}

			// 2) Отправляем в эмбедер с X горутинами
			embeddings := make([][]float32, len(batch))
			errs := make([]error, len(batch))

			wg := sync.WaitGroup{}
			tasks := make(chan int, len(batch)) // индексы задач

			for i := 0; i < len(batch); i++ {
				tasks <- i
			}
			close(tasks)

			workerCount := p.cfg.EmbedWorkers
			if workerCount > len(batch) {
				workerCount = len(batch)
			}

			for w := 0; w < workerCount; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for idx := range tasks {
						vec, err := p.emb.Embed(ctx, batch[idx])
						if err != nil {
							errs[idx] = err
							log.Printf("Error embedding item %d: %v", idx, err)
							continue
						}
						embeddings[idx] = vec
					}
				}()
			}

			wg.Wait()

			// 3) Формируем батч для векторной базы, отбрасываем ошибки
			var successItems []any
			var successEmbeddings [][]float32
			for i := range batch {
				if errs[i] == nil {
					successItems = append(successItems, batch[i])
					successEmbeddings = append(successEmbeddings, embeddings[i])
				}
			}

			if len(successItems) > 0 {
				if err := p.vecDB.UpsertBatch(ctx, successItems, successEmbeddings); err != nil {
					log.Printf("Error upserting to vector DB: %v", err)
				}
			}
		}
	}
}
