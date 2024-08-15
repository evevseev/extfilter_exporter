package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

const metricsPrefix = "extfilter_"

type extfilterCollector struct {
	statsPath          string
	workerPackets      *prometheus.Desc
	workerIpPackets    *prometheus.Desc
	workerBytes        *prometheus.Desc
	workerMatched      *prometheus.Desc
	workerFragments    *prometheus.Desc
	workerShortPackets *prometheus.Desc
	receivedPackets    *prometheus.Desc
	missedPackets      *prometheus.Desc
	inputErrors        *prometheus.Desc
	noBuffer           *prometheus.Desc
}

func newExtfilterCollector(statsPath string) *extfilterCollector {
	return &extfilterCollector{
		statsPath: statsPath,
		workerPackets: prometheus.NewDesc(metricsPrefix+"worker_packets_total",
			"Total packets processed by the worker",
			[]string{"core"}, nil,
		),
		workerIpPackets: prometheus.NewDesc(metricsPrefix+"ip_packets_total",
			"Total IP packets processed by the worker",
			[]string{"core", "ip_version"}, nil,
		),
		workerBytes: prometheus.NewDesc(metricsPrefix+"bytes_total",
			"Total bytes processed by the worker",
			[]string{"core"}, nil,
		),
		workerMatched: prometheus.NewDesc(metricsPrefix+"matches_total",
			"Total matches by the worker",
			[]string{"core", "type"}, nil,
		),
		workerFragments: prometheus.NewDesc(metricsPrefix+"fragments_total",
			"Total fragments recieved by the worker",
			[]string{"core", "ip_version"}, nil,
		),
		workerShortPackets: prometheus.NewDesc(metricsPrefix+"short_packets",
			"Total short packets processed by the worker",
			[]string{"core", "ip_version"}, nil,
		),
		receivedPackets: prometheus.NewDesc(metricsPrefix+"packets_received_total",
			"Total packets received on all ports",
			nil, nil,
		),
		missedPackets: prometheus.NewDesc(metricsPrefix+"packets_missed_total",
			"Total packets missed on all ports",
			nil, nil,
		),
		inputErrors: prometheus.NewDesc(metricsPrefix+"input_errors_total",
			"Total input errors encountered on all ports",
			nil, nil,
		),
		noBuffer: prometheus.NewDesc(metricsPrefix+"rx_no_buffer_total",
			"Total RX buffer errors encountered on all ports",
			nil, nil,
		),
	}
}

func (collector *extfilterCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.workerPackets
	ch <- collector.workerIpPackets
	ch <- collector.workerBytes
	ch <- collector.workerMatched
	ch <- collector.workerFragments
	ch <- collector.workerShortPackets
	ch <- collector.receivedPackets
	ch <- collector.missedPackets
	ch <- collector.inputErrors
	ch <- collector.noBuffer
}

func (collector *extfilterCollector) Collect(ch chan<- prometheus.Metric) {
	file, err := os.Open(collector.statsPath)
	if err != nil {
		log.Println("Error opening stats file:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		metric, valueStr := parseMetric(line)
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			log.Println("Error parsing metric value:", err)
			continue
		}

		collector.processMetric(ch, metric, value)
	}

	if err := scanner.Err(); err != nil {
		log.Println("Error reading stats file:", err)
	}
}

func parseMetric(line string) (string, string) {
	parts := strings.Split(line, "=")
	return parts[0], parts[1]
}

func (collector *extfilterCollector) processMetric(ch chan<- prometheus.Metric, metric string, value float64) {
	metricParts := strings.Split(metric, ".")

	switch metricParts[0] {
	case "worker":
		core := metricParts[2]
		name := metricParts[3]
		switch name {
		case "total_packets":
			ch <- prometheus.MustNewConstMetric(collector.workerPackets, prometheus.CounterValue, value, core)
		case "ip_packets", "ipv4_packets", "ipv6_packets":
			var ipVersion string
			if name == "ipv4_packets" {
				ipVersion = "4"
			} else if name == "ipv6_packets" {
				ipVersion = "6"
			} else if name == "ip_packets" {
				return
			}
			ch <- prometheus.MustNewConstMetric(collector.workerIpPackets, prometheus.CounterValue, value, core, ipVersion)
		case "total_bytes":
			ch <- prometheus.MustNewConstMetric(collector.workerBytes, prometheus.CounterValue, value, core)
		case "matched_ip_port", "matched_ssl_sni", "matched_ssl_ip", "matched_http_bl_ipv4", "matched_http_bl_ipv6":
			mType := strings.TrimPrefix(name, "matched_")
			ch <- prometheus.MustNewConstMetric(collector.workerMatched, prometheus.CounterValue, value, core, mType)
		case "ipv4_fragments", "ipv6_fragments":
			var ipVersion string
			if name == "ipv4_fragments" {
				ipVersion = "4"
			} else if name == "ipv6_fragments" {
				ipVersion = "6"
			}
			ch <- prometheus.MustNewConstMetric(collector.workerFragments, prometheus.CounterValue, value, core, ipVersion)
		case "ipv4_short_packets":
			ch <- prometheus.MustNewConstMetric(collector.workerShortPackets, prometheus.CounterValue, value, core, "4")
		default:
			log.Println("Unknown worker metric name:", name)
		}
	case "allports":
		switch metricParts[1] {
		case "received_packets":
			ch <- prometheus.MustNewConstMetric(collector.receivedPackets, prometheus.CounterValue, value)
		case "missed_packets":
			ch <- prometheus.MustNewConstMetric(collector.missedPackets, prometheus.CounterValue, value)
		case "ierrors":
			ch <- prometheus.MustNewConstMetric(collector.inputErrors, prometheus.CounterValue, value)
		case "rx_nombuf":
			ch <- prometheus.MustNewConstMetric(collector.noBuffer, prometheus.CounterValue, value)
		default:
			log.Println("Unknown allports metric name:", metricParts[1])
		}
	case "allworkers":
	default:
		log.Println("Unknown metric type:", metricParts[0])
	}
}
