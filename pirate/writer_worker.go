package pirate

import (
	"github.com/op/go-logging"
	"sync"
)

type writerWorker struct {
	writer   MetricWriter
	logger   *logging.Logger
	chMetric chan *Metric
}

func NewWriterWorker(writer MetricWriter, logger *logging.Logger, chMetric chan *Metric) *writerWorker {
	worker := new(writerWorker)
	worker.writer = writer
	worker.logger = logger
	worker.chMetric = chMetric

	return worker
}

func (w *writerWorker) Run(concurrency int) {
	var wg sync.WaitGroup

	w.logger.Infof("[Writer] Starting %d writer workers", concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go w.run(wg)
	}

	wg.Wait()
}

func (w *writerWorker) run(wg sync.WaitGroup) {
	for metric := range w.chMetric {
		if err := w.writer.Write(metric); err != nil {
			w.logger.Debugf("[Writer] Re-scheduling metric")
			w.chMetric <- metric
		}
	}

	wg.Done()
}
