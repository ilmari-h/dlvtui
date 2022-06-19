package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/go-delve/delve/service/api"
	"github.com/rivo/tview"
)

type VarsPage struct {
	flex           *tview.Flex
	widget         tview.Primitive
	commandHandler *CommandHandler

	locals     *tview.TreeNode
	localsTree *tview.TreeView

	args     *tview.TreeNode
	argsTree *tview.TreeView

	returns     *tview.TreeNode
	returnsTree *tview.TreeView

	varHeaders   []*tview.TreeView
	varHeaderIdx int

	expandedCache map[uint64]bool
}

func NewVarPage() *VarsPage {
	localsHeader := tview.NewTreeNode(fmt.Sprintf("[%s::b]locals",
		iToColorS(gConfig.Colors.ListHeaderFg),
	)).
		SetColor(tcell.ColorGreen).
		SetSelectable(true)
	localsHeader.SetSelectable(false)
	localsTree := tview.NewTreeView().SetRoot(localsHeader)
	localsTree.SetCurrentNode(localsHeader)
	localsTree.SetBackgroundColor(tcell.ColorDefault)
	localsTree.SetInputCapture(listInputCaptureC)

	argsHeader := tview.NewTreeNode(fmt.Sprintf("[%s::b]arguments",
		iToColorS(gConfig.Colors.ListHeaderFg),
	)).
		SetColor(tcell.ColorGreen).SetSelectable(true)
	argsHeader.SetSelectable(false)
	argsTree := tview.NewTreeView().SetRoot(argsHeader)
	argsTree.SetCurrentNode(argsHeader)
	argsTree.SetBackgroundColor(tcell.ColorDefault)
	argsTree.SetInputCapture(listInputCaptureC)

	returnsHeader := tview.NewTreeNode(fmt.Sprintf("[%s::b]return values",
		iToColorS(gConfig.Colors.ListHeaderFg),
	)).
		SetColor(tcell.ColorGreen).
		SetSelectable(true)
	returnsHeader.SetSelectable(false)
	returnsTree := tview.NewTreeView().SetRoot(returnsHeader)
	returnsTree.SetCurrentNode(returnsHeader)
	returnsTree.SetBackgroundColor(tcell.ColorDefault)
	returnsTree.SetInputCapture(listInputCaptureC)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(localsTree, 1, 1, false).
		AddItem(argsTree, 1, 1, false).
		AddItem(returnsTree, 1, 1, false)

	pageFrame := tview.NewFrame(flex).
		SetBorders(0, 0, 0, 0, 0, 0).
		AddText(fmt.Sprintf("[%s::b]Current stack frame:", iToColorS(gConfig.Colors.HeaderFg)),
			true,
			tview.AlignLeft,
			tcell.ColorWhite,
		)
	pageFrame.SetBackgroundColor(tcell.ColorDefault)

	return &VarsPage{
		flex:   flex,
		widget: pageFrame,

		locals:     localsHeader,
		localsTree: localsTree,

		args:     argsHeader,
		argsTree: argsTree,

		returns:     returnsHeader,
		returnsTree: returnsTree,

		varHeaders:    []*tview.TreeView{localsTree, argsTree, returnsTree},
		varHeaderIdx:  0,
		expandedCache: make(map[uint64]bool),
	}
}

func (page *VarsPage) resizeTrees() {
	visLocals := 0
	visArgs := 0
	page.locals.Walk(func(node, parent *tview.TreeNode) bool {
		visLocals++
		return node.IsExpanded()
	})

	page.args.Walk(func(node, parent *tview.TreeNode) bool {
		visArgs++
		return node.IsExpanded()
	})

	page.flex.ResizeItem(page.localsTree, visLocals, 0)
	page.flex.ResizeItem(page.argsTree, visArgs, 0)

}

func (page *VarsPage) RenderVariables(args []api.Variable, locals []api.Variable, returns []api.Variable) {

	page.locals.ClearChildren()
	page.args.ClearChildren()

	page.AddVars(page.locals, locals)
	page.AddVars(page.args, args)

	page.resizeTrees()
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
				page.resizeTrees()
			})
		}
		parent.AddChild(newNode)
	}
}

func (varsView *VarsPage) GetName() string {
	return "vars"
}

func (page *VarsPage) HandleKeyEvent(event *tcell.EventKey) *tcell.EventKey {

	// If current header doesn't have content, move to one that does.
	if len(page.varHeaders[page.varHeaderIdx].GetRoot().GetChildren()) == 0 {
		for i, t := range page.varHeaders {
			if len(t.GetRoot().GetChildren()) != 0 {
				page.varHeaderIdx = i
				break
			}
		}
	}
	// If moving with TAB/backTAB skip empty headers.
	newI := page.varHeaderIdx
	if keyPressed(event, gConfig.Keys.NextSection) {
		for i, t := range page.varHeaders {
			if i <= page.varHeaderIdx {
				continue
			}
			if len(t.GetRoot().GetChildren()) != 0 {
				newI = i
				break
			}
		}
	} else if keyPressed(event, gConfig.Keys.PrevSection) {
		for i := page.varHeaderIdx - 1; i >= 0; i-- {
			if len(page.varHeaders[i].GetRoot().GetChildren()) != 0 {
				newI = i
				break
			}
		}
	}
	page.varHeaderIdx = newI
	currentTree := page.varHeaders[newI]
	if currentTree.GetCurrentNode() == nil {
		// Focus one child if nothing focused
		if len(currentTree.GetRoot().GetChildren()) != 0 {
			currentTree.SetCurrentNode(currentTree.GetRoot().GetChildren()[0])
		}
	}

	page.varHeaders[page.varHeaderIdx].InputHandler()(event, func(p tview.Primitive) {})
	return nil
}

func (page *VarsPage) SetCommandHandler(ch *CommandHandler) {
	page.commandHandler = ch
}

func (page *VarsPage) GetWidget() tview.Primitive {
	return page.widget
}
