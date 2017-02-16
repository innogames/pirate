package pirate

import (
	"bytes"
	"fmt"
	"regexp"
)

var (
	attrRegexp = regexp.MustCompile(`\{([a-z]+)\.([a-zA-Z][a-zA-Z0-9_]*)}`)
)

type Context struct {
	attr   map[string][]byte
	metric *Metric
}

func NewCtx(attr map[string][]byte, metric *Metric) *Context {
	return &Context{attr, metric}
}

type pathTemplate struct {
	parts []node
}

func ParsePathTemplate(input []byte) (*pathTemplate, error) {
	tpl := &pathTemplate{}

	prev := 0
	for _, match := range attrRegexp.FindAllSubmatchIndex(input, -1) {
		start, end := match[0], match[1]

		// everything between placeholders is static
		if start > prev {
			tpl.parts = append(tpl.parts, &staticNode{input[prev:start]})
		}

		holder := input[match[2]:match[3]]
		name := input[match[4]:match[5]]

		switch string(holder) {
		case "attr":
			tpl.parts = append(tpl.parts, &attrNode{string(name)})
		case "metric":
			if string(name) != "name" {
				return nil, fmt.Errorf(`Invalid member name "%s" on "metric", only "name" allowed`, name)
			}
			tpl.parts = append(tpl.parts, &metricNameNode{})
		default:
			return nil, fmt.Errorf(`Invalid variable holder "%s", only "attr" and "metric" allowed`, holder)
		}

		prev = end
	}

	// remaining static part
	if rest := input[prev:]; len(rest) > 0 {
		tpl.parts = append(tpl.parts, &staticNode{rest})
	}

	return tpl, nil
}

func (tpl *pathTemplate) Resolve(ctx *Context) ([]byte, error) {
	var buf []byte

	for _, p := range tpl.parts {
		b, err := p.Resolve(ctx)
		if err != nil {
			return nil, err
		}

		buf = append(buf, b...)
	}

	return buf, nil
}

type node interface {
	Resolve(ctx *Context) ([]byte, error)
}

type staticNode struct {
	value []byte
}

func (node *staticNode) Resolve(ctx *Context) ([]byte, error) {
	return node.value, nil
}

type attrNode struct {
	name string
}

func (node attrNode) Resolve(ctx *Context) ([]byte, error) {
	if value, ok := ctx.attr[node.name]; ok {
		if bytes.IndexByte(value, '.') != -1 {
			value = bytes.Replace(value, []byte{'.'}, []byte{'_'}, -1)
		}
		return value, nil
	}

	return nil, fmt.Errorf(`Failed to resolve attribute "%s"`, node.name)
}

type metricNameNode struct{}

func (node metricNameNode) Resolve(ctx *Context) ([]byte, error) {
	return ctx.metric.Name, nil
}
