// Package engine wraps the gogpu/ui headless rendering pipeline and the
// EditorView widget into a frame-based offscreen engine. The host sends
// JSON-line commands on stdin and reads JSON-line events on stdout.
package engine

import (
	"github.com/gogpu/gg"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/render"
	"github.com/gogpu/ui/widget"

	"gode/editor"
)

// IDEEngine drives a full IDE layout in headless mode: title bar, sidebar,
// editor area, right panel, input bar, and status bar.
type IDEEngine struct {
	cc       *gg.Context
	canvas   widget.Canvas
	app      *app.App
	layout   *editor.IDELayout
	model    editor.IEditableTextModel
	provider *fixedWindowProvider
	width    int
	height   int
	scale    float32

	// Tab bar state (for backward compatibility)
	tabCanvas     widget.Canvas
	tabCC         *gg.Context
	tabWidth      int
	tabHeight     int
	tabScale      float32
	tabsDirty     bool

	onDidChange func(Range, string)
	OnTabSelected func(idx int)
	OnTabClose    func(idx int)

	// dirtyVersion is incremented whenever the visual state changes.
	dirtyVersion    uint64
	lastSentVersion uint64
}

// NewIDEEngine creates a headless IDE engine with the given initial viewport size.
func NewIDEEngine(width, height int) *IDEEngine {
	cc := gg.NewContext(width, height)
	c := render.NewCanvas(cc, width, height)

	model := editor.NewTextModel("")
	opts := editor.DefaultOptions()
	v := editor.NewEditorView(model, opts)

	// Register CJK font
	if family := registerCJKFont(); family != "" {
		v.SetCJKFontFamily(family)
	}

	provider := &fixedWindowProvider{w: width, h: height}
	uiApp := app.New(app.WithWindowProvider(provider))

	// Create IDE layout
	layout := editor.NewIDELayout()
	layout.SetEditor(v)
	layout.SetTitle("Gode")
	layout.SetSubtitle("main")

	uiApp.SetRoot(layout)

	// Initialize tab bar
	var e *IDEEngine
	tb := editor.NewTabBar()
	tb.OnTabSelected = func(idx int) {
		if e.OnTabSelected != nil {
			e.OnTabSelected(idx)
		}
	}
	tb.OnTabClose = func(idx int) {
		if e.OnTabClose != nil {
			e.OnTabClose(idx)
		}
	}

	e = &IDEEngine{
		cc:       cc,
		canvas:   c,
		app:      uiApp,
		layout:   layout,
		model:    model,
		provider: provider,
		width:    width,
		height:   height,
		scale:    1,
		tabWidth: 800,
		tabHeight: 35,
		tabScale:  1,
	}

	// Set up editor callbacks
	v.OnDidChange = func(r editor.Range, text string) {
		if e.onDidChange != nil {
			e.onDidChange(rangeFromEditor(r), text)
		}
	}

	// Set up sidebar callbacks
	layout.Sidebar.OnItemSelected = func(item *editor.TreeItem) {
		// Handle file selection from sidebar
	}

	// Set up input bar callbacks
	layout.InputBar.OnSubmit = func(text string) {
		// Handle command submission
	}

	// Set up status bar with default values
	layout.SetLanguageStatus("JavaScript")
	layout.SetCursorStatus(1, 1)

	e.focusEditor()
	return e
}

// focusEditor requests focus on the editor view.
func (e *IDEEngine) focusEditor() {
	editorView := e.layout.Editor.(*editor.EditorView)
	x := editorView.VM.TextLeft() + 1
	e.HandleEvent(BuildMouseEvent(InputMouse{MouseType: "press", Button: "left", X: x, Y: 2}))
	e.HandleEvent(BuildMouseEvent(InputMouse{MouseType: "release", Button: "left", X: x, Y: 2}))
}

// SetOnDidChange registers a callback for edit events.
func (e *IDEEngine) SetOnDidChange(fn func(Range, string)) {
	e.onDidChange = fn
}

