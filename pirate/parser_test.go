package pirate

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHeaderNoEndOfLine(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		p := NewParser([]byte{})

		key, value, err := p.ReadHeader()

		assert.Nil(t, key)
		assert.Nil(t, value)
		assert.Equal(t, MissingEndOfLine, err)
	})

	t.Run("absent eol", func(t *testing.T) {
		p := NewParser([]byte("foo"))

		key, value, err := p.ReadHeader()

		assert.Nil(t, key)
		assert.Nil(t, value)
		assert.Equal(t, MissingEndOfLine, err)
	})
}

func TestEndOfHeader(t *testing.T) {
	p := NewParser([]byte("   \n foobar"))

	key, value, err := p.ReadHeader()

	assert.Nil(t, key)
	assert.Nil(t, value)
	assert.Equal(t, EndOfHeader, err)

	assert.Equal(t, []byte(" foobar"), p.buf, "After end of header line break must be skipped")
}

func TestOneHeaderWithSemicolon(t *testing.T) {
	t.Run("without whitespace", func(t *testing.T) {
		p := NewParser([]byte("foo=bar;\n"))

		key, value, err := p.ReadHeader()

		assert.Equal(t, []byte("foo"), key)
		assert.Equal(t, []byte("bar"), value)
		assert.Nil(t, err)

		assert.Equal(t, []byte("\n"), p.buf)
	})

	t.Run("with whitespace", func(t *testing.T) {
		p := NewParser([]byte("  foo =   bar    ;  \n"))

		key, value, err := p.ReadHeader()

		assert.Equal(t, []byte("foo"), key)
		assert.Equal(t, []byte("bar"), value)
		assert.Nil(t, err)

		assert.Equal(t, []byte("\n"), p.buf)
	})
}

func TestOneHeaderWithoutSemicolon(t *testing.T) {
	t.Run("without whitespace", func(t *testing.T) {
		p := NewParser([]byte("foo=bar\n"))

		key, value, err := p.ReadHeader()

		assert.Equal(t, []byte("foo"), key)
		assert.Equal(t, []byte("bar"), value)
		assert.Nil(t, err)

		assert.Equal(t, []byte("\n"), p.buf)
	})

	t.Run("with whitespace", func(t *testing.T) {
		p := NewParser([]byte("   foo  = bar   \n"))

		key, value, err := p.ReadHeader()

		assert.Equal(t, []byte("foo"), key)
		assert.Equal(t, []byte("bar"), value)
		assert.Nil(t, err)

		assert.Equal(t, []byte("\n"), p.buf)
	})
}

func TestMultipleHeaders(t *testing.T) {
	t.Run("without whitespace", func(t *testing.T) {
		p := NewParser([]byte("foo=bar;baz=1337\n"))

		// first header
		key, value, err := p.ReadHeader()

		assert.Equal(t, []byte("foo"), key)
		assert.Equal(t, []byte("bar"), value)
		assert.Nil(t, err)

		// second header
		key, value, err = p.ReadHeader()

		assert.Equal(t, []byte("baz"), key)
		assert.Equal(t, []byte("1337"), value)
		assert.Nil(t, err)

		assert.Equal(t, []byte("\n"), p.buf)
	})

	t.Run("with whitespace", func(t *testing.T) {
		p := NewParser([]byte(" foo     = bar  ;       baz =      1337   ;    \n"))

		// first header
		key, value, err := p.ReadHeader()

		assert.Equal(t, []byte("foo"), key)
		assert.Equal(t, []byte("bar"), value)
		assert.Nil(t, err)

		assert.Equal(t, []byte("baz =      1337   ;    \n"), p.buf)

		// second header
		key, value, err = p.ReadHeader()

		assert.Equal(t, []byte("baz"), key)
		assert.Equal(t, []byte("1337"), value)
		assert.Nil(t, err)

		assert.Equal(t, []byte("\n"), p.buf)
	})
}

