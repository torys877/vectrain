package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/trace"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/torys877/vectrain/internal/app/embedders/ollama"
	"github.com/torys877/vectrain/internal/app/pipeline"
	"github.com/torys877/vectrain/internal/app/sources/kafka"
	"github.com/torys877/vectrain/internal/app/storages/qdrant"
	"github.com/torys877/vectrain/internal/config"
	routes "github.com/torys877/vectrain/internal/http"
)

func main() {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	go func() {
		http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
		log.Println("Prometheus metrics available at http://localhost:9100/metrics")
		log.Fatal(http.ListenAndServe(":9100", nil))
	}()

	// ------------------ pprof HTTP ------------------
	go func() {
		log.Println("pprof available at http://localhost:6060/debug/pprof/")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	// ------------------ CPU Profiling ------------------
	//cpuFile, err := os.Create("cpu.prof")
	//if err != nil {
	//	log.Fatalf("could not create CPU profile: %v", err)
	//}
	//defer cpuFile.Close()
	//if err := pprof.StartCPUProfile(cpuFile); err != nil {
	//	log.Fatalf("could not start CPU profile: %v", err)
	//}
	//defer pprof.StopCPUProfile()

	// ------------------ Heap Profiling ------------------
	//heapFile, err := os.Create("heap.prof")
	//if err != nil {
	//	log.Fatalf("could not create heap profile: %v", err)
	//}
	//defer heapFile.Close()
	//
	//// Set up periodic heap profiling
	//go func() {
	//	i := 0
	//	for range time.Tick(10 * time.Second) {
	//		//f, err := os.Create("heap_periodic_" + time.Now().Format("20060102_150405") + strconv.Itoa(i) + ".prof")
	//		f, err := os.Create("heap_periodic_" + strconv.Itoa(i) + ".prof")
	//		if err != nil {
	//			log.Printf("could not create periodic heap profile: %v", err)
	//			continue
	//		}
	//		pprof.WriteHeapProfile(f)
	//		f.Close()
	//		i += 10
	//	}
	//}()

	// Write final heap profile at program exit
	//defer func() {
	//	if err := pprof.WriteHeapProfile(heapFile); err != nil {
	//		log.Printf("could not write final heap profile: %v", err)
	//	}
	//}()

	// ------------------ Runtime Trace ------------------
	traceFile, err := os.Create("trace.out")
	if err != nil {
		log.Fatalf("could not create trace file: %v", err)
	}
	defer traceFile.Close()
	if err := trace.Start(traceFile); err != nil {
		log.Fatalf("could not start trace: %v", err)
	}
	defer trace.Stop()

	// ------------------ Memory Monitoring ------------------
	go func() {
		var m runtime.MemStats
		start := time.Now()
		for range time.Tick(500 * time.Millisecond) {
			runtime.ReadMemStats(&m)
			fmt.Printf("Elapsed: %v, Alloc: %v KB, Sys: %v KB, NumGC: %v\n",
				time.Since(start).Truncate(time.Millisecond),
				m.Alloc/1024, m.Sys/1024, m.NumGC)
		}
	}()

	fmt.Println(" === Vectrain === ")
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()
	fmt.Println("configPath:", *configPath)
	if *configPath == "" {
		fmt.Println("Error: --config argument is required")
		os.Exit(1)
	}

	appConfig, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	kafkaSource, err := kafka.NewKafkaClient(appConfig.Source)
	if err != nil {
		fmt.Println("Source Error:", err)
		os.Exit(1)
	}
	//defer kafkaSource.Close()

	qdrantStorage, err := qdrant.NewQdrantClient(appConfig.Storage)
	if err != nil {
		fmt.Println("Storage Error:", err)
		os.Exit(1)
	}
	//defer qdrantStorage.Close()

	ollamaEmbedder, err := ollama.NewOllamaClient(appConfig.Embedder)
	if err != nil {
		fmt.Println("Embedder Error:", err)
		os.Exit(1)
	}

	appPipeline := pipeline.NewPipeline(appConfig, kafkaSource, ollamaEmbedder, qdrantStorage)

	e := echo.New()
	err = routes.SetupRoutes(e, appConfig, appPipeline)
	if err != nil {
		fmt.Printf("Routes not set, check configuration: %v\n", err)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Start server
	srvErrCh := make(chan error, 1)
	go func() {
		if err := e.Start(":" + strconv.Itoa(appConfig.App.Http.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErrCh <- err
		}
		close(srvErrCh)
	}()

	// Start pipeline
	pipelineErrCh := make(chan error, 1)
	go func() {
		if err := appPipeline.Run(ctx); err != nil {
			pipelineErrCh <- err
		}
		close(pipelineErrCh)
	}()

	select {
	case <-ctx.Done():
		fmt.Println("Signal received, shutting down...")
	case err := <-srvErrCh:
		fmt.Println("Server error:", err)
	case err := <-pipelineErrCh:
		fmt.Println("Pipeline error:", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Fatal(err)
	}

	appPipeline.Stop()

	fmt.Println("CPU profile, heap profile, and trace collected. Exiting.")
}
