package pirate

import (
	"fmt"
	"github.com/op/go-logging"
	"net"
)

const (
	UdpBufferSize = 64 * 1024
)

type UdpServer struct {
	address *net.UDPAddr
	logger  *logging.Logger
	stats   *MonitoringStats
	limiter *IpLimiter
	chUdp   chan<- []byte
}

func NewUdpServer(address string, ratelimit *RateLimitConfig, logger *logging.Logger, stats *MonitoringStats, chUdp chan<- []byte) (*UdpServer, error) {
	parsedAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, fmt.Errorf("Unable to resolve UDP address %s: %s", address, err)
	}

	limiter := NewIpLimiter(ratelimit.Amount, ratelimit.Interval)

	return &UdpServer{parsedAddr, logger, stats, limiter, chUdp}, nil
}

func (s *UdpServer) Run() error {
	conn, err := net.ListenUDP("udp", s.address)
	if err != nil {
		return fmt.Errorf("Unable to start UDP server on %s: %s", *s.address, err)
	}
	defer conn.Close()

	buf := make([]byte, UdpBufferSize)
	for {
		// accept packet
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			s.logger.Infof("[UDP] Failed to read packet: %s", err)
			continue
		}

		s.logger.Debugf("[UDP] Received %d bytes", n)
		s.stats.IncBytesIn(n)
		s.stats.IncUdpReceived()

		// check rate limit
		if !s.limiter.Allow(addr.IP) {
			s.logger.Infof("[UDP] Rate Limit reached for address: %s", addr.IP.String())
			s.stats.IncUdpDropped()

			continue
		}

		// forward packet
		packet := make([]byte, n)
		copy(packet, buf)

		select {
		case s.chUdp <- packet:
		default:
			s.logger.Debug("[UDP] Buffer is full, packet got dropped")
			s.stats.IncUdpDropped()
		}
	}
}