// Resize changes the viewport size and device pixel scale.
func (e *IDEEngine) Resize(w, h int, scale float32) {
	if scale <= 0 {
		scale = 1
	}
	if w == e.width && h == e.height && scale == e.scale {
		return
	}
	e.width, e.height, e.scale = w, h, scale

	e.provider.updateSize(w, h, float64(scale))

	// Rebuild offscreen render target
	e.cc = gg.NewContext(w, h)
	e.canvas = render.NewCanvas(e.cc, w, h)

	// Apply scale to layout options
	e.applyLayoutScale()
	e.markDirty()
}

// applyLayoutScale applies the current scale to layout dimensions.
func (e *IDEEngine) applyLayoutScale() {
	e.layout.Options.TitleBarHeight = 38 * e.scale
	e.layout.Options.SidebarWidth = 240 * e.scale
	e.layout.Options.RightPanelWidth = 200 * e.scale
	e.layout.Options.InputBarHeight = 44 * e.scale
	e.layout.Options.StatusBarHeight = 24 * e.scale

	// Update editor options
	opts := editor.DefaultOptions()
	opts.FontSize = 14 * e.scale
	opts.LineHeight = 19 * e.scale

	editorView := e.layout.Editor.(*editor.EditorView)
	editorView.Options = opts
	editorView.VM.Options = opts
}

// SetGlyphMarginWidth sets the width reserved for breakpoints.
func (e *IDEEngine) SetGlyphMarginWidth(w float32) {
	if w < 0 {
		w = 0
	}
	editorView := e.layout.Editor.(*editor.EditorView)
	editorView.Options.GlyphMarginWidth = w
	editorView.VM.Options.GlyphMarginWidth = w
	e.markDirty()
}

// SetBreakpoints updates the set of lines with breakpoints.
func (e *IDEEngine) SetBreakpoints(lines []int) {
	bp := make(map[int]bool, len(lines))
	for _, l := range lines {
		if l > 0 {
			bp[l] = true
		}
	}
	editorView := e.layout.Editor.(*editor.EditorView)
	editorView.SetBreakpoints(bp)
	e.markDirty()
}

// Focus focuses the editor view.
func (e *IDEEngine) Focus() {
	ctx := e.app.Window().Context()
	editorView := e.layout.Editor.(*editor.EditorView)
	ctx.RequestFocus(editorView)
	e.app.Frame()
	e.markDirty()
}

// SetTokens replaces syntax-highlighting token spans.
func (e *IDEEngine) SetTokens(lines []TokenLine) {
	spans := make(map[int][]editor.TokenSpan, len(lines))
	for _, tl := range lines {
		ss := make([]editor.TokenSpan, 0, len(tl.Spans))
		for _, s := range tl.Spans {
			ss = append(ss, editor.TokenSpan{
				Start: s.Start,
				End:   s.End,
				Color: parseColor(s.Color, editor.ForegroundColor()),
			})
		}
		spans[tl.Line] = ss
	}
	editorView := e.layout.Editor.(*editor.EditorView)
	editorView.SetTokens(spans)
	e.markDirty()
}

// SetText replaces the entire document.
func (e *IDEEngine) SetText(text string) {
	model := editor.NewTextModel(text)
	e.model = model
	editorView := e.layout.Editor.(*editor.EditorView)
	editorView.Model = model
	editorView.VM.Model = model
	editorView.SetCursor(editor.Position{Line: 1, Column: 1})
	e.markDirty()
}

// GetContent returns the full document text.
func (e *IDEEngine) GetContent() string {
	return e.model.ValueInRange(editor.Range{
		Start: editor.Position{Line: 1, Column: 1},
		End:   editor.Position{Line: e.model.LineCount(), Column: e.model.LineMaxColumn(e.model.LineCount())},
	})
}

// Selection returns the current selection.
func (e *IDEEngine) Selection() (anchor, active Pos) {
	editorView := e.layout.Editor.(*editor.EditorView)
	s := editorView.Selection()
	return posFromEditor(s.Anchor), posFromEditor(s.Active)
}

// SetSelection sets the anchor and active positions.
func (e *IDEEngine) SetSelection(anchor, active Pos) {
	editorView := e.layout.Editor.(*editor.EditorView)
	editorView.SetSelection(posToEditor(anchor), posToEditor(active))
	e.markDirty()
}

// SetCursor updates the cursor position in the status bar.
func (e *IDEEngine) SetCursor(line, column int) {
	e.layout.SetCursorStatus(line, column)
	e.markDirty()
}

