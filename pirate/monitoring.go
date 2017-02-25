package pirate

import (
	"github.com/op/go-logging"
	"sync"
	"time"
)

type MonitoringStats struct {
	stats map[string]int
	mu    sync.Mutex
}

func NewMonitoringStats() *MonitoringStats {
	return &MonitoringStats{stats: make(map[string]int)}
}

func (s *MonitoringStats) IncBytesIn(delta int) {
	s.add("bytes_in", delta)
}

func (s *MonitoringStats) IncBytesOut(delta int) {
	s.add("bytes_out", delta)
}

func (s *MonitoringStats) IncUdpReceived() {
	s.add("udp_received", 1)
}

func (s *MonitoringStats) IncUdpDropped() {
	s.add("udp_dropped", 1)
}

func (s *MonitoringStats) IncMsgReceived() {
	s.add("messages_received", 1)
}

func (s *MonitoringStats) IncMsgDropped() {
	s.add("messages_dropped", 1)
}

func (s *MonitoringStats) IncMetricsReceived(delta int) {
	s.add("metrics_received", delta)
}

func (s *MonitoringStats) IncMetricsDropped(delta int) {
	s.add("metrics_dropped", delta)
}

func (s *MonitoringStats) IncMetricsWritten() {
	s.add("metrics_written", 1)
}

func (s *MonitoringStats) add(key string, delta int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats[key] = s.stats[key] + delta
}

func (s *MonitoringStats) Reset() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldStats := s.stats
	s.stats = make(map[string]int)

	return oldStats
}

type MonitoringWorker struct {
	cfg      *Config
	logger   *logging.Logger
	chMetric chan<- *Metric
	stats    *MonitoringStats
}

func NewMonitoringWorker(cfg *Config, logger *logging.Logger, chMetric chan<- *Metric, stats *MonitoringStats) *MonitoringWorker {
	return &MonitoringWorker{cfg, logger, chMetric, stats}
}

func (w *MonitoringWorker) Run() {
	w.logger.Info("[Monitoring] Starting monitoring worker")

	for {
		time.Sleep(1 * time.Minute)

		now := time.Now()

		for key, value := range w.stats.Reset() {
			w.logger.Infof("[Monitoring] %s = %d", key, value)

			if w.cfg.MonitoringEnabled {
				rawMetric := NewMetric(key, float32(value), now)
				path, err := w.cfg.MonitoringTemplate.Resolve(NewMonitoringCtx(rawMetric))
				if err != nil {
					w.logger.Errorf("[Monitoring] Failed to resolve path: %s", err)
					continue
				}

				select {
				case w.chMetric <- NewMetric(string(path), float32(value), now):
				default:
					w.logger.Noticef("[Monitoring] Write buffer is full, failed to send monitoring metric %s", key)
				}
			}
		}
	}
}
