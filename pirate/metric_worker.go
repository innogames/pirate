package pirate

import (
	"github.com/op/go-logging"
	"sync"
)

type metricWorker struct {
	cfg      *Config
	logger   *logging.Logger
	chMsg    <-chan *Message
	chMetric chan<- *Metric
}

func NewMetricWorker(cfg *Config, logger *logging.Logger, chMsg <-chan *Message, chMetric chan<- *Metric) *metricWorker {
	return &metricWorker{cfg, logger, chMsg, chMetric}
}

func (w *metricWorker) Run(concurrency int) {
	var wg sync.WaitGroup

	w.logger.Infof("[MetricResolver] Starting %d metric workers", concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go w.run(wg)
	}

	wg.Wait()
}

func (w *metricWorker) run(wg sync.WaitGroup) {
	var projectCfg *ProjectConfig
	var metricCfg *MetricConfig

	for msg := range w.chMsg {
		projectCfg = w.cfg.Projects[string(msg.Header["project"])]

		for _, metric := range msg.Metrics {
			metricCfg = projectCfg.Metrics[string(metric.Name)]

			path, err := metricCfg.GraphiteTemplate.Resolve(NewCtx(msg.Header, metric))
			if err != nil {
				w.logger.Errorf("[MetricResolver] %s", err)
				continue
			}

			w.logger.Debugf("[MetricResolver] Resolved path of %s.%s to %s", msg.Header["project"], metric.Name, path)
			w.chMetric <- &Metric{path, metric.Value, metric.Timestamp}
		}
	}

	wg.Done()
}