// SetLanguage updates the language mode in the status bar.
func (e *IDEEngine) SetLanguage(lang string) {
	e.layout.SetLanguageStatus(lang)
	e.markDirty()
}

// SetTitle updates the title bar text.
func (e *IDEEngine) SetTitle(title string) {
	e.layout.SetTitle(title)
	e.markDirty()
}

// SetSubtitle updates the title bar subtitle.
func (e *IDEEngine) SetSubtitle(subtitle string) {
	e.layout.SetSubtitle(subtitle)
	e.markDirty()
}

// SetSidebarItems replaces the sidebar tree items.
func (e *IDEEngine) SetSidebarItems(items []*editor.TreeItem) {
	e.layout.SetSidebarItems(items)
	e.markDirty()
}

// SetRightPanelItems replaces the right panel file items.
func (e *IDEEngine) SetRightPanelItems(items []*editor.FileItem) {
	e.layout.SetRightPanelItems(items)
	e.markDirty()
}

// HandleEvent dispatches a single gogpu/ui event.
func (e *IDEEngine) HandleEvent(ev event.Event) {
	// Handle Tab key specially
	if ke, ok := ev.(*event.KeyEvent); ok &&
		ke.Key == event.KeyTab && ke.KeyType != event.KeyRelease &&
		!ke.IsShift() && !ke.IsCtrl() && !ke.IsAlt() && !ke.IsSuper() {
		editorView := e.layout.Editor.(*editor.EditorView)
		if editorView.IsFocused() {
			editorView.InsertText("\t")
			e.markDirty()
			return
		}
	}
	e.app.HandleEvent(ev)
	e.markDirty()
}

// NeedsRedraw reports whether the engine has visual changes.
func (e *IDEEngine) NeedsRedraw() bool {
	return e.dirtyVersion != e.lastSentVersion
}

// markDirty increments the visual-state version counter.
func (e *IDEEngine) markDirty() {
	e.dirtyVersion++
}

// Render performs layout + draw and returns the RGBA pixels.
func (e *IDEEngine) Render() ([]byte, bool) {
	e.app.Frame()
	ok := e.app.Window().DrawTo(e.canvas)
	if !ok {
		return nil, false
	}
	data := e.cc.ResizeTarget().Data()
	out := make([]byte, len(data))
	copy(out, data)
	e.lastSentVersion = e.dirtyVersion
	return out, true
}

// ViewportSize returns the current viewport dimensions.
func (e *IDEEngine) ViewportSize() (int, int) {
	return e.width, e.height
}

// --- Tab Bar Methods (backward compatibility) ---

// SetTabs updates the tab bar with new tab information.
func (e *IDEEngine) SetTabs(tabs []TabInfo, activeIdx int) {
	editorTabs := make([]editor.TabInfo, len(tabs))
	for i, t := range tabs {
		editorTabs[i] = editor.TabInfo{
			Label:       t.Label,
			Description: t.Description,
			IconName:    t.IconName,
			IsDirty:     t.IsDirty,
			IsActive:    t.IsActive,
			IsPinned:    t.IsPinned,
		}
	}
	_ = editorTabs // Tab bar is no longer used in IDE layout
	e.tabsDirty = true
}

// SetTabViewport sets the rendering dimensions for the tab bar.
func (e *IDEEngine) SetTabViewport(w, h int, scale float32) {
	if scale <= 0 {
		scale = 1
	}
	e.tabWidth, e.tabHeight, e.tabScale = w, h, scale
	e.tabsDirty = true
}

// RenderTabBar renders the tab bar and returns RGBA pixels.
func (e *IDEEngine) RenderTabBar() ([]byte, bool) {
	if !e.tabsDirty {
		return nil, false
	}
	// Tab bar is integrated into the layout
	e.tabsDirty = false
	return nil, true
}

// HandleTabEvent dispatches a mouse event to the tab bar.
func (e *IDEEngine) HandleTabEvent(me InputMouse) bool {
	// Tab events are handled within the layout
	return false
}

// TabViewportSize returns the current tab bar dimensions.
func (e *IDEEngine) TabViewportSize() (int, int) {
	return e.tabWidth, e.tabHeight
}
