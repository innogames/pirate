package pirate

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEmptyTemplate(t *testing.T) {
	t.Run("parse", func(t *testing.T) {
		tpl, err := ParsePathTemplate([]byte(""))

		assert.Nil(t, tpl.parts)
		assert.Nil(t, err)
	})

	t.Run("resolve", func(t *testing.T) {
		tpl := &pathTemplate{}

		resolved, err := tpl.Resolve(&Context{})

		assert.Nil(t, resolved)
		assert.Nil(t, err)
	})
}

func TestParseSimpleTemplate(t *testing.T) {
	t.Run("one static part", func(t *testing.T) {
		tpl, err := ParsePathTemplate([]byte("foo.bar.baz"))

		assert.Len(t, tpl.parts, 1)
		assert.Equal(t, &staticNode{[]byte("foo.bar.baz")}, tpl.parts[0])
		assert.Nil(t, err)
	})

	t.Run("one attr var", func(t *testing.T) {
		tpl, err := ParsePathTemplate([]byte("{attr.foo}"))

		assert.Len(t, tpl.parts, 1)
		assert.Equal(t, &attrNode{"foo"}, tpl.parts[0])
		assert.Nil(t, err)
	})

	t.Run("one metric var", func(t *testing.T) {
		tpl, err := ParsePathTemplate([]byte("{metric.foo}"))

		assert.Error(t, err)
		assert.Nil(t, tpl)
	})
}

func TestParseMixedTemplate(t *testing.T) {
	t.Run("one attribute in between", func(t *testing.T) {
		tpl, err := ParsePathTemplate([]byte("foo.{attr.bar}.baz"))

		assert.Equal(
			t,
			[]node{
				&staticNode{[]byte("foo.")},
				&attrNode{"bar"},
				&staticNode{[]byte(".baz")},
			},
			tpl.parts,
		)
		assert.Nil(t, err)
	})

	t.Run("metric and attribute in between", func(t *testing.T) {
		tpl, err := ParsePathTemplate([]byte("foo.{attr.bar}.baz.{metric.name}.something.else"))

		assert.Equal(
			t,
			[]node{
				&staticNode{[]byte("foo.")},
				&attrNode{"bar"},
				&staticNode{[]byte(".baz.")},
				&metricNameNode{},
				&staticNode{[]byte(".something.else")},
			},
			tpl.parts,
		)
		assert.Nil(t, err)
	})
}

type testNode struct {
	f func(ctx *Context) ([]byte, error)
}

func (node *testNode) Resolve(ctx *Context) ([]byte, error) {
	return node.f(ctx)
}

func TestResolveTemplate(t *testing.T) {
	t.Run("successfull resolution", func(t *testing.T) {
		tpl := &pathTemplate{
			parts: []node{
				&testNode{func(ctx *Context) ([]byte, error) { return []byte("aa"), nil }},
				&testNode{func(ctx *Context) ([]byte, error) { return []byte("bb"), nil }},
				&testNode{func(ctx *Context) ([]byte, error) { return []byte("cc"), nil }},
				&testNode{func(ctx *Context) ([]byte, error) { return []byte("dd"), nil }},
				&testNode{func(ctx *Context) ([]byte, error) { return []byte("ee"), nil }},
			},
		}

		res, err := tpl.Resolve(&Context{})
		assert.Equal(t, []byte("aabbccddee"), res)
		assert.Nil(t, err)
	})

	t.Run("failing node", func(t *testing.T) {
		called := 0

		tpl := &pathTemplate{
			parts: []node{
				&testNode{func(ctx *Context) ([]byte, error) { called++; return []byte("aa"), nil }},
				&testNode{func(ctx *Context) ([]byte, error) { called++; return []byte("bb"), nil }},
				&testNode{func(ctx *Context) ([]byte, error) { called++; return []byte("cc"), nil }},
				&testNode{func(ctx *Context) ([]byte, error) { called++; return nil, errors.New("dummy") }},
				&testNode{func(ctx *Context) ([]byte, error) { called++; return []byte("ee"), nil }},
			},
		}

		b, err := tpl.Resolve(&Context{})
		assert.Nil(t, b, "primary result must be nil, when error occures")
		assert.Equal(t, 4, called, "only first 4 nodes until the error must be resolved")
		assert.Error(t, err)
	})
}

func TestResolveNodes(t *testing.T) {
	t.Run("static node", func(t *testing.T) {
		node := &staticNode{[]byte("foo.bar.baz")}

		res, err := node.Resolve(&Context{})

		assert.Equal(t, []byte("foo.bar.baz"), res)
		assert.Nil(t, err)
	})

	t.Run("attr node with existing value", func(t *testing.T) {
		node := &attrNode{"foo"}

		ctx := &Context{attr: map[string][]byte{"foo": []byte("some_value")}}
		res, err := node.Resolve(ctx)

		assert.Equal(t, []byte("some_value"), res)
		assert.Nil(t, err)
	})

	t.Run("attr node with dot in value", func(t *testing.T) {
		node := &attrNode{"version"}

		ctx := &Context{attr: map[string][]byte{"version": []byte("1.3.37")}}
		res, err := node.Resolve(ctx)

		assert.Equal(t, []byte("1_3_37"), res)
		assert.Nil(t, err)
	})

	t.Run("attr node with unknown value", func(t *testing.T) {
		node := &attrNode{"bar"}

		ctx := &Context{attr: map[string][]byte{"foo": []byte("some_value")}}
		res, err := node.Resolve(ctx)

		assert.Nil(t, res)
		assert.Error(t, err)
	})

	t.Run("metric node with existing value", func(t *testing.T) {
		node := &attrNode{"value"}

		ctx := &Context{attr: map[string][]byte{"value": []byte("1337")}}
		res, err := node.Resolve(ctx)

		assert.Equal(t, []byte("1337"), res)
		assert.Nil(t, err)
	})

	t.Run("metric node with dot in value", func(t *testing.T) {
		node := &attrNode{"name"}

		ctx := &Context{attr: map[string][]byte{"name": []byte("something_60.0")}}
		res, err := node.Resolve(ctx)

		assert.Equal(t, []byte("something_60_0"), res)
		assert.Nil(t, err)
	})

	t.Run("metric node with unknown value", func(t *testing.T) {
		node := &attrNode{"bar"}

		ctx := &Context{attr: map[string][]byte{"value": []byte("1337")}}
		res, err := node.Resolve(ctx)

		assert.Nil(t, res)
		assert.Error(t, err)
	})
}
