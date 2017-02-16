package pirate

import (
	"bytes"
	"errors"
)

var (
	// errors
	EndOfHeader      = errors.New("End of header line was reached")
	MissingEndOfLine = errors.New("End of line missing")
	InvalidKey       = errors.New("Invalid key")
	InvalidValue     = errors.New("Invalid value")
	IncompletePair   = errors.New("Incomplete pair")
	EndOfMetrics     = errors.New("End of metrics block was reached")

	// basic char sets
	num        = []byte("0123456789")
	alphaLower = []byte("abcdefghijklmnopqrstuvwxyz")
	alphaUpper = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	alpha      = append(alphaLower, alphaUpper...)
	alphaNum   = append(alpha, num...)

	// specific char sets
	headerKeyChars   = append(alphaLower, '_')
	headerValueChars = append(alphaNum, []byte("+/-_.")...)
	metricKeyChars   = append(alphaNum, '_')
	timestampChars   = num
	metricValueChars = append(num, '.')
)

type Parser struct {
	buf []byte
}

func NewParser(b []byte) *Parser {
	return &Parser{b}
}

func DecodeMessage(b []byte, msg *Message) error {
	if msg == nil {
		return errors.New("Message must not be nil")
	}

	msg.Header = make(map[string][]byte)
	msg.Metrics = make([]*Metric, 0, 10)

	p := NewParser(b)
	for {
		key, value, err := p.ReadHeader()
		if err == EndOfHeader {
			break
		}

		if err != nil {
			return err
		}

		msg.Header[string(key)] = value
	}

	for {
		key, ts, value, err := p.ReadMetric()
		if err == EndOfMetrics {
			break
		}

		if err != nil {
			return err
		}

		msg.Metrics = append(msg.Metrics, &Metric{key, value, ts})
	}

	return nil
}

func (p *Parser) ReadHeader() (key []byte, value []byte, err error) {
	var ok bool

	if bytes.IndexByte(p.buf, '\n') == -1 {
		return nil, nil, MissingEndOfLine
	}

	p.skipSpaces()

	if p.skipByte('\n') {
		return nil, nil, EndOfHeader
	}

	if key, ok = p.readAny(headerKeyChars); !ok {
		return nil, nil, InvalidKey
	}

	p.skipSpaces()

	if !p.skipByte('=') {
		return nil, nil, IncompletePair
	}

	p.skipSpaces()

	if value, ok = p.readAny(headerValueChars); !ok {
		return nil, nil, InvalidValue
	}

	p.skipSpaces()

	if p.skipByte(';') {
		p.skipSpaces()
	}

	return
}

func (p *Parser) ReadMetric() (key []byte, ts []byte, value []byte, err error) {
	var ok bool

	p.skipSpaces()

	if len(p.buf) == 0 || p.skipByte('\n') {
		return nil, nil, nil, EndOfMetrics
	}

	// read key
	if key, ok = p.readAny(metricKeyChars); !ok {
		return nil, nil, nil, InvalidKey
	}

	p.skipSpaces()

	// read value
	if value, ok = p.readAny(metricValueChars); !ok {
		return nil, nil, nil, InvalidValue
	}

	p.skipSpaces()

	// read timestamp
	if ts, ok = p.readAny(timestampChars); !ok {
		return nil, nil, nil, InvalidValue
	}

	p.skipSpaces()
	p.skipByte('\n')

	return
}

func (p *Parser) skipSpaces() {
	var i int
	for i = 0; i < len(p.buf); i++ {
		if p.buf[i] != ' ' {
			break
		}
	}

	p.buf = p.buf[i:]
}

func (p *Parser) skipByte(b byte) bool {
	if len(p.buf) > 0 && p.buf[0] == b {
		p.buf = p.buf[1:]

		return true
	}

	return false
}

func (p *Parser) readAny(allowed []byte) ([]byte, bool) {
	if len(p.buf) == 0 {
		return nil, false
	}

	var i int
	for i = 0; i < len(p.buf); i++ {
		if !isAny(p.buf[i], allowed) {
			break
		}
	}

	b := p.buf[0:i]
	p.buf = p.buf[i:]

	return b, i > 0
}

func isAny(b byte, chars []byte) bool {
	for _, c := range chars {
		if b == c {
			return true
		}
	}

	return false
}
