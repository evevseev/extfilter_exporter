package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var (
		listenAddress      = flag.String("web.listen-address", ":9513", "Address on which to expose metrics and web interface.")
		metricsPath        = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		extfilterStatsPath = flag.String("extfilter.stats-path", "", "ExtFilter stats file")
	)
	flag.Parse()

	if *extfilterStatsPath == "" {
		log.Fatal("Extfilter stats file is not provided")
		os.Exit(1)
	}

	reg := prometheus.NewRegistry()
	col := newExtfilterCollector(*extfilterStatsPath)
	reg.MustRegister(col)

	mux := http.NewServeMux()
	mux.Handle(*metricsPath, promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	log.Printf("Starting extfilter exporter on %v/metrics\n", *listenAddress)
	if err := http.ListenAndServe(*listenAddress, mux); err != nil {
		log.Fatalf("Unable to start extfilter exporter: %v", err)
		os.Exit(1)
	}
}
