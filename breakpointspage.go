package main

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/go-delve/delve/service/api"
	"github.com/rivo/tview"
)

// TODO: on select just call "switchgoroutine"
// in addition, map debugger state behind current goroutine ID

type BreakpointsPage struct {
	commandHandler *CommandHandler
	treeView       *tview.TreeView
	widget         *tview.Frame
	fileList       map[string]*tview.TreeNode
}

func NewBreakpointsPage() *BreakpointsPage {
	root := tview.NewTreeNode(".").
		SetColor(tcell.ColorGreen)

	treeView := tview.NewTreeView().
		SetRoot(root)

	treeView.SetBackgroundColor(tcell.ColorDefault)

	pageFrame := tview.NewFrame(treeView).
		SetBorders(0, 0, 0, 0, 0, 0).
		AddText("[::b]Breakpoints:", true, tview.AlignLeft, tcell.ColorWhite)
	pageFrame.SetBackgroundColor(tcell.ColorDefault)
	treeView.SetCurrentNode(root)
	bp := BreakpointsPage{
		treeView: treeView,
		widget:   pageFrame,
	}
	return &bp
}

func (page *BreakpointsPage) SetCommandHandler(ch *CommandHandler) {
	page.commandHandler = ch
}

func (page *BreakpointsPage) RenderBreakpoints(bps []*api.Breakpoint) {
	sort.SliceStable(bps, func(i, j int) bool {
		return bps[i].Line < bps[j].Line || bps[i].File < bps[j].File
	})
	page.fileList = make(map[string]*tview.TreeNode)
	rootNode := page.treeView.GetRoot()
	rootNode.ClearChildren()
	for _, bp := range bps {
		if bp.ID < 0 {
			continue
		}

		fileNode, ok := page.fileList[bp.File]
		if !ok {
			fileNode = tview.NewTreeNode(fmt.Sprintf("[green::b]%s", bp.File)).
				SetSelectable(true)
			rootNode.AddChild(fileNode)
			page.fileList[bp.File] = fileNode
			fileNode.SetSelectedFunc(func() {
				fileNode.SetExpanded(!fileNode.IsExpanded())
			})
			fileNode.SetColor(tcell.ColorBlack)
		}

		bpNode := tview.NewTreeNode(fmt.Sprintf("[green]%s[white]:%d", bp.FunctionName, bp.Line)).
			SetSelectable(true)
		bpNode.SetColor(tcell.ColorBlack)

		bpNode.SetReference(bp)
		bpNode.SetSelectable(true)
		bpNode.SetSelectedFunc(func() {
			ref := bpNode.GetReference().(*api.Breakpoint)
			page.commandHandler.RunCommand(&OpenFile{
				File:   ref.File,
				AtLine: ref.Line - 1,
			})
		})
		fileNode.AddChild(bpNode)
	}
}

func (page *BreakpointsPage) GetWidget() tview.Primitive {
	return page.widget
}

func (page *BreakpointsPage) GetName() string {
	return "breakpoints"
}

func (page *BreakpointsPage) HandleKeyEvent(event *tcell.EventKey) *tcell.EventKey {
	rune := event.Rune()
	if rune == 'd' {
		selectedNode := page.treeView.GetCurrentNode()
		selectedBp := selectedNode.GetReference().(*api.Breakpoint)
		if selectedBp != nil {
			page.fileList[selectedBp.File].RemoveChild(selectedNode)
			page.commandHandler.RunCommand(&ClearBreakpoint{Breakpoint: selectedBp})
			page.treeView.SetCurrentNode(page.fileList[selectedBp.File])
		}
		return nil
	}
	page.treeView.InputHandler()(event, func(p tview.Primitive) {})
	return nil
}
