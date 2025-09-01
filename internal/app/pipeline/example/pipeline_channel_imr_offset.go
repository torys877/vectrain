package pipeline

import (
	"context"
	"log"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// ----------------- Интерфейсы -----------------

type Source interface {
	// FetchBatch возвращает до size элементов, каждый элемент содержит offset и partition
	FetchBatch(ctx context.Context, size int) ([]KafkaItem, error)
	Close() error
}

type Embedder interface {
	Embed(ctx context.Context, item any) ([]float32, error)
}

type VectorDB interface {
	UpsertBatch(ctx context.Context, items []any, embeddings [][]float32) error
}

// ----------------- Типы -----------------

type KafkaItem struct {
	Msg       any
	Partition int32
	Offset    int64
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
	cons  *kafka.Consumer // нужен для commit
}

func NewPipeline(src Source, emb Embedder, vecDB VectorDB, cfg PipelineConfig, cons *kafka.Consumer) *Pipeline {
	return &Pipeline{
		src:   src,
		emb:   emb,
		vecDB: vecDB,
		cfg:   cfg,
		cons:  cons,
	}
}

// ----------------- Асинхронный Run -----------------

func (p *Pipeline) Run(ctx context.Context) error {
	messageCh := make(chan KafkaItem, p.cfg.BatchSize*2)
	embeddingCh := make(chan struct {
		item      KafkaItem
		embedding []float32
	}, p.cfg.BatchSize*2)

	wg := sync.WaitGroup{}

	// 1) Горутин чтения из источника
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
					return
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
				vec, err := p.emb.Embed(ctx, item.Msg)
				if err != nil {
					log.Printf("Embedding error: %v", err)
					continue
				}
				embeddingCh <- struct {
					item      KafkaItem
					embedding []float32
				}{item, vec}
			}
		}()
	}

	// 3) Горутин для записи в Qdrant батчами и коммита offset
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(embeddingCh)

		var batchItems []any
		var batchEmbeddings [][]float32
		var batchOffsets []KafkaItem // для commit

		for res := range embeddingCh {
			batchItems = append(batchItems, res.item.Msg)
			batchEmbeddings = append(batchEmbeddings, res.embedding)
			batchOffsets = append(batchOffsets, res.item)

			if len(batchItems) >= p.cfg.VectorBatchSize {
				if err := p.vecDB.UpsertBatch(ctx, batchItems, batchEmbeddings); err != nil {
					log.Printf("VectorDB error: %v", err)
				} else {
					// commit offsets только после успешного upsert
					p.commitOffsets(batchOffsets)
				}
				batchItems = batchItems[:0]
				batchEmbeddings = batchEmbeddings[:0]
				batchOffsets = batchOffsets[:0]
			}
		}

		// остатки
		if len(batchItems) > 0 {
			if err := p.vecDB.UpsertBatch(ctx, batchItems, batchEmbeddings); err != nil {
				log.Printf("VectorDB error: %v", err)
			} else {
				p.commitOffsets(batchOffsets)
			}
		}
	}()

	wg.Wait()
	return nil
}

// ----------------- commit helper -----------------

func (p *Pipeline) commitOffsets(items []KafkaItem) {
	if len(items) == 0 || p.cons == nil {
		return
	}

	offsets := make([]kafka.TopicPartition, 0, len(items))
	for _, item := range items {
		offsets = append(offsets, kafka.TopicPartition{
			Partition: item.Partition,
			Offset:    kafka.Offset(item.Offset + 1), // следующий offset
		})
	}

	_, err := p.cons.CommitOffsets(offsets)
	if err != nil {
		log.Printf("Failed to commit offsets: %v", err)
	}
}
