package pipeline

import (
	"context"
	"fmt"
	"github.com/torys877/vectrain/internal/config"
	"github.com/torys877/vectrain/internal/infra/logger"
	"github.com/torys877/vectrain/internal/utils"
	"github.com/torys877/vectrain/pkg/types"
	"go.uber.org/zap"
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

	for _, opt := range opts {
		opt(p)
	}

	return p
}

func (p *Pipeline) Run(ctx context.Context) error {
	logger.Info("running pipeline")
	if err := p.validate(); err != nil {
		return err
	}

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

	return p.runPipeline(ctx)
}

func (p *Pipeline) runPipeline(ctx context.Context) error {
	messageCh := make(chan *types.Entity, p.cfg.Pipeline.SourceBatchSize*2)
	embeddingCh := make(chan *types.Entity, p.cfg.Pipeline.StorageBatchSize*2)

	var wg sync.WaitGroup
	storageErrCh := make(chan error, 1)

	// Embedder workers
	for i := 0; i < p.cfg.Pipeline.EmbedderWorkersCnt; i++ {
		wg.Add(1)
		go func() {
			p.embed(ctx, messageCh, embeddingCh, &wg)
		}()
	}

	// Storage processor
	wg.Add(1)
	go p.store(ctx, embeddingCh, storageErrCh, &wg)

	// Message consumer, consume and send in embedder
	wg.Add(1)
	go p.consume(ctx, messageCh, &wg)

	// wait for workers to finish
	select {
	case <-ctx.Done():
		logger.Info("context cancelled, waiting for workers to finish...")
		wg.Wait()
		return ctx.Err()

	case err := <-storageErrCh:
		// critical error, stop pipeline
		return err
	}
}

func (p *Pipeline) validate() error {
	logger.Info("validate pipeline configuration")
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
	logger.Info("prepare pipeline")
	logger.Info("source connecting...")
	if err := p.source.Connect(); err != nil {
		return fmt.Errorf("source connect failed: %w", err)
	}
	logger.Info("source connected")

	logger.Info("storage connecting...")
	if err := p.storage.Connect(); err != nil {
		return fmt.Errorf("storage connect failed: %w", err)
	}
	logger.Info("storage connected")

	return nil
}

func (p *Pipeline) consume(
	ctx context.Context,
	messageCh chan<- *types.Entity,
	wg *sync.WaitGroup,
) {
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

			batch, err := p.source.Fetch(ctx, p.cfg.Pipeline.SourceBatchSize) // handle error, stop?
			if err != nil {
				logger.Error("fetch error", zap.Error(err))
				continue
			}

			if len(batch) == 0 {
				continue
			}

			if err = p.source.BeforeProcessHook(ctx, batch); err != nil {
				logger.Warn("before process hook error", zap.Error(err)) // not critical, continue
			}

			for _, item := range batch {
				select {
				case <-ctx.Done():
					return
				case messageCh <- item:
				}
			}
		}
	}
}
func (p *Pipeline) store(ctx context.Context,
	embeddingCh <-chan *types.Entity,
	storageErrCh chan<- error,
	wg *sync.WaitGroup,
) {
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
				}
				vectors = vectors[:0]
			}
		}
	}
}

func (p *Pipeline) storeBatch(ctx context.Context, batch []*types.Entity) error {
	if err := p.storage.Store(ctx, batch); err != nil {
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

func (p *Pipeline) embed(
	ctx context.Context,
	messageChIn <-chan *types.Entity,
	embeddingChOut chan<- *types.Entity,
	wg *sync.WaitGroup,
) {
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

			select {
			case <-ctx.Done():
				return
			case embeddingChOut <- item:
			}
		}
	}
}

func (p *Pipeline) Start() {
	logger.Info("starting pipeline...")
	p.running.Store(true)
}

func (p *Pipeline) Stop() {
	logger.Info("stopping pipeline...")
	p.running.Store(false)
}

func (p *Pipeline) Configuration() *config.AppConfig {
	return p.cfg
}
