// Package effectfulcomposition builds the finite two-original frame around a
// legal Series 1 tree. All run-time choices are instructions in retained programs.
package effectfulcomposition

import (
	"fmt"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/owningtree"
	"github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/examples/substitution/b"
)

// Tree carries the retained template and Series 1's common occurrence bounds.
type Tree struct {
	template     author.Template
	slots, depth int64
	paths        []string
}

// Template returns a copy of the immutable compiled subtree.
func (t Tree) Template() author.Template { return t.template }
func (t Tree) Slots() int64              { return t.slots }
func (t Tree) Depth() int64              { return t.depth }

func Leaf(candidate string) (Tree, error) {
	var t author.Template
	var err error
	switch candidate {
	case "A":
		t, err = a.New()
	case "B":
		t, err = b.New()
	default:
		return Tree{}, fmt.Errorf("unknown pure leaf %q", candidate)
	}
	if err != nil {
		return Tree{}, err
	}
	t, err = owningtree.Leaf(t)
	return Tree{template: t, slots: 4, depth: 3, paths: []string{""}}, err
}
func Relay(child Tree) (Tree, error) {
	t, err := owningtree.Relay(child.template)
	return Tree{template: t, slots: child.slots + 1, depth: child.depth + 1, paths: child.paths}, err
}
func Fork(left, right Tree) (Tree, error) {
	t, err := owningtree.Fork(left.template, right.template)
	var paths []string
	for _, p := range left.paths {
		paths = append(paths, "l_"+p)
	}
	for _, p := range right.paths {
		paths = append(paths, "r_"+p)
	}
	return Tree{template: t, slots: 1 + left.slots + right.slots, depth: 1 + max(left.depth, right.depth), paths: paths}, err
}

// PureFork constructs the two-leaf grid and the selected relay placements.
func PureFork(left, right, relay string) (Tree, error) {
	l, err := Leaf(left)
	if err != nil {
		return Tree{}, err
	}
	r, err := Leaf(right)
	if err != nil {
		return Tree{}, err
	}
	switch relay {
	case "", "above":
	case "left":
		l, err = Relay(l)
	case "right":
		r, err = Relay(r)
	default:
		return Tree{}, fmt.Errorf("unknown relay placement %q", relay)
	}
	if err != nil {
		return Tree{}, err
	}
	t, err := Fork(l, r)
	if err == nil && relay == "above" {
		return Relay(t)
	}
	return t, err
}
