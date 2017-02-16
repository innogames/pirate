package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"
)

func main() {
	addr := flag.String("addr", ":33333", "Server address to send UDP packages to")
	attr := flag.String("attr", "", "Attributes")
	name := flag.String("name", "", "Metric name")
	min := flag.Float64("min", 0, "Minimum for random value")
	max := flag.Float64("max", 0, "Maximum for random value")
	frequency := flag.Duration("freq", 500*time.Millisecond, "Frequency to generate metrics")
	compression := flag.Bool("gzip", false, "Use gzip compression")
	flag.Parse()

	type metric struct {
		Name      string
		Timestamp int64
		Value     float64
	}

	udpAddr, err := net.ResolveUDPAddr("udp", *addr)
	if err != nil {
		fail("Failed to resolve UDP address: %s", err)
	}

	// establish UDP connection
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		fail("Failed to dial UDP: %s", err)
	}
	defer conn.Close()

	for {
		startTime := time.Now()

		// generate metrics
		amount := int(time.Second / *frequency)
		metrics := make([]*metric, amount)
		for i := 0; i < amount; i++ {
			metrics[i] = &metric{*name, time.Now().Unix(), rand.Float64()*(*max-*min) + *min}
		}

		// write metrics to buffer
		buf := bytes.NewBuffer(make([]byte, 0, 1024*1024*1024))
		buf.Write([]byte(*attr))
		buf.WriteByte('\n')
		for _, m := range metrics {
			buf.WriteString(fmt.Sprintf("%s %f %d\n", m.Name, m.Value, m.Timestamp))
		}

		// compress buffer, if gzip enabled
		if *compression {
			gzBuf := bytes.NewBuffer(make([]byte, 0, 64*1024*1024))
			gzWriter := gzip.NewWriter(gzBuf)
			buf.WriteTo(gzWriter)
			gzWriter.Flush()
			gzWriter.Close()

			buf = gzBuf
		}

		log.Printf("Sending %d metrics\n", amount)

		// write buffer to UDP connection
		if _, err := buf.WriteTo(conn); err != nil {
			fail("Failed to send UDP package: %s", err)
		}

		time.Sleep(1*time.Second - (time.Now().Sub(startTime)))
	}
}

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
