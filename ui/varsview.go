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
	returns *tview.TreeNode

	expandedCache map[uint64]bool
}

func NewVarsView() *VarsView {
	gHeader := tview.NewTreeNode("globals").SetColor(tcell.Color101).SetSelectable(true)
	lHeader := tview.NewTreeNode("locals").SetColor(tcell.Color101).SetSelectable(true)
	aHeader := tview.NewTreeNode("args").SetColor(tcell.Color101).SetSelectable(true)
	rHeader := tview.NewTreeNode("return").SetColor(tcell.Color101).SetSelectable(true)

	topHeader := tview.NewTreeNode("").SetColor(tcell.ColorRebeccaPurple).SetSelectable(true)
	tree := tview.NewTreeView().SetRoot(topHeader)
	tree.SetCurrentNode(topHeader)
	tree.SetBackgroundColor(tcell.ColorDefault)
	topHeader.AddChild(gHeader)
	topHeader.AddChild(lHeader)
	topHeader.AddChild(aHeader)
	topHeader.AddChild(rHeader)

	return &VarsView{
		tree: tree,
		globals: gHeader,
		locals: lHeader,
		args: aHeader,
		expandedCache: make(map[uint64]bool),
	}
}

func (varsView *VarsView)RenderDebuggerMove(args []api.Variable, locals []api.Variable, globals []api.Variable, returns []api.Variable) {

	varsView.locals.ClearChildren()
	varsView.args.ClearChildren()
	varsView.globals.ClearChildren()

	varsView.AddVars(varsView.locals, locals)
	varsView.AddVars(varsView.args, args)
	varsView.AddVars(varsView.globals, globals)
}

func getVarTitle(vr *api.Variable, expanded bool) string {
	namestr := fmt.Sprintf("[green::b]%s", vr.Name)
	typestr := fmt.Sprintf("[purple]<%s>[white:-:-]",vr.RealType)
	valstr := ""
	if vr.Value != "" {
		valstr += fmt.Sprintf(" %s", vr.Value)
	}
	suffix := ""
	if vr.Children != nil && len(vr.Children) > 0 {
		if expanded {
			suffix = " [blue]-"
		} else {
			suffix = " [blue]+"
		}
	}
	return namestr + typestr + valstr + suffix
}

func (varsView *VarsView)AddVars(parent *tview.TreeNode, vars []api.Variable ){
	for vi := range vars {
		vr := vars[vi]
		newNode := tview.NewTreeNode( getVarTitle(&vr, varsView.expandedCache[vr.Addr]) ).
			SetReference(vr)
		newNode.SetSelectable(true)
		newNode.SetColor(tcell.ColorBlack)

		// If node has children, initially collapse. Expand on select.
		if vr.Children != nil && len(vr.Children) > 0 {
			varsView.AddVars(newNode, vr.Children)
			if !varsView.expandedCache[vr.Addr] {
				newNode.CollapseAll()
			}
			newNode.SetSelectedFunc(func() {
				varsView.expandedCache[vr.Addr] = !newNode.IsExpanded()
				r := newNode.GetReference().(api.Variable);
				if !newNode.IsExpanded() {
					newNode.Expand()
				} else {
					newNode.Collapse()
				}
				newNode.SetText( getVarTitle(&r, varsView.expandedCache[vr.Addr] ) )
			})
		}
		parent.AddChild(newNode)
	}
}

func (varsView *VarsView) GetWidget() *tview.TreeView {
	return varsView.tree
}