func TestInvalidHeaders(t *testing.T) {
	t.Run("invalid key", func(t *testing.T) {
		p := NewParser([]byte(" 1337   \n"))

		key, value, err := p.ReadHeader()

		assert.Nil(t, key)
		assert.Nil(t, value)
		assert.Equal(t, InvalidKey, err)
	})

	t.Run("missing equals", func(t *testing.T) {
		p := NewParser([]byte(" foo bar   \n"))

		key, value, err := p.ReadHeader()

		assert.Nil(t, key)
		assert.Nil(t, value)
		assert.Equal(t, IncompletePair, err)
	})

	t.Run("missing value", func(t *testing.T) {
		p := NewParser([]byte(" foo =\n"))

		key, value, err := p.ReadHeader()

		assert.Nil(t, key)
		assert.Nil(t, value)
		assert.Equal(t, InvalidValue, err)
	})

	t.Run("invalid value", func(t *testing.T) {
		p := NewParser([]byte(" foo = *** \n"))

		key, value, err := p.ReadHeader()

		assert.Nil(t, key)
		assert.Nil(t, value)
		assert.Equal(t, InvalidValue, err)
	})
}

func TestEndOfMetrics(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		p := NewParser([]byte{})

		key, ts, value, err := p.ReadMetric()

		assert.Nil(t, key)
		assert.Nil(t, value)
		assert.Nil(t, ts)
		assert.Equal(t, EndOfMetrics, err)

		assert.Equal(t, []byte{}, p.buf)
	})

	t.Run("only white spaces", func(t *testing.T) {
		p := NewParser([]byte("   "))

		key, ts, value, err := p.ReadMetric()

		assert.Nil(t, key)
		assert.Nil(t, value)
		assert.Nil(t, ts)
		assert.Equal(t, EndOfMetrics, err)

		assert.Equal(t, []byte{}, p.buf)
	})

	t.Run("white with linebreak", func(t *testing.T) {
		p := NewParser([]byte("   \n"))

		key, ts, value, err := p.ReadMetric()

		assert.Nil(t, key)
		assert.Nil(t, value)
		assert.Nil(t, ts)
		assert.Equal(t, EndOfMetrics, err)

		assert.Equal(t, []byte{}, p.buf)
	})
}

func TestSingleMetric(t *testing.T) {
	t.Run("without linebreak", func(t *testing.T) {
		p := NewParser([]byte("   foo      20.6    1234567890   "))

		key, ts, value, err := p.ReadMetric()

		assert.Equal(t, []byte("foo"), key)
		assert.Equal(t, []byte("20.6"), value)
		assert.Equal(t, []byte("1234567890"), ts)
		assert.Nil(t, err)

		assert.Equal(t, []byte{}, p.buf)
	})

	t.Run("with linebreak", func(t *testing.T) {
		p := NewParser([]byte("   foo    20.6    1234567890     \n"))

		key, ts, value, err := p.ReadMetric()

		assert.Equal(t, []byte("foo"), key)
		assert.Equal(t, []byte("20.6"), value)
		assert.Equal(t, []byte("1234567890"), ts)
		assert.Nil(t, err)

		assert.Equal(t, []byte{}, p.buf)
	})
}

func TestMultipleMetrics(t *testing.T) {
	p := NewParser([]byte("   foo    20.6    1234567890     \n bar  17.999   12345\nbaz ... 1337"))

	// first metric
	key, ts, value, err := p.ReadMetric()

	assert.Equal(t, []byte("foo"), key)
	assert.Equal(t, []byte("20.6"), value)
	assert.Equal(t, []byte("1234567890"), ts)
	assert.Nil(t, err)

	assert.Equal(t, []byte(" bar  17.999   12345\nbaz ... 1337"), p.buf)

	// second metric
	key, ts, value, err = p.ReadMetric()

	assert.Equal(t, []byte("bar"), key)
	assert.Equal(t, []byte("17.999"), value)
	assert.Equal(t, []byte("12345"), ts)
	assert.Nil(t, err)

	assert.Equal(t, []byte("baz ... 1337"), p.buf)

	// third metric
	key, ts, value, err = p.ReadMetric()

	assert.Equal(t, []byte("baz"), key)
	assert.Equal(t, []byte("..."), value) // this is absolutely valid here, actual validation must be done outside of the parser
	assert.Equal(t, []byte("1337"), ts)
	assert.Nil(t, err)

	assert.Equal(t, []byte{}, p.buf)
}

