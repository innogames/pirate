package pirate

import (
	"errors"
	"fmt"
	"github.com/op/go-logging"
	"strconv"
	"sync"
	"time"
)

type validatorWorker struct {
	cfg    *Config
	logger *logging.Logger
	stats  *MonitoringStats
	chIn   <-chan *Message
	chOut  chan<- *Message
}

func NewValidatorWorker(cfg *Config, logger *logging.Logger, stats *MonitoringStats, chIn <-chan *Message, chOut chan<- *Message) *validatorWorker {
	return &validatorWorker{cfg, logger, stats, chIn, chOut}
}

func (w *validatorWorker) Run(concurrency int) {
	wg := &sync.WaitGroup{}

	w.logger.Infof("[Validator] Starting %d validation workers", concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go w.run(wg)
	}

	wg.Wait()
}

func (w *validatorWorker) run(wg *sync.WaitGroup) {
	for msg := range w.chIn {
		metricsBefore := len(msg.Metrics)

		w.stats.IncMsgReceived()
		w.stats.IncMetricsReceived(metricsBefore)

		if err := w.validateMsg(msg); err != nil {
			w.logger.Noticef("[Validator] Validation failed: %s", err)
			w.stats.IncMsgDropped()
			w.stats.IncMetricsDropped(metricsBefore)

			continue
		}

		w.logger.Debugf("[Validator] Validation succeeded with %d of %d metrics", len(msg.Metrics), metricsBefore)
		w.stats.IncMetricsDropped(metricsBefore - len(msg.Metrics))

		w.chOut <- msg
	}

	wg.Done()
}

func (w *validatorWorker) validateMsg(msg *Message) error {
	// check, if project attribute is set
	pid, exists := msg.Header["project"]
	if !exists {
		return errors.New("Missing project attribute")
	}

	// check, if target project is configured
	projectCfg, exists := w.cfg.Projects[string(pid)]
	if !exists {
		return fmt.Errorf(`Unknown project ID "%s"`, pid)
	}

	// validate headers against regex
	for key, value := range msg.Header {
		// project is already valid by its existence in config
		if key == "project" {
			continue
		}

		attrRegexp, exists := projectCfg.AttributesRegex[key]
		if !exists {
			return fmt.Errorf(`Unknown attribute "%s" in project "%s"`, key, pid)
		}

		if !attrRegexp.Match(value) {
			return fmt.Errorf(`Attribute value "%s" does not match regexp for %s.%s`, value, pid, key)
		}
	}

	if len(msg.Metrics) == 0 {
		return errors.New("Missing metrics")
	}

	// validate metrics
	validIdx := 0
	for _, metric := range msg.Metrics {
		if err := w.validateMetric(projectCfg, metric); err != nil {
			w.logger.Infof("[Validator] Validation failed for %s.%s: %s", pid, metric.Name, err)
			continue
		}

		// keep valid element
		msg.Metrics[validIdx] = metric
		validIdx++
	}
	msg.Metrics = msg.Metrics[:validIdx]

	if len(msg.Metrics) == 0 {
		return errors.New("No valid metrics found")
	}

	return nil
}

func (w *validatorWorker) validateMetric(cfg *ProjectConfig, metric *Metric) error {
	// check, if metrics key is configured
	key := string(metric.Name)
	metricCfg, exists := cfg.Metrics[key]
	if !exists {
		return fmt.Errorf(`unknown metric key "%s"`, key)
	}

	// validate timestamp
	ts, err := strconv.ParseInt(string(metric.Timestamp), 10, 64)
	if err != nil {
		return errors.New("timestamp must be int64-compatible")
	}

	metricTime := time.Unix(ts, 0)
	if time.Now().Add(10 * time.Second).Truncate(time.Second).Before(metricTime) { // TODO: make max future time configurable
		return fmt.Errorf("future timestamp (%s ahead)", time.Until(metricTime))
	}

	if time.Now().Add(-3 * time.Hour).Truncate(time.Second).After(metricTime) { // TODO: make max age configurable
		return fmt.Errorf("timestamp too old (%s behind)", time.Until(metricTime.Truncate(time.Second)))
	}

	// validate value
	value, err := strconv.ParseFloat(string(metric.Value), 64)
	if err != nil {
		return errors.New("value must be float64-compatible")
	}

	if value < metricCfg.Min {
		return errors.New("value lower than configured minimum")
	}

	if value > metricCfg.Max {
		return errors.New("value higher than configured maximum")
	}

	return nil
}
