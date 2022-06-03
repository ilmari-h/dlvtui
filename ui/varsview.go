package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/go-delve/delve/service/api"
	"github.com/rivo/tview"
)

type VarsView struct {
	tree *tview.TreeView
	globals *tview.TreeNode
	locals *tview.TreeNode
	args *tview.TreeNode
	vars *tview.TreeNode

	expandedCache map[uint64]bool
}

func NewVarsView() *VarsView {
	gHeader := tview.NewTreeNode("globals").SetColor(tcell.Color101).SetSelectable(true)
	lHeader := tview.NewTreeNode("locals").SetColor(tcell.Color101).SetSelectable(true)
	aHeader := tview.NewTreeNode("args").SetColor(tcell.Color101).SetSelectable(true)
	vHeader := tview.NewTreeNode("vars").SetColor(tcell.Color101).SetSelectable(true)

	topHeader := tview.NewTreeNode("").SetColor(tcell.ColorRebeccaPurple).SetSelectable(true)
	tree := tview.NewTreeView().SetRoot(topHeader)
	tree.SetCurrentNode(topHeader)
	tree.SetBackgroundColor(tcell.ColorDefault)
	topHeader.AddChild(gHeader)
	topHeader.AddChild(lHeader)
	topHeader.AddChild(aHeader)
	topHeader.AddChild(vHeader)

	return &VarsView{
		tree: tree,
		globals: gHeader,
		locals: lHeader,
		args: aHeader,
		vars: vHeader,
		expandedCache: make(map[uint64]bool),
	}
}

func (varsView *VarsView)RenderBreakpointHit(state *api.BreakpointInfo) {
	varsView.locals.ClearChildren()
	varsView.args.ClearChildren()
	varsView.globals.ClearChildren()

	varsView.AddVars(varsView.locals, state.Locals)
	varsView.AddVars(varsView.args, state.Arguments)
	varsView.AddVars(varsView.globals, state.Variables)
}

func (varsView *VarsView)AddVars(parent *tview.TreeNode, vars []api.Variable ){
	for vi := range vars {
		vr := vars[vi]
		newNode := tview.NewTreeNode( fmt.Sprintf("[green::b]%s[purple]<%s>[white:-:-]: %s",vr.Name, vr.Type, vr.Value)).
			SetReference(vr)
		newNode.SetSelectable(true)
		newNode.SetColor(tcell.ColorBlack)

		// If node has children, initially collapse. Expand on select.
		if vr.Children != nil && len(vr.Children) > 0 {
			varsView.AddVars(newNode, vr.Children)
			if !varsView.expandedCache[vr.Addr] {
				newNode.SetText( newNode.GetText() + " [+]" )
				newNode.CollapseAll()
			}
			newNode.SetSelectedFunc(func() {
				varsView.expandedCache[vr.Addr] = !newNode.IsExpanded()
				r := newNode.GetReference().(api.Variable);
				if !newNode.IsExpanded() {
					newNode.SetText( fmt.Sprintf("[green::b]%s[purple]<%s>[white:-:-]: %s",r.Name, r.Type, r.Value))
					newNode.Expand()
				} else {
					newNode.SetText( fmt.Sprintf("[green::b]%s[purple]<%s>[white:-:-]: %s [+]",r.Name, r.Type, r.Value))
					newNode.Collapse()
				}
			})
		}
		parent.AddChild(newNode)
	}
}

func (varsView *VarsView) GetWidget() *tview.TreeView {
	return varsView.tree
}
