package pirate

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/op/go-logging"
	"io"
	"sync"
)

type DecompressFunc func(b []byte) ([]byte, error)

type compressionWorker struct {
	decompress DecompressFunc
	logger     *logging.Logger
	chIn       <-chan []byte
	chOut      chan<- []byte
}

func NewPlainDecompressor() DecompressFunc {
	return func(b []byte) ([]byte, error) {
		return b, nil
	}
}

func NewGzipDecompressor() DecompressFunc {
	return func(b []byte) ([]byte, error) {
		reader, err := gzip.NewReader(bytes.NewBuffer(b))
		if err != nil {
			return nil, fmt.Errorf("Failed to initialize gzip reader: %s", err)
		}
		defer reader.Close()

		return io.ReadAll(reader)
	}
}

func NewCompressionWorker(decomp DecompressFunc, logger *logging.Logger, chIn <-chan []byte, chOut chan<- []byte) *compressionWorker {
	return &compressionWorker{decomp, logger, chIn, chOut}
}

func (w *compressionWorker) Run(concurrency int) {
	wg := &sync.WaitGroup{}

	w.logger.Infof("[Decompressor] Starting %d decompression workers", concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go w.run(wg)
	}

	wg.Wait()
}

func (w *compressionWorker) run(wg *sync.WaitGroup) {
	for in := range w.chIn {
		out, err := w.decompress(in)
		if err != nil {
			w.logger.Warningf("[Decompressor] Failed to decompress: %s", err)
			continue
		}

		if w.logger.IsEnabledFor(logging.DEBUG) {
			w.logger.Debugf("[Decompressor] Decompressed %d bytes to %d bytes", len(in), len(out))
			for _, row := range bytes.Split(bytes.TrimSpace(out), []byte("\n")) {
				w.logger.Debugf("[Decompressor] > %s", row)
			}
		}

		w.chOut <- out
	}

	wg.Done()
}
