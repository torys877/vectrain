package pipeline

import (
	"context"
	"github.com/torys877/vectrain/pkg/types"
	"log"
	"sync"
)

// ----------------- Pipeline -----------------

type PipelineConfig struct {
	BatchSize       int
	EmbedWorkers    int
	VectorBatchSize int
}

type Pipeline struct {
	cfg      PipelineConfig
	source   types.Source
	embedder types.Embedder
	storage  types.Storage
}

func NewPipeline(src types.Source, emb types.Embedder, storage types.Storage, cfg PipelineConfig) *Pipeline {
	return &Pipeline{
		source:   src,
		embedder: emb,
		storage:  storage,
		cfg:      cfg,
	}
}

type EmbeddingItem struct {
	item   *types.Entity
	vector []float32
}

func (p *Pipeline) Run(ctx context.Context) error {
	messageCh := make(chan *types.Entity, p.cfg.BatchSize*2)
	embeddingCh := make(chan *types.VectorEntity, p.cfg.BatchSize*2)

	wg := sync.WaitGroup{}

	// 1 - run retreive from source
	wg.Add(1)
	go func() {
		defer close(messageCh)
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				batch, err := p.source.FetchBatch(ctx, p.cfg.BatchSize)
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

	// 2 - run embedders
	for i := 0; i < p.cfg.EmbedWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range messageCh {
				vec, err := p.embedder.Embed(ctx, item.Text)
				if err != nil {
					log.Printf("Embedding error: %v", err)
					continue
				}

				vectorEntity := &types.VectorEntity{
					Entity: *item,
					Vector: vec,
				}

				embeddingCh <- vectorEntity
			}
		}()
	}

	// 3 - run storage save
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(embeddingCh)

		var batchItems []*types.VectorEntity
		var batchEmbeddings [][]float32
		var batchOffsets []*types.Entity // для commit

		for res := range embeddingCh {
			batchItems = append(batchItems, res)

			if len(batchItems) >= p.cfg.VectorBatchSize {
				if err := p.storage.StoreBatch(ctx, batchItems); err != nil {
					log.Printf("VectorDB error: %v", err)
				} else {
					sourceEntities := make([]*types.Entity, len(batchItems))
					for _, item := range batchItems {
						sourceEntities = append(sourceEntities, &item.Entity)
					}
					err = p.source.AfterProcess(ctx, sourceEntities, []*types.ErrorEntity{}) // TODO handle error items
					if err != nil {
						return //TODO handle error
					}
				}
				batchItems = batchItems[:0]
				batchEmbeddings = batchEmbeddings[:0]
				batchOffsets = batchOffsets[:0]
			}
		}

		// остатки
		if len(batchItems) > 0 { // TODO REMOVE DUPLICATE CODE
			if err := p.storage.StoreBatch(ctx, batchItems); err != nil {
				log.Printf("VectorDB error: %v", err)
			} else {
				sourceEntities := make([]*types.Entity, len(batchItems))
				for _, item := range batchItems {
					sourceEntities = append(sourceEntities, &item.Entity)
				}
				err = p.source.AfterProcess(ctx, sourceEntities, []*types.ErrorEntity{}) // TODO handle error items
				if err != nil {
					return //TODO handle error
				}
			}
		}
	}()

	wg.Wait()
	return nil
}
