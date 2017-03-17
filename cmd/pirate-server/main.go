package main

import (
	"flag"
	"fmt"
	"github.com/innogames/pirate/pirate"
	"github.com/op/go-logging"
	"os"
	"runtime"
)

func main() {
	configFile := flag.String("config", "/etc/pirate/config.yml", "Path to config file")
	flag.Parse()

	cfg, err := pirate.LoadConfig(*configFile)
	if err != nil {
		fail("Failed to load configuration: %s\n", err)
	}

	logger := createLogger(cfg)
	cfg.Log(logger)

	chUdp := make(chan []byte, 100)
	chUdpDecomp := make(chan []byte, 100)
	chMsg := make(chan *pirate.Message, 100)
	chValidMsg := make(chan *pirate.Message, 100)
	chMetric := make(chan *pirate.Metric, 1000)

	stats := pirate.NewMonitoringStats()

	server, err := pirate.NewUdpServer(cfg.UdpAddress, cfg.PerIpRateLimit, logger, stats, chUdp)
	if err != nil {
		fail("Failed to initialize server: %s\n", err)
	}

	writer, err := pirate.NewWriter(cfg.GraphiteTarget, logger, stats)
	if err != nil {
		fail("Failed to initialize writer: %s", err)
	}

	decompressor := pirate.NewPlainDecompressor()
	if cfg.Gzip {
		decompressor = pirate.NewGzipDecompressor()
	}

	numCpus := runtime.NumCPU()

	go pirate.NewCompressionWorker(decompressor, logger, chUdp, chUdpDecomp).Run(numCpus)
	go pirate.NewParserWorker(logger, chUdpDecomp, chMsg).Run(numCpus)
	go pirate.NewValidatorWorker(cfg, logger, stats, chMsg, chValidMsg).Run(numCpus)
	go pirate.NewMetricWorker(cfg, logger, chValidMsg, chMetric).Run(numCpus)
	go pirate.NewWriterWorker(writer, logger, chMetric).Run(1)
	go pirate.NewMonitoringWorker(cfg, logger, chMetric, stats).Run()

	if err := server.Run(); err != nil {
		fail("UDP Server error: %s", err)
	}
}

func createLogger(cfg *pirate.Config) *logging.Logger {
	format := logging.MustStringFormatter(`%{time:2006-01-02 15:04:05.000} %{level:.4s} %{message}`)
	logger := logging.MustGetLogger("pirate")
	backend := logging.NewBackendFormatter(logging.NewLogBackend(os.Stdout, "", 0), format)
	leveledBackend := logging.AddModuleLevel(backend)
	leveledBackend.SetLevel(cfg.LogLevel, "pirate")
	logger.SetBackend(leveledBackend)

	return logger
}

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
