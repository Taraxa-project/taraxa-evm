package trie

import (
	"fmt"
	"math/rand"
	"reflect"
	"strconv"

	"github.com/emicklei/dot"
)

func dot_draw_level(g *dot.Graph, n node) {
	if g == nil {
		return
	}
	switch n := n.(type) {
	case *full_node:
		for _, c := range n.children {
			g.Edge(dot_node(g, n), dot_node(g, c))
		}
	case *short_node:
		g.Edge(dot_node(g, n), dot_node(g, n.val))
	}
}

func dot_node(g *dot.Graph, n node) (ret dot.Node) {
	reflect_n := reflect.ValueOf(n)
	switch n := n.(type) {
	case *short_node, *full_node:
		ret = g.Node(fmt.Sprint(reflect_n.Pointer()))
		ret.Label(reflect_n.Type().String())
	default:
		ret = g.Node(strconv.FormatUint(rand.Uint64(), 10))
		if n == nil {
			ret.Label("NULL")
			return
		}
		ret.Label(reflect_n.Type().String())
		if _, ok := n.(value_node); ok {
			g.AddToSameRank("leaves", ret)
		}
	}
	return
}
