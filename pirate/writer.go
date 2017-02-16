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
	"syscall"
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

	return &wrappedWriter{file, reopen, logger}, nil
}

func NewTcpWriter(addr string, logger *logging.Logger) (*wrappedWriter, error) {

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("TCP error: %s", err)
	}

	reopen := func() error {
		conn.Close()

		if conn, err = net.Dial("tcp", addr); err != nil {
			return fmt.Errorf("TCP error: %s", err)
		}

		return nil
	}

	return &wrappedWriter{conn, reopen, logger}, nil
}

type MetricWriter interface {
	Write(m *Metric) error
	WriteRaw(path []byte, value []byte, timestamp []byte) error
}

type wrappedWriter struct {
	writer io.Writer
	reopen func() error
	logger *logging.Logger
}

func (w *wrappedWriter) Write(m *Metric) error {
	return w.WriteRaw(m.Name, m.Value, m.Timestamp)
}

func (w *wrappedWriter) WriteRaw(path []byte, value []byte, timestamp []byte) error {
	buf := bytes.Join([][]byte{path, []byte(" "), value, []byte(" "), timestamp, []byte("\n")}, []byte{})

	w.logger.Debugf("[Writer] Writing: %s", buf)
	if _, err := w.writer.Write(buf); err != nil {
		w.logger.Warningf("[Writer] Failed to write metric, trying to reopen")
		if err = w.reopen(); err != nil {
			return err
		}
	}

	if _, err := w.writer.Write(buf); err != nil {
		return fmt.Errorf("[Metric Writer] Failed to write metric: %s", err)
	}

	return nil
}
