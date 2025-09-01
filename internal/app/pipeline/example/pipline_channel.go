package pipeline

import (
	"context"
	"log"
	"sync"
)

// ----------------- Интерфейсы -----------------

type Source interface {
	FetchBatch(ctx context.Context, size int) ([]any, error)
	Close() error
}

type Embedder interface {
	Embed(ctx context.Context, item any) ([]float32, error)
}

type VectorDB interface {
	UpsertBatch(ctx context.Context, items []any, embeddings [][]float32) error
}

// ----------------- Pipeline -----------------

type PipelineConfig struct {
	BatchSize       int
	EmbedWorkers    int
	VectorBatchSize int
}

type Pipeline struct {
	src   Source
	emb   Embedder
	vecDB VectorDB
	cfg   PipelineConfig
}

func NewPipeline(src Source, emb Embedder, vecDB VectorDB, cfg PipelineConfig) *Pipeline {
	return &Pipeline{
		src:   src,
		emb:   emb,
		vecDB: vecDB,
		cfg:   cfg,
	}
}

// ----------------- Асинхронный Run -----------------

func (p *Pipeline) Run(ctx context.Context) error {
	// Канал для сообщений, которые нужно эмбедировать
	messageCh := make(chan any, p.cfg.BatchSize*2)
	// Канал для результатов эмбединга
	embeddingCh := make(chan struct {
		item      any
		embedding []float32
	}, p.cfg.BatchSize*2)

	wg := sync.WaitGroup{}

	// 1) Горутин для чтения из источника
	wg.Add(1)
	go func() {
		defer close(messageCh)
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				batch, err := p.src.FetchBatch(ctx, p.cfg.BatchSize)
				if err != nil {
					log.Printf("Error fetching batch: %v", err)
					continue
				}
				if len(batch) == 0 {
					return // источник пуст
				}
				for _, item := range batch {
					messageCh <- item
				}
			}
		}
	}()

	// 2) Горутинов для эмбедера
	for i := 0; i < p.cfg.EmbedWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range messageCh {
				vec, err := p.emb.Embed(ctx, item)
				if err != nil {
					log.Printf("Embedding error: %v", err)
					continue
				}
				embeddingCh <- struct {
					item      any
					embedding []float32
				}{item, vec}
			}
		}()
	}

	// 3) Горутин для записи в Qdrant батчами
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(embeddingCh)

		var batchItems []any
		var batchEmbeddings [][]float32

		for res := range embeddingCh {
			batchItems = append(batchItems, res.item)
			batchEmbeddings = append(batchEmbeddings, res.embedding)

			// Если накопился батч нужного размера, записываем
			if len(batchItems) >= p.cfg.VectorBatchSize {
				if err := p.vecDB.UpsertBatch(ctx, batchItems, batchEmbeddings); err != nil {
					log.Printf("VectorDB error: %v", err)
				}
				batchItems = batchItems[:0]
				batchEmbeddings = batchEmbeddings[:0]
			}
		}

		// Записываем остатки
		if len(batchItems) > 0 {
			if err := p.vecDB.UpsertBatch(ctx, batchItems, batchEmbeddings); err != nil {
				log.Printf("VectorDB error: %v", err)
			}
		}
	}()

	wg.Wait()
	return nil
}
