package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/go-delve/delve/service/api"
	"github.com/rivo/tview"
)

type VarsPage struct {
	widget         tview.Primitive
	commandHandler *CommandHandler

	treeView *tview.TreeView
	locals   *tview.TreeNode

	args *tview.TreeNode

	returns *tview.TreeNode

	varHeaders   []*tview.TreeNode
	varHeaderIdx int

	expandedCache map[uint64]bool

	lastSelected struct {
		exists bool
		val    api.Variable
	}
}

func NewVarPage() *VarsPage {

	root := tview.NewTreeNode(".").
		SetColor(tcell.ColorDefault).
		SetSelectable(false)
	treeView := tview.NewTreeView().
		SetRoot(root)
	treeView.SetBackgroundColor(tcell.ColorDefault)
	treeView.SetInputCapture(listInputCaptureC)

	localsHeader := tview.NewTreeNode(fmt.Sprintf("[%s::b]locals",
		iToColorS(gConfig.Colors.ListHeaderFg),
	)).
		SetColor(iToColorTcell(gConfig.Colors.ListHeaderFg)).
		SetSelectable(false)

	argsHeader := tview.NewTreeNode(fmt.Sprintf("[%s::b]arguments",
		iToColorS(gConfig.Colors.ListHeaderFg),
	)).
		SetColor(tcell.ColorGreen).
		SetSelectable(false)

	returnsHeader := tview.NewTreeNode(fmt.Sprintf("[%s::b]return values",
		iToColorS(gConfig.Colors.ListHeaderFg),
	)).
		SetColor(iToColorTcell(gConfig.Colors.ListHeaderFg)).
		SetSelectable(false)

	treeView.GetRoot().AddChild(localsHeader)
	treeView.GetRoot().AddChild(argsHeader)
	treeView.GetRoot().AddChild(returnsHeader)

	pageFrame := tview.NewFrame(treeView).
		SetBorders(0, 0, 0, 0, 0, 0).
		AddText(fmt.Sprintf("[%s::b]Current stack frame:", iToColorS(gConfig.Colors.HeaderFg)),
			true,
			tview.AlignLeft,
			tcell.ColorWhite,
		)
	pageFrame.SetBackgroundColor(tcell.ColorDefault)

	return &VarsPage{
		widget: pageFrame,

		treeView: treeView,
		locals:   localsHeader,
		args:     argsHeader,
		returns:  returnsHeader,

		varHeaders:    []*tview.TreeNode{localsHeader, argsHeader, returnsHeader},
		varHeaderIdx:  0,
		expandedCache: make(map[uint64]bool),
	}
}

func (page *VarsPage) RenderVariables(args []api.Variable, locals []api.Variable, returns []api.Variable) {

	page.locals.ClearChildren()
	page.args.ClearChildren()

	page.AddVars(page.locals, locals)
	page.AddVars(page.args, args)
	page.AddVars(page.returns, returns)

	if !page.lastSelected.exists {
		foundSelectable := false
		page.treeView.GetRoot().Walk(func(node, parent *tview.TreeNode) bool {
			if !foundSelectable && node.GetReference() != nil {
				foundSelectable = true
				page.treeView.SetCurrentNode(node)
			}
			return !foundSelectable
		})
	}
}

func getVarTitle(vr *api.Variable, expanded bool) string {
	namestr := fmt.Sprintf("[%s::b]%s", iToColorS(gConfig.Colors.VarNameFg), vr.Name)
	typestr := fmt.Sprintf("[%s]<%s>[%s:-:-]",
		iToColorS(gConfig.Colors.VarTypeFg),
		vr.RealType,
		iToColorS(gConfig.Colors.VarValueFg),
	)
	valstr := ""
	addrstr := fmt.Sprintf("[%s] 0x%x", iToColorS(gConfig.Colors.VarAddrFg), vr.Addr)
	if vr.Value != "" {
		valstr += fmt.Sprintf(" %s", vr.Value)
	}
	suffix := ""
	if vr.Children != nil && len(vr.Children) > 0 {
		suffix = fmt.Sprintf(" [%s]", iToColorS(gConfig.Colors.ListExpand))
		if expanded {
			suffix += "-"
		} else {
			suffix += "+"
		}
	}
	return namestr + typestr + valstr + suffix + addrstr
}

func (page *VarsPage) AddVars(parent *tview.TreeNode, vars []api.Variable) {

	addedLocals := 0
	addedArgs := 0
	for _, vr := range vars {
		newNode := tview.NewTreeNode(getVarTitle(&vr, page.expandedCache[vr.Addr])).
			SetReference(vr)
		newNode.SetSelectable(true)
		newNode.SetColor(tcell.ColorBlack)

		if vr.Addr == page.lastSelected.val.Addr {
			page.treeView.SetCurrentNode(newNode)
		}

		if parent == page.locals {
			addedLocals++
		} else if parent == page.args {
			addedArgs++
		}

		// If node has children, initially collapse. Expand on select.
		if vr.Children != nil && len(vr.Children) > 0 {
			page.AddVars(newNode, vr.Children)

			// Expand or collapse node according to what was cached from previous action.
			if !page.expandedCache[vr.Addr] {
				newNode.CollapseAll()
			} else {
				newNode.Expand()
			}

			newNode.SetSelectedFunc(func() {
				r := newNode.GetReference().(api.Variable)
				page.expandedCache[r.Addr] = !newNode.IsExpanded()
				if !newNode.IsExpanded() {
					newNode.Expand()
				} else {
					newNode.Collapse()
				}
				newNode.SetText(getVarTitle(&r, page.expandedCache[r.Addr]))
			})
		}
		parent.AddChild(newNode)
	}
}

func (varsView *VarsPage) GetName() string {
	return "vars"
}

func (page *VarsPage) HandleKeyEvent(event *tcell.EventKey) *tcell.EventKey {

	page.treeView.InputHandler()(event, func(p tview.Primitive) {})
	if page.treeView.GetCurrentNode() != nil && page.treeView.GetCurrentNode().GetReference() != nil {
		page.lastSelected.val = page.treeView.GetCurrentNode().GetReference().(api.Variable)
		page.lastSelected.exists = true
	}
	return nil
}

func (page *VarsPage) SetCommandHandler(ch *CommandHandler) {
	page.commandHandler = ch
}

func (page *VarsPage) GetWidget() tview.Primitive {
	return page.widget
}
