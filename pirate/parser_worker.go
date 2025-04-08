package pirate

import (
	"github.com/op/go-logging"
	"sync"
)

type ParserWorker struct {
	logger *logging.Logger
	chUdp  <-chan []byte
	chMsg  chan<- *Message
}

func NewParserWorker(logger *logging.Logger, chUdp <-chan []byte, chMsg chan<- *Message) *ParserWorker {
	return &ParserWorker{logger, chUdp, chMsg}
}

func (w *ParserWorker) Run(concurrency int) {
	wg := &sync.WaitGroup{}

	w.logger.Infof("[Parser] Starting %d parser workers", concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go w.run(wg)
	}

	wg.Wait()
}

func (w *ParserWorker) run(wg *sync.WaitGroup) {
	for udp := range w.chUdp {
		msg := &Message{}

		if err := DecodeMessage(udp, msg); err != nil {
			w.logger.Warningf("[Parser] Error: %s", err)
			continue
		}

		w.logger.Debugf("[Parser] Parsed %d bytes to %d headers and %d metrics", len(udp), len(msg.Header), len(msg.Metrics))
		w.chMsg <- msg
	}

	wg.Done()
}
