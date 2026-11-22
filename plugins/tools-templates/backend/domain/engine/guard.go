package engine

import (
	"fmt"
	"text/template"
	"text/template/parse"
)

const (
	tickName  = "__tick"
	enterName = "__enter"
	leaveName = "__leave"

	maxSteps = 2_000_000
	maxDepth = 1000
)

var errTooDeep = fmt.Errorf("%w: templates call each other too deep", ErrTooSlow)

func (p *pass) tick() (string, error) {
	if p.stopped.Load() {
		return "", ErrTooSlow
	}
	if p.steps++; p.steps > maxSteps {
		return "", ErrTooSlow
	}
	return "", nil
}

func (p *pass) enter() (string, error) {
	if p.depth++; p.depth > maxDepth {
		return "", errTooDeep
	}
	return p.tick()
}

func (p *pass) leave() (string, error) {
	p.depth--
	return "", nil
}

func guard(parsed *template.Template) {
	for _, t := range parsed.Templates() {
		if t.Tree == nil || t.Tree.Root == nil {
			continue
		}
		guardList(t.Tree.Root)
		root := t.Tree.Root
		root.Nodes = append(append([]parse.Node{call(enterName)}, root.Nodes...), call(leaveName))
	}
}

func guardList(list *parse.ListNode) {
	if list == nil {
		return
	}
	for _, node := range list.Nodes {
		switch n := node.(type) {
		case *parse.RangeNode:
			guardBranch(&n.BranchNode)
			n.List.Nodes = append([]parse.Node{call(tickName)}, n.List.Nodes...)
			if n.ElseList != nil {
				n.ElseList.Nodes = append([]parse.Node{call(tickName)}, n.ElseList.Nodes...)
			}
		case *parse.IfNode:
			guardBranch(&n.BranchNode)
		case *parse.WithNode:
			guardBranch(&n.BranchNode)
		}
	}
}

func guardBranch(branch *parse.BranchNode) {
	guardList(branch.List)
	guardList(branch.ElseList)
}

func call(name string) parse.Node {
	command := &parse.CommandNode{NodeType: parse.NodeCommand, Args: []parse.Node{parse.NewIdentifier(name)}}
	return &parse.ActionNode{NodeType: parse.NodeAction, Pipe: &parse.PipeNode{NodeType: parse.NodePipe, Cmds: []*parse.CommandNode{command}}}
}
