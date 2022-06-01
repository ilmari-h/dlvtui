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
}

type VarListing int
const (
	Local VarListing = iota
	Global
	Args
	Vars
)

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
	}
}


func (varsView *VarsView)AddVars(parent *tview.TreeNode, vars []api.Variable, listing VarListing) {
	for vi := range vars {
		vr := vars[vi]
		newNode := tview.NewTreeNode( fmt.Sprintf("[green::b]%s[purple]<%s>[white:-:-]: %s",vr.Name, vr.Type, vr.Value)).
			SetReference(vr)
		newNode.SetSelectable(true)
		newNode.SetColor(tcell.ColorBlack)
		if parent == nil {
			if listing == Local {
				varsView.locals.AddChild(newNode)
			} else if listing == Args {
				varsView.args.AddChild(newNode)
			} else if listing == Global {
				varsView.globals.AddChild(newNode)
			} else {
				varsView.vars.AddChild(newNode)
			}
		} else {
			parent.AddChild(newNode)
		}
		if vr.Children != nil && len(vr.Children) > 0 {
			varsView.AddVars(newNode,vr.Children,listing)
		}
	}
}

func (varsView *VarsView) GetWidget() *tview.TreeView {
	return varsView.tree
}
