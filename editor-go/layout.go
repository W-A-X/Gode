package editor

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// IDELayout is a gogpu/ui widget that arranges the full IDE layout:
// title bar, left sidebar, main editor area, right panel, status bar, and input bar.
//
// Layout structure:
// ┌─────────────────────────────────────────────────────────┐
// │ TitleBar                                                 │
// ├──────────┬──────────────────────────────┬───────────────┤
// │ Sidebar  │ Editor Area                  │ RightPanel    │
// │          │                              │               │
// │          │                              │               │
// │          │                              │               │
// ├──────────┴──────────────────────────────┴───────────────┤
// │ InputBar                                                │
// ├─────────────────────────────────────────────────────────┤
// │ StatusBar                                               │
// └─────────────────────────────────────────────────────────┘
type IDELayout struct {
	widget.WidgetBase

	// TitleBar is the top title bar widget.
	TitleBar *TitleBar

	// Sidebar is the left sidebar (project tree).
	Sidebar *Sidebar

	// Editor is the main editor view (or split editor).
	Editor widget.Widget

	// RightPanel is the right file explorer panel.
	RightPanel *RightPanel

	// InputBar is the bottom input/command bar.
	InputBar *InputBar

	// StatusBar is the bottom status bar.
	StatusBar *StatusBar

	// Options carries layout configuration.
	Options IDELayoutOptions

	// layoutCache stores computed rectangles for child widgets.
	layoutCache layoutRects
}

// IDELayoutOptions configures the layout dimensions.
type IDELayoutOptions struct {
	TitleBarHeight float32
	SidebarWidth   float32
	RightPanelWidth float32
	InputBarHeight float32
	StatusBarHeight float32
}

// DefaultIDELayoutOptions returns sensible defaults matching VS Code.
func DefaultIDELayoutOptions() IDELayoutOptions {
	return IDELayoutOptions{
		TitleBarHeight:  38,
		SidebarWidth:    240,
		RightPanelWidth: 200,
		InputBarHeight:  44,
		StatusBarHeight: 24,
	}
}

// layoutRects stores computed child widget rectangles.
type layoutRects struct {
	titleBar   geometry.Rect
	sidebar    geometry.Rect
	editor     geometry.Rect
	rightPanel geometry.Rect
	inputBar   geometry.Rect
	statusBar  geometry.Rect
}

// NewIDELayout creates a new IDE layout with default child widgets.
func NewIDELayout() *IDELayout {
	l := &IDELayout{
		TitleBar:   NewTitleBar(),
		Sidebar:    NewSidebar(),
		RightPanel: NewRightPanel(),
		InputBar:   NewInputBar(),
		StatusBar:  NewStatusBar(),
		Options:    DefaultIDELayoutOptions(),
	}
	return l
}

// SetEditor sets the main editor widget.
func (l *IDELayout) SetEditor(editor widget.Widget) {
	l.Editor = editor
}

// Layout implements widget.Widget. It computes the layout of all children.
func (l *IDELayout) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	size := c.Biggest()

	// Title bar spans full width at top
	titleH := l.Options.TitleBarHeight
	l.layoutCache.titleBar = geometry.Rect{
		Min: geometry.Pt(0, 0),
		Max: geometry.Pt(size.Width, titleH),
	}

	// Status bar spans full width at bottom
	statusH := l.Options.StatusBarHeight
	l.layoutCache.statusBar = geometry.Rect{
		Min: geometry.Pt(0, size.Height-statusH),
		Max: geometry.Pt(size.Width, size.Height),
	}

	// Input bar above status bar
	inputH := l.Options.InputBarHeight
	l.layoutCache.inputBar = geometry.Rect{
		Min: geometry.Pt(0, size.Height-statusH-inputH),
		Max: geometry.Pt(size.Width, size.Height-statusH),
	}

	// Content area between title bar and input bar
	contentTop := titleH
	contentBottom := size.Height - statusH - inputH

	// Left sidebar
	sidebarW := l.Options.SidebarWidth
	l.layoutCache.sidebar = geometry.Rect{
		Min: geometry.Pt(0, contentTop),
		Max: geometry.Pt(sidebarW, contentBottom),
	}

	// Right panel
	rightW := l.Options.RightPanelWidth
	l.layoutCache.rightPanel = geometry.Rect{
		Min: geometry.Pt(size.Width-rightW, contentTop),
		Max: geometry.Pt(size.Width, contentBottom),
	}

	// Editor fills remaining space
	l.layoutCache.editor = geometry.Rect{
		Min: geometry.Pt(sidebarW, contentTop),
		Max: geometry.Pt(size.Width-rightW, contentBottom),
	}

	// Set bounds for child widgets
	l.TitleBar.SetBounds(l.layoutCache.titleBar)
	l.Sidebar.SetBounds(l.layoutCache.sidebar)
	l.RightPanel.SetBounds(l.layoutCache.rightPanel)
	l.InputBar.SetBounds(l.layoutCache.inputBar)
	l.StatusBar.SetBounds(l.layoutCache.statusBar)

	if l.Editor != nil {
		if be, ok := l.Editor.(interface{ SetBounds(geometry.Rect) }); ok {
			be.SetBounds(l.layoutCache.editor)
		}
	}

	return size
}

// Children implements widget.Widget.
func (l *IDELayout) Children() []widget.Widget {
	children := []widget.Widget{
		l.TitleBar,
		l.Sidebar,
		l.RightPanel,
		l.InputBar,
		l.StatusBar,
	}
	if l.Editor != nil {
		children = append(children, l.Editor)
	}
	return children
}

// Draw implements widget.Widget.
func (l *IDELayout) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := l.Bounds()
	size := bounds.Size()
	if size.Width <= 0 || size.Height <= 0 {
		return
	}

	// Background
	canvas.DrawRect(geometry.Rect{Min: geometry.Pt(0, 0), Max: size.ToPoint()}, IDELayoutBackgroundColor)

	// Children draw themselves via the widget tree dispatch
}

// Event implements widget.Widget.
func (l *IDELayout) Event(ctx widget.Context, e event.Event) bool {
	// Events are dispatched to children via the widget tree
	return false
}

// SetTitle sets the title bar text.
func (l *IDELayout) SetTitle(title string) {
	l.TitleBar.Title = title
	l.TitleBar.SetNeedsRedraw(true)
}

// SetSubtitle sets the title bar subtitle (e.g., branch name).
func (l *IDELayout) SetSubtitle(subtitle string) {
	l.TitleBar.Subtitle = subtitle
	l.TitleBar.SetNeedsRedraw(true)
}

// SetSidebarItems replaces the sidebar tree items.
func (l *IDELayout) SetSidebarItems(items []*TreeItem) {
	l.Sidebar.SetItems(items)
}

// SetRightPanelItems replaces the right panel file items.
func (l *IDELayout) SetRightPanelItems(items []*FileItem) {
	l.RightPanel.SetItems(items)
}

// SetCursorStatus updates the cursor position in the status bar.
func (l *IDELayout) SetCursorStatus(line, column int) {
	l.StatusBar.SetCursor(line, column)
}

// SetLanguageStatus updates the language mode in the status bar.
func (l *IDELayout) SetLanguageStatus(lang string) {
	l.StatusBar.SetLanguage(lang)
}

// IDELayout Colors
var (
	IDELayoutBackgroundColor = widget.RGBA8(0x1E, 0x1F, 0x22, 0xFF) // Gray1
)
