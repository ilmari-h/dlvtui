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
	locals         *tview.TreeNode
	args           *tview.TreeNode
	returns        *tview.TreeNode

	varHeaders []*tview.TreeView
	varHeaderIdx int

	expandedCache map[uint64]bool
}

func NewVarPage() *VarsPage {
	localsHeader := tview.NewTreeNode("locals").SetColor(tcell.ColorGreen).SetSelectable(true)
	localsHeader.SetSelectable(false)
	localsTree := tview.NewTreeView().SetRoot(localsHeader)
	localsTree.SetCurrentNode(localsHeader)
	localsTree.SetBackgroundColor(tcell.ColorDefault)

	argsHeader := tview.NewTreeNode("arguments").SetColor(tcell.ColorGreen).SetSelectable(true)
	argsHeader.SetSelectable(false)
	argsTree := tview.NewTreeView().SetRoot(argsHeader)
	argsTree.SetCurrentNode(argsHeader)
	argsTree.SetBackgroundColor(tcell.ColorDefault)

	returnsHeader := tview.NewTreeNode("return values").SetColor(tcell.ColorGreen).SetSelectable(true)
	returnsHeader.SetSelectable(false)
	returnsTree := tview.NewTreeView().SetRoot(returnsHeader)
	returnsTree.SetCurrentNode(returnsHeader)
	returnsTree.SetBackgroundColor(tcell.ColorDefault)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(localsTree, 0, 1, false).
		AddItem(argsTree, 0, 1, false).
		AddItem(returnsTree, 0, 1, false)

	pageFrame := tview.NewFrame(flex).
		SetBorders(0,0,0,0,0,0).
		AddText("[::b]Current stack frame:", true, tview.AlignLeft, tcell.ColorWhite)
	pageFrame.SetBackgroundColor(tcell.ColorDefault)

	return &VarsPage{
		widget: pageFrame,
		locals:        localsHeader,
		args:          argsHeader,
		returns: 	   returnsHeader,
		varHeaders: []*tview.TreeView{localsTree,argsTree,returnsTree},
		varHeaderIdx: 0,
		expandedCache: make(map[uint64]bool),
	}
}

func (page *VarsPage) RenderVariables(args []api.Variable, locals []api.Variable, returns []api.Variable) {

	page.locals.ClearChildren()
	page.args.ClearChildren()

	page.AddVars(page.locals, locals)
	page.AddVars(page.args, args)
}

func getVarTitle(vr *api.Variable, expanded bool) string {
	namestr := fmt.Sprintf("[green::b]%s", vr.Name)
	typestr := fmt.Sprintf("[purple]<%s>[white:-:-]", vr.RealType)
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

func (page *VarsPage) AddVars(parent *tview.TreeNode, vars []api.Variable) {
	for vi := range vars {
		vr := vars[vi]
		newNode := tview.NewTreeNode(getVarTitle(&vr, page.expandedCache[vr.Addr])).
			SetReference(vr)
		newNode.SetSelectable(true)
		newNode.SetColor(tcell.ColorBlack)

		// If node has children, initially collapse. Expand on select.
		if vr.Children != nil && len(vr.Children) > 0 {
			page.AddVars(newNode, vr.Children)
			if !page.expandedCache[vr.Addr] {
				newNode.CollapseAll()
			}
			newNode.SetSelectedFunc(func() {
				page.expandedCache[vr.Addr] = !newNode.IsExpanded()
				r := newNode.GetReference().(api.Variable)
				if !newNode.IsExpanded() {
					newNode.Expand()
				} else {
					newNode.Collapse()
				}
				newNode.SetText(getVarTitle(&r, page.expandedCache[vr.Addr]))
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
	if len( page.varHeaders[page.varHeaderIdx].GetRoot().GetChildren() ) == 0 {
		for i, t := range page.varHeaders {
			if len( t.GetRoot().GetChildren() ) != 0 {
				page.varHeaderIdx = i
				break
			}
		}
	}
	// If moving with TAB/backTAB skip empty headers.
	newI := page.varHeaderIdx
	if event.Key() == tcell.KeyTAB {
		for i, t := range page.varHeaders {
			if i <= page.varHeaderIdx {
				continue
			}
			if len( t.GetRoot().GetChildren() ) != 0 {
				newI = i
				break
			}
		}
	} else if event.Key() == tcell.KeyBacktab {
		for i := page.varHeaderIdx-1; i >=0; i-- {
			if len( page.varHeaders[i].GetRoot().GetChildren() ) != 0 {
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
