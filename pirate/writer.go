package pirate

import (
	"bytes"
	"fmt"
	"github.com/op/go-logging"
	"io"
	"net"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func NewWriter(target string, logger *logging.Logger) (MetricWriter, error) {
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Graphite writer: %s", err)
	}

	switch parsed.Scheme {
	case "file":
		return NewFileWriter(parsed.Path, logger)
	case "tcp":
		return NewTcpWriter(parsed.Host, logger)
	default:
		return nil, fmt.Errorf(`Unsupported graphite target (scheme must be "tcp" or "file"): %s`, parsed.Scheme)
	}
}

func NewFileWriter(filename string, logger *logging.Logger) (*wrappedWriter, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("Failed to open graphite target file %s: %s", filename, err)
	}

	reopen := func() error {
		file.Close()

		if file, err = os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644); err != nil {
			// Huston, we've got a problem!
			return fmt.Errorf("Failed to reopen file %s: %s\n", filename, err)
		}

		return nil
	}

	// reopen file handle on USR1 signal
	go func() {
		chSig := make(chan os.Signal, 1)
		signal.Notify(chSig, syscall.SIGUSR1)

		for {
			<-chSig
			logger.Debugf("[SignalHandler] Reopening grafsy file %s", filename)
			if err := reopen(); err != nil {
				logger.Errorf("[SignalHandler] %s", err)
			}
		}
	}()

	return &wrappedWriter{writer: file, reopen: reopen, logger: logger}, nil
}

func NewTcpWriter(addr string, logger *logging.Logger) (*wrappedWriter, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("TCP error: %s", err)
	}

	writer := &wrappedWriter{writer: conn, logger: logger}
	writer.reopen = func() error {
		conn.Close()

		var newWriter net.Conn

		for {
			if newWriter, err = net.Dial("tcp", addr); err == nil {
				break
			}

			logger.Debug("[TCP Writer] Failed to reconnect, trying again in 500 ms")
			time.Sleep(500 * time.Millisecond)
		}

		writer.writer = newWriter
		logger.Info("[TCP Writer] Reconnected successfully")

		return nil
	}

	return writer, nil
}

type MetricWriter interface {
	Write(m *Metric) error
	WriteRaw(path []byte, value []byte, timestamp []byte) error
}

type wrappedWriter struct {
	writer io.Writer
	reopen func() error
	logger *logging.Logger
	mu     sync.Mutex
}

func (w *wrappedWriter) Write(m *Metric) error {
	return w.WriteRaw(m.Name, m.Value, m.Timestamp)
}

func (w *wrappedWriter) WriteRaw(path []byte, value []byte, timestamp []byte) error {
	buf := bytes.Join([][]byte{path, []byte(" "), value, []byte(" "), timestamp, []byte("\n")}, []byte{})

	w.logger.Debugf("[Writer] Writing: %s", buf)

	if _, err := w.writer.Write(buf); err != nil {
		w.mu.Lock()
		defer w.mu.Unlock()

		// try to write again within the lock (it might be, that another goroutine already reconnected successfully)
		if _, err := w.writer.Write(buf); err != nil {
			w.logger.Warningf("[Writer] Failed to write metric, trying to reopen")
			if err = w.reopen(); err != nil {
				return err
			}

			if _, err := w.writer.Write(buf); err != nil {
				return fmt.Errorf("[Metric Writer] Failed to write metric: %s", err)
			}
		}
	}

	return nil
}
