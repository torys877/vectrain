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

// all messages are processed
func (p *Pipeline) runWithReliability(ctx context.Context) error {
	messageCh := make(chan *types.Entity, p.cfg.Pipeline.SourceBatchSize*2)
	embeddingCh := make(chan *types.Entity, p.cfg.Pipeline.StorageBatchSize*2)
	wg := sync.WaitGroup{}
	//vectors := make([]*types.Entity, 0, p.cfg.Pipeline.SourceBatchSize)

	// embedder workers
	for i := 0; i < p.cfg.Pipeline.EmbedderWorkersCnt; i++ {
		wg.Add(1)
		go func() {
			p.embedMsg(ctx, messageCh, embeddingCh, &wg)
		}()
	}

	// store vectors in storage
	go func() {
		vectors := make([]*types.Entity, 0, p.cfg.Pipeline.StorageBatchSize)
		for {
			select {
			case <-ctx.Done():
				return
			case item, ok := <-embeddingCh:
				if !ok {
					return
				}
				vectors = append(vectors, item)
				if len(vectors) >= p.cfg.Pipeline.StorageBatchSize {
					fmt.Println("Store Vectors")
					if err := p.storage.StoreBatch(ctx, vectors); err != nil {
						log.Printf("VectorDB error: %v", err)
						return
					}
					fmt.Println("Store Vectors Done")
					err := p.source.AfterProcessHook(ctx, vectors)
					if err != nil {
						log.Printf("Error after process hook: %v", err)
					}

					vectors = make([]*types.Entity, 0, p.cfg.Pipeline.StorageBatchSize)
				}
			}
		}
	}()

	// consumer takes data from source
	go func() {
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

				fmt.Println("Pipeline Started")
				fmt.Println("Retrieve Data From Source")

				batch, err := p.source.FetchBatch(ctx, p.cfg.Pipeline.SourceBatchSize)
				if err != nil {
					log.Printf("error: %v", err)
					continue
				}

				fmt.Println("Retrieve Data From Source Done, Batch Size: ", len(batch))

				if len(batch) == 0 {
					// sleep if we don't have any messages
					time.Sleep(100 * time.Millisecond)
					continue
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
	}()

	<-ctx.Done()
	wg.Wait()
	close(embeddingCh)

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
				continue
			}
			if len(batch) == 0 {
				return nil
			}
			for _, item := range batch {
				messageCh <- item
			}
		}
	}
}

func (p *Pipeline) embedMsg(ctx context.Context, messageChIn <-chan *types.Entity, embeddingChOut chan<- *types.Entity, wg *sync.WaitGroup) {
	defer wg.Done()
	for item := range messageChIn {
		vec, err := p.embedder.Embed(ctx, item.Text)
		if err != nil {
			log.Printf("Embedding error: %v", err) // handle error
			//fmt.Println("Embedding error: ", err)
			item.Err = err
		} else {
			//fmt.Println("COrrect Embed: ", vec)
			item.Vector = vec
		}

		embeddingChOut <- item
	}
}

func (p *Pipeline) saveMsgs(ctx context.Context, embeddingCh chan *types.Entity, wg *sync.WaitGroup) error {
	defer wg.Done()
	defer close(embeddingCh)

	var batchItems []*types.Entity
	var batchOffsets []*types.Entity // для commit

	for res := range embeddingCh {
		batchItems = append(batchItems, res)

		if len(batchItems) >= p.cfg.Pipeline.StorageBatchSize {
			if err := p.storage.StoreBatch(ctx, batchItems); err != nil {
				log.Printf("VectorDB error: %v", err)
			} else {
				sourceEntities := make([]*types.Entity, len(batchItems))
				for _, item := range batchItems {
					sourceEntities = append(sourceEntities, item)
				}
				err = p.source.AfterProcessHook(ctx, sourceEntities) // TODO handle error items
				if err != nil {
					return err //TODO handle error
				}
			}
			batchItems = batchItems[:0]
			batchOffsets = batchOffsets[:0]
		}
	}

	// all others to commit
	if len(batchItems) > 0 { // TODO REMOVE DUPLICATE CODE
		if err := p.storage.StoreBatch(ctx, batchItems); err != nil {
			log.Printf("VectorDB error: %v", err)
		} else {
			sourceEntities := make([]*types.Entity, len(batchItems))
			for _, item := range batchItems {
				sourceEntities = append(sourceEntities, item)
			}
			err = p.source.AfterProcessHook(ctx, sourceEntities) // TODO handle error items
			if err != nil {
				return err
			}
		}
	}
	return nil
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
