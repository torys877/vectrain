package pipeline

import (
	"context"
	"fmt"
	"github.com/torys877/vectrain/internal/config"
	"github.com/torys877/vectrain/internal/utils"
	"github.com/torys877/vectrain/pkg/types"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// ----------------- Pipeline -----------------

type Option func(*Pipeline)

type Pipeline struct {
	//mode     string
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

func NewPipeline(opts ...Option) *Pipeline {
	p := &Pipeline{}
	p.running.Store(false)
	//p.mode = cfg.App.Pipeline.Mode

	for _, opt := range opts {
		opt(p)
	}

	return p
}

func (p *Pipeline) Run(ctx context.Context) error {
	fmt.Println("Run pipeline")
	fmt.Println("Validate pipeline configuration")
	if err := p.validate(); err != nil {
		return err
	}

	fmt.Println("Prepare pipeline")
	if err := p.prepare(); err != nil {
		return err
	}

	defer func() {
		if err := p.source.Close(); err != nil {
			fmt.Println("source was not closed correctly:", err)
		}
	}()
	defer func() {
		if err := p.storage.Close(); err != nil {
			fmt.Println("storage was not closed correctly:", err)
		}
	}()

	return p.runWithReliability(ctx)
}

func (p *Pipeline) validate() error {
	if p.cfg == nil {
		return fmt.Errorf("configuration is nil")
	}
	if utils.IsNil(p.source) {
		return fmt.Errorf("source is nil")
	}
	if utils.IsNil(p.embedder) {
		return fmt.Errorf("embedder is nil")
	}
	if utils.IsNil(p.storage) {
		return fmt.Errorf("storage is nil")
	}
	return nil
}

func (p *Pipeline) prepare() error {
	fmt.Println("Source Connecting...")
	if err := p.source.Connect(); err != nil {
		return fmt.Errorf("source connect failed: %w", err)
	}
	fmt.Println("Source Connected")

	fmt.Println("Storage Connecting...")
	if err := p.storage.Connect(); err != nil {
		return fmt.Errorf("storage connect failed: %w", err)
	}
	fmt.Println("Storage Connected")

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

		vectors := make([]*types.Entity, 0, p.cfg.Pipeline.StorageBatchSize)

		for {
			select {
			case <-ctx.Done():
				if len(vectors) > 0 {
					if err := p.storeBatch(ctx, vectors); err != nil {
						storageErrCh <- err
					}
				}
				return

			case item, ok := <-embeddingCh:
				if !ok {
					if len(vectors) > 0 {
						if err := p.storeBatch(ctx, vectors); err != nil {
							storageErrCh <- err
						}
					}
					return
				}

				vectors = append(vectors, item)

				if len(vectors) >= p.cfg.Pipeline.StorageBatchSize {
					if err := p.storeBatch(ctx, vectors); err != nil {
						storageErrCh <- err
						log.Printf("storage batch error: %v", err)
					}
					vectors = vectors[:0] // Очищаем slice
				}
			}
		}
	}()

	// Message consumer, consume and send in embedder
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

	// wait for workers to finish
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

func (p *Pipeline) storeBatch(ctx context.Context, batch []*types.Entity) error {
	if err := p.storage.StoreBatch(ctx, batch); err != nil {
		return fmt.Errorf("storage error: %w", err)
	}

	allItems := make([]*types.Entity, 0, len(batch))
	for _, item := range batch {
		allItems = append(allItems, item)
	}

	if len(allItems) > 0 {
		if err := p.source.AfterProcessHook(ctx, allItems); err != nil {
			return fmt.Errorf("after process hook error: %w", err)
		}
	}

	return nil
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
