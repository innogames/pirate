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
	chUdp   chan<- []byte
}

func NewUdpServer(address string, logger *logging.Logger, chUdp chan<- []byte) (*UdpServer, error) {
	parsedAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, fmt.Errorf("Unable to resolve UDP address %s: %s", address, err)
	}

	return &UdpServer{parsedAddr, logger, chUdp}, nil
}

func (s *UdpServer) Run() error {
	conn, err := net.ListenUDP("udp", s.address)
	if err != nil {
		return fmt.Errorf("Unable to start UDP server on %s: %s", *s.address, err)
	}
	defer conn.Close()

	buf := make([]byte, UdpBufferSize)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			s.logger.Infof("[UDP] Failed to read packet: %s", err)
			continue
		}

		s.logger.Debugf("[UDP] Received %d bytes", n)

		packet := make([]byte, n)
		copy(packet, buf)

		select {
		case s.chUdp <- packet:
		default:
			s.logger.Debug("[UDP] Buffer is full, packet got dropped")
		}
	}
}
