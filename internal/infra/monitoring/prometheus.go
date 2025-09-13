package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"strconv"
)

type PrometheusConfig struct {
	Active bool
	Port   int
}

func RunPrometheus(pCfg PrometheusConfig) {
	if !pCfg.Active || pCfg.Port == 0 {
		return
	}
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	go func() {
		http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
		log.Printf("Prometheus metrics available at http://localhost:%d/metrics\n", pCfg.Port)
		log.Fatal(http.ListenAndServe(":"+strconv.Itoa(pCfg.Port), nil))
	}()
}
