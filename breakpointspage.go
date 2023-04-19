package main

import (
	"github.com/ilmari-h/dlvtui/nav"
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type BreakpointsPage struct {
	preRenderSelection *nav.UiBreakpoint // Id of selected breakpoint before rerender.
	commandHandler     *CommandHandler
	treeView           *tview.TreeView
	widget             *tview.Frame
	fileList           map[string]*tview.TreeNode
}

func NewBreakpointsPage() *BreakpointsPage {
	root := tview.NewTreeNode(".").
		SetColor(tcell.ColorDefault)

	treeView := tview.NewTreeView().
		SetRoot(root)

	treeView.SetBackgroundColor(tcell.ColorDefault)
	treeView.SetInputCapture(listInputCaptureC)

	pageFrame := tview.NewFrame(treeView).
		SetBorders(0, 0, 0, 0, 0, 0).
		AddText(fmt.Sprintf("[%s::b]Breakpoints:", iToColorS(gConfig.Colors.HeaderFg)),
			true,
			tview.AlignLeft,
			tcell.ColorWhite,
		)
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

func (page *BreakpointsPage) RenderBreakpoints(bps []*nav.UiBreakpoint) {
	sort.SliceStable(bps, func(i, j int) bool {
		return bps[i].Line < bps[j].Line || bps[i].File < bps[j].File
	})
	page.fileList = make(map[string]*tview.TreeNode)
	rootNode := page.treeView.GetRoot()
	rootNode.ClearChildren()

	var lastSelectedNode *tview.TreeNode = nil

	for _, bp := range bps {
		if bp.ID < 0 {
			continue
		}
		fileNode, ok := page.fileList[bp.File]
		if !ok {
			fileNode = tview.NewTreeNode(fmt.Sprintf("[%s::b]%s",
				iToColorS(gConfig.Colors.ListHeaderFg),
				bp.File,
			)).
				SetSelectable(true)
			rootNode.AddChild(fileNode)
			page.fileList[bp.File] = fileNode
			fileNode.SetSelectedFunc(func() {
				fileNode.SetExpanded(!fileNode.IsExpanded())
			})
			fileNode.SetColor(tcell.ColorBlack)
		}

		bpNode := tview.NewTreeNode(fmt.Sprintf("[%s]%s  [%s]%s[%s]:%d",
			iToColorS(gConfig.Colors.BpFg),
			gConfig.Icons.Bp,
			iToColorS(gConfig.Colors.VarNameFg),
			bp.FunctionName,
			iToColorS(gConfig.Colors.LineFg),
			bp.Line,
		)).
			SetSelectable(true)

		if page.preRenderSelection != nil && bp.Line == page.preRenderSelection.Line && bp.File == page.preRenderSelection.File {
			lastSelectedNode = bpNode
		}

		bpNode.SetColor(tcell.ColorBlack)

		current := bp.Line == page.commandHandler.view.navState.CurrentDebuggerPos.Line &&
			bp.File == page.commandHandler.view.navState.CurrentDebuggerPos.File
		if current {
			bpNode.SetText(fmt.Sprintf("[%s]%s  [%s::b]%s[%s]:%d",
				iToColorS(gConfig.Colors.BpActiveFg),
				gConfig.Icons.BpActive,
				iToColorS(gConfig.Colors.VarTypeFg),
				bp.FunctionName,
				iToColorS(gConfig.Colors.LineFg),
				bp.Line,
			))
		} else if bp.Disabled {
			bpNode.SetText(fmt.Sprintf("[%s]%s  [%s]%s[%s]:%d",
				iToColorS(gConfig.Colors.BpFg),
				gConfig.Icons.BpDisabled,
				iToColorS(gConfig.Colors.VarNameFg),
				bp.FunctionName,
				iToColorS(gConfig.Colors.LineFg),
				bp.Line,
			))
		}

		bpNode.SetReference(bp)
		bpNode.SetSelectable(true)
		bpNode.SetSelectedFunc(func() {
			ref := bpNode.GetReference().(*nav.UiBreakpoint)
			page.commandHandler.RunCommand(&OpenFile{
				File:   ref.File,
				AtLine: ref.Line - 1,
			})
		})
		fileNode.AddChild(bpNode)
	}
	if lastSelectedNode != nil {
		page.treeView.SetCurrentNode(lastSelectedNode)
	}
}

func (page *BreakpointsPage) GetWidget() tview.Primitive {
	return page.widget
}

func (page *BreakpointsPage) GetName() string {
	return "breakpoints"
}

func (page *BreakpointsPage) HandleKeyEvent(event *tcell.EventKey) *tcell.EventKey {
	if keyPressed(event, gConfig.Keys.ClearBreakpoint) {
		selectedNode := page.treeView.GetCurrentNode()
		if selectedNode.GetReference() == nil {
			return nil
		}
		selectedBp := selectedNode.GetReference().(*nav.UiBreakpoint)
		page.fileList[selectedBp.File].RemoveChild(selectedNode)

		if selectedBp.Disabled {
			page.commandHandler.RunCommand(&ClearBreakpoint{selectedBp, false, selectedBp})
		} else {
			page.commandHandler.RunCommand(&ClearBreakpoint{selectedBp, false, nil})
		}
		page.treeView.SetCurrentNode(page.fileList[selectedBp.File])
		return nil
	} else if keyPressed(event, gConfig.Keys.ToggleBreakpoint) {
		selectedNode := page.treeView.GetCurrentNode()
		if selectedNode.GetReference() == nil {
			return nil
		}
		selectedBp := selectedNode.GetReference().(*nav.UiBreakpoint)
		page.preRenderSelection = selectedBp
		if !selectedBp.Disabled {
			page.commandHandler.RunCommand(&ClearBreakpoint{selectedBp, true, nil})
		} else {
			page.commandHandler.RunCommand(&CreateBreakpoint{selectedBp.Line, selectedBp.File})
		}
	}
	page.treeView.InputHandler()(event, func(p tview.Primitive) {})
	return nil
}