func TestInvalidMetrics(t *testing.T) {
	t.Run("invalid key", func(t *testing.T) {
		p := NewParser([]byte("  *** 123 123"))

		key, ts, value, err := p.ReadMetric()

		assert.Nil(t, key)
		assert.Nil(t, value)
		assert.Nil(t, ts)
		assert.Equal(t, InvalidKey, err)
	})

	t.Run("invalid timestamp", func(t *testing.T) {
		p := NewParser([]byte(" foo  timestamp 123"))

		key, ts, value, err := p.ReadMetric()

		assert.Nil(t, key)
		assert.Nil(t, value)
		assert.Nil(t, ts)
		assert.Equal(t, InvalidValue, err)
	})

	t.Run("invalid value", func(t *testing.T) {
		p := NewParser([]byte("   foo    123   value \n"))

		key, ts, value, err := p.ReadMetric()

		assert.Nil(t, key)
		assert.Nil(t, value)
		assert.Nil(t, ts)
		assert.Equal(t, InvalidValue, err)
	})
}

func TestMessageDecoding(t *testing.T) {
	t.Run("empty message", func(t *testing.T) {
		var msg *Message

		err := DecodeMessage([]byte{}, msg)

		assert.Error(t, err)
	})

	t.Run("invalid header", func(t *testing.T) {
		msg := &Message{}
		rawMsg := []byte("foo=bar; baz\nfps 30 1234567890\n")

		err := DecodeMessage(rawMsg, msg)

		assert.Equal(t, IncompletePair, err)
	})

	t.Run("invalid metric", func(t *testing.T) {
		msg := &Message{}
		rawMsg := []byte("foo=bar;\nfps invalid 30\n")

		err := DecodeMessage(rawMsg, msg)

		assert.Equal(t, InvalidValue, err)
	})

	t.Run("valid message", func(t *testing.T) {
		msg := &Message{}
		rawMsg := []byte("project=my_project; foo=bar\nfps 30 1234567890\nmemory_usage 102400 1234567891")

		err := DecodeMessage(rawMsg, msg)

		assert.Len(t, msg.Header, 2)
		assert.Len(t, msg.Metrics, 2)
		assert.Equal(t, []byte("my_project"), msg.Header["project"])
		assert.Equal(t, []byte("bar"), msg.Header["foo"])
		assert.Equal(t, &Metric{[]byte("fps"), []byte("30"), []byte("1234567890")}, msg.Metrics[0])
		assert.Equal(t, &Metric{[]byte("memory_usage"), []byte("102400"), []byte("1234567891")}, msg.Metrics[1])
		assert.Nil(t, err)
	})
}

func benchmarkParseHeaders(n int, b *testing.B) {
	input := append(bytes.Repeat([]byte("some_key = some_value ; "), n), '\n')
	var p *Parser

	for i := 0; i < b.N; i++ {
		p = NewParser(input)
		for x := 0; x < n; x++ {
			p.ReadHeader()
		}
	}
}

func BenchmarkParse1Header(b *testing.B)    { benchmarkParseHeaders(1, b) }
func BenchmarkParse10Headers(b *testing.B)  { benchmarkParseHeaders(10, b) }
func BenchmarkParse100Headers(b *testing.B) { benchmarkParseHeaders(100, b) }

func benchmarkParseMetrics(n int, b *testing.B) {
	input := bytes.Repeat([]byte("some_metric_name 1234567890 1234567890\n"), n)
	var p *Parser

	for i := 0; i < b.N; i++ {
		p = NewParser(input)
		for x := 0; x < n; x++ {
			p.ReadMetric()
		}
	}
}

func BenchmarkParse1Metric(b *testing.B)     { benchmarkParseMetrics(1, b) }
func BenchmarkParse10Metrics(b *testing.B)   { benchmarkParseMetrics(10, b) }
func BenchmarkParse100Metrics(b *testing.B)  { benchmarkParseMetrics(100, b) }
func BenchmarkParse1000Metrics(b *testing.B) { benchmarkParseMetrics(1000, b) }
