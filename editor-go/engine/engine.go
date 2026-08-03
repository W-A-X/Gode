package engine

import (
	"strconv"
	"strings"

	"github.com/gogpu/gg"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/render"
	"github.com/gogpu/ui/widget"

	"gode/editor"
)

type fixedWindowProvider struct {
	w, h int
	dpr  float64
}

func (p *fixedWindowProvider) Size() (int, int) { return p.w, p.h }
func (p *fixedWindowProvider) ScaleFactor() float64 {
	// Return 1: the framework operates in physical pixels, matching the
	// gg.Context's coordinate system. DPR scaling is applied manually at
	// the boundaries (font metrics, mouse/wheel coordinates).
	return 1
}
func (p *fixedWindowProvider) RequestRedraw() {}

// updateSize changes the viewport size reported to the gogpu/ui framework.
// This must be kept in sync with the canvas dimensions so that widget layout
// coordinates match the pixel buffer. Without this, resize would leave the
// widget tree laid out in the old coordinate system while the canvas is
// rebuilt at the new size, causing character misalignment and wrong sizing.
func (p *fixedWindowProvider) updateSize(w, h int, dpr float64) {
	p.w = w
	p.h = h
	p.dpr = dpr
}

// Engine drives an EditorView in headless mode: it renders into a software
// gg.Context and exposes a frame-based API.
type Engine struct {
	cc       *gg.Context
	canvas   widget.Canvas
	app      *app.App
	view     *editor.EditorView
	model    editor.IEditableTextModel
	provider *fixedWindowProvider
	width    int
	height   int
	scale    float32

	glyphMarginWidth float32

	onDidChange func(Range, string)

	// dirtyVersion is incremented whenever the visual state changes.
	// The main loop compares it to lastSentVersion to decide whether to
	// send another frame over the WebSocket. This avoids pushing full
	// frames for no-op events (modifier keys that don't redraw, etc.).
	dirtyVersion    uint64
	lastSentVersion uint64
}

// New creates a headless engine with the given initial viewport size.
func New(width, height int) *Engine {
	cc := gg.NewContext(width, height)
	c := render.NewCanvas(cc, width, height)

	model := editor.NewTextModel("")
	opts := editor.DefaultOptions()
	v := editor.NewEditorView(model, opts)

	provider := &fixedWindowProvider{w: width, h: height}
	uiApp := app.New(app.WithWindowProvider(provider))
	uiApp.SetRoot(v)

	e := &Engine{
		cc:       cc,
		canvas:   c,
		app:      uiApp,
		view:     v,
		model:    model,
		provider: provider,
		width:    width,
		height:   height,
		scale:    1,
	}
	v.OnDidChange = func(r editor.Range, text string) {
		if e.onDidChange != nil {
			e.onDidChange(rangeFromEditor(r), text)
		}
	}
	e.focusEditor()
	return e
}

// focusEditor requests focus on the editor view. In headless mode the widget
// context only learns about focus through ctx.RequestFocus, which the mouse
// press handler calls, so we synthesize a press inside the text area.
func (e *Engine) focusEditor() {
	x := e.view.VM.TextLeft() + 1
	e.HandleEvent(BuildMouseEvent(InputMouse{MouseType: "press", Button: "left", X: x, Y: 2}))
	e.HandleEvent(BuildMouseEvent(InputMouse{MouseType: "release", Button: "left", X: x, Y: 2}))
}

func rangeFromEditor(r editor.Range) Range {
	return Range{
		Start: posFromEditor(r.Start),
		End:   posFromEditor(r.End),
	}
}

// SetOnDidChange registers a callback for edit events.
func (e *Engine) SetOnDidChange(fn func(Range, string)) {
	e.onDidChange = fn
}

// Resize changes the viewport size and device pixel scale. The next frame
// re-lays out. Font and line metrics scale with the device pixel ratio so the
// rendered pixels stay crisp on HiDPI displays.
//
// Only the render target and font metrics are rebuilt here. The widget tree
// (app/root/view) and all editor state — selection, scroll offset, token
// colors — are preserved across resizes. Previously this method recreated the
// whole app and re-ran focusEditor(), which reset the caret to {1,1} on every
// viewport change and made the editor behave as read-only.
func (e *Engine) Resize(w, h int, scale float32) {
	if scale <= 0 {
		scale = 1
	}
	if w == e.width && h == e.height && scale == e.scale {
		return
	}
	e.width, e.height, e.scale = w, h, scale

	// Keep the window provider in sync so the framework layout coordinates
	// match the canvas pixel buffer. Without this, the widget tree is laid
	// out in the old coordinate system (causing character misalignment and
	// wrong sizing) while the canvas is rebuilt at the new size.
	e.provider.updateSize(w, h, float64(scale))

	// Rebuild the offscreen render target at the new (physical) size.
	e.cc = gg.NewContext(w, h)
	e.canvas = render.NewCanvas(e.cc, w, h)

	// Scale the visual metrics with the device pixel ratio. Glyph margin width
	// is preserved across resizes so the breakpoint gutter keeps its slot.
	e.applyViewOptions()
	e.markDirty()
}

// applyViewOptions applies the current scale and glyph-margin width to the
// editor view's options. It is called by Resize and SetGlyphMarginWidth.
func (e *Engine) applyViewOptions() {
	opts := editor.DefaultOptions()
	opts.FontSize = 14 * e.scale
	opts.LineHeight = 19 * e.scale
	opts.GlyphMarginWidth = e.glyphMarginWidth
	e.view.Options = opts
	e.view.VM.Options = opts
}

// SetGlyphMarginWidth sets the width (in device pixels) reserved for the
// breakpoint/decoration gutter at the left of the editor. The host supplies the
// value so the engine's text area aligns with VS Code's glyph margin.
func (e *Engine) SetGlyphMarginWidth(w float32) {
	if w < 0 {
		w = 0
	}
	e.glyphMarginWidth = w
	e.applyViewOptions()
	e.markDirty()
}

// SetBreakpoints updates the set of 1-based lines that have breakpoints. The
// engine renders a marker in the glyph margin for each line.
func (e *Engine) SetBreakpoints(lines []int) {
	bp := make(map[int]bool, len(lines))
	for _, l := range lines {
		if l > 0 {
			bp[l] = true
		}
	}
	e.view.SetBreakpoints(bp)
	e.markDirty()
}

// Focus focuses the editor view without moving the caret. The host calls this
// when VS Code focuses the editor programmatically (e.g. tab switching) so
// that EditorView.handleKey — which drops keys unless IsFocused() — keeps
// accepting input.
func (e *Engine) Focus() {
	ctx := e.app.Window().Context()
	ctx.RequestFocus(e.view)
	e.app.Frame()
	e.markDirty()
}

// SetTokens replaces the syntax-highlighting token spans for the given lines.
// Colors are parsed from the CSS strings resolved by the host from the VS Code
// theme color map; unparseable colors fall back to the editor foreground.
func (e *Engine) SetTokens(lines []TokenLine) {
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
	e.view.SetTokensForVersion(spans, e.view.ModelVersion())
	e.markDirty()
}

// SetText replaces the entire document while preserving the current selection
// and scroll state when possible. If the new text is shorter than the current
// selection, the cursor is clamped to valid positions.
func (e *Engine) SetText(text string) {
	model := editor.NewTextModel(text)
	e.model = model
	e.view.Model = model
	e.view.VM.Model = model

	// Bump the model version so that any in-flight token updates from the
	// previous file are discarded as stale (fixes highlight/content mismatch
	// on rapid file switches).
	e.view.bumpModelVersion()

	// Preserve scroll position and clamp selection to the new content.
	oldCursor := e.view.Cursor()
	newMaxLine := model.LineCount()
	newMaxCol := model.LineMaxColumn(clampInt(oldCursor.Line, 1, newMaxLine))

	newPos := editor.Position{
		Line:   clampInt(oldCursor.Line, 1, newMaxLine),
		Column: clampInt(oldCursor.Column, 1, newMaxCol),
	}
	e.view.SetCursor(newPos)
	e.markDirty()
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// GetContent returns the full document text.
func (e *Engine) GetContent() string {
	return e.model.ValueInRange(editor.Range{
		Start: editor.Position{Line: 1, Column: 1},
		End:   editor.Position{Line: e.model.LineCount(), Column: e.model.LineMaxColumn(e.model.LineCount())},
	})
}

// Selection returns the current selection.
func (e *Engine) Selection() (anchor, active Pos) {
	s := e.view.Selection()
	return posFromEditor(s.Anchor), posFromEditor(s.Active)
}

// SetSelection sets the anchor and active positions.
func (e *Engine) SetSelection(anchor, active Pos) {
	e.view.SetSelection(posToEditor(anchor), posToEditor(active))
	e.markDirty()
}

// HandleEvent dispatches a single gogpu/ui event. Frame() is intentionally
// omitted here: the caller (Render) always invokes Frame() before DrawTo,
// so running it twice would waste a full layout pass per event.
func (e *Engine) HandleEvent(ev event.Event) {
	// The gogpu/ui focus manager intercepts plain Tab for widget focus
	// navigation and consumes it before the event ever reaches the editor
	// widget, so Tab would never indent. Handle it here instead: insert a
	// tab directly (preserving focus and caret). Modified Tabs (e.g.
	// Shift+Tab) fall through to the framework's focus handling.
	if ke, ok := ev.(*event.KeyEvent); ok &&
		ke.Key == event.KeyTab && ke.KeyType != event.KeyRelease &&
		!ke.IsShift() && !ke.IsCtrl() && !ke.IsAlt() && !ke.IsSuper() &&
		e.view.IsFocused() {
		e.view.InsertText("\t")
		e.markDirty()
		return
	}
	e.app.HandleEvent(ev)
	e.markDirty()
}

// NeedsRedraw reports whether the engine has visual changes that haven't
// yet been flushed to a frame. The main loop uses this to avoid sending
// identical frames over the WebSocket for no-op events.
func (e *Engine) NeedsRedraw() bool {
	return e.dirtyVersion != e.lastSentVersion
}

// markDirty increments the visual-state version counter.
func (e *Engine) markDirty() {
	e.dirtyVersion++
}

// Render performs layout + draw and returns the RGBA pixels.
func (e *Engine) Render() ([]byte, bool) {
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
func (e *Engine) ViewportSize() (int, int) {
	return e.width, e.height
}

// BuildKeyEvent constructs a gogpu/ui KeyEvent from protocol fields.
func BuildKeyEvent(k InputKey) *event.KeyEvent {
	keyType := event.KeyPress
	switch k.KeyType {
	case "release":
		keyType = event.KeyRelease
	case "repeat":
		keyType = event.KeyRepeat
	}
	key := lookupKey(k.Key)
	var r rune
	if len(k.Rune) > 0 {
		r = []rune(k.Rune)[0]
	}
	mods := buildMods(k.Shift, k.Ctrl, k.Alt, k.Super)
	return event.NewKeyEvent(keyType, key, r, mods)
}

// BuildMouseEvent constructs a gogpu/ui MouseEvent from protocol fields.
func BuildMouseEvent(m InputMouse) *event.MouseEvent {
	mouseType := event.MouseMove
	switch m.MouseType {
	case "press":
		mouseType = event.MousePress
	case "release":
		mouseType = event.MouseRelease
	case "drag":
		mouseType = event.MouseDrag
	case "double_click":
		mouseType = event.MouseDoubleClick
	}
	btn := event.ButtonLeft
	var btnState event.ButtonState
	switch m.Button {
	case "right":
		btn = event.ButtonRight
		btnState = event.ButtonStateRight
	case "middle":
		btn = event.ButtonMiddle
		btnState = event.ButtonStateMiddle
	case "":
		btn = event.ButtonNone
	}
	if mouseType == event.MousePress || mouseType == event.MouseDrag {
		switch btn {
		case event.ButtonLeft:
			btnState = event.ButtonStateLeft
		case event.ButtonRight:
			btnState = event.ButtonStateRight
		case event.ButtonMiddle:
			btnState = event.ButtonStateMiddle
		}
	}
	pos := geometry.Pt(m.X, m.Y)
	mods := buildMods(m.Shift, m.Ctrl, m.Alt, m.Super)
	return event.NewMouseEvent(mouseType, btn, btnState, pos, pos, mods)
}

// BuildWheelEvent constructs a gogpu/ui WheelEvent from protocol fields.
func BuildWheelEvent(w InputWheel) *event.WheelEvent {
	mods := buildMods(w.Shift, w.Ctrl, false, false)
	delta := geometry.Pt(w.DX, w.DY)
	zero := geometry.Pt(0, 0)
	return event.NewWheelEvent(delta, zero, zero, mods)
}

func buildMods(shift, ctrl, alt, super bool) event.Modifiers {
	var m event.Modifiers
	if shift {
		m |= event.ModShift
	}
	if ctrl {
		m |= event.ModCtrl
	}
	if alt {
		m |= event.ModAlt
	}
	if super {
		m |= event.ModSuper
	}
	return m
}

// parseColor parses a CSS color string ("#rrggbb", "#rrggbbaa", "rgb(r,g,b)",
// "rgba(r,g,b,a)") into a widget.Color. On any error it returns fallback so
// rendering never breaks on a malformed token color from the host.
func parseColor(s string, fallback widget.Color) widget.Color {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	if s[0] == '#' {
		hex := s[1:]
		switch len(hex) {
		case 6, 8:
			r, errR := strconv.ParseUint(hex[0:2], 16, 8)
			g, errG := strconv.ParseUint(hex[2:4], 16, 8)
			b, errB := strconv.ParseUint(hex[4:6], 16, 8)
			if errR != nil || errG != nil || errB != nil {
				return fallback
			}
			a := uint64(255)
			if len(hex) == 8 {
				av, errA := strconv.ParseUint(hex[6:8], 16, 8)
				if errA != nil {
					return fallback
				}
				a = av
			}
			return widget.RGBA8(uint8(r), uint8(g), uint8(b), uint8(a))
		default:
			return fallback
		}
	}
	if inner, ok := stripFunc(s, "rgba"); ok {
		parts := strings.Split(inner, ",")
		if len(parts) >= 3 {
			r := parseComp(parts[0], 255)
			g := parseComp(parts[1], 255)
			b := parseComp(parts[2], 255)
			a := float32(1)
			if len(parts) >= 4 {
				a = parseAlpha(parts[3])
			}
			return widget.Color{R: float32(r) / 255, G: float32(g) / 255, B: float32(b) / 255, A: a}
		}
		return fallback
	}
	if inner, ok := stripFunc(s, "rgb"); ok {
		parts := strings.Split(inner, ",")
		if len(parts) >= 3 {
			r := parseComp(parts[0], 255)
			g := parseComp(parts[1], 255)
			b := parseComp(parts[2], 255)
			return widget.Color{R: float32(r) / 255, G: float32(g) / 255, B: float32(b) / 255, A: 1}
		}
		return fallback
	}
	return fallback
}

// stripFunc returns the inner substring of "name(...)" or false if s does not
// start with name(.
func stripFunc(s, name string) (string, bool) {
	if !strings.HasPrefix(s, name+"(") || !strings.HasSuffix(s, ")") {
		return "", false
	}
	return s[len(name)+1 : len(s)-1], true
}

func parseComp(s string, def uint8) uint8 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 32)
	if err != nil {
		return def
	}
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	return uint8(v)
}

func parseAlpha(s string) float32 {
	s = strings.TrimSpace(s)
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 1
	}
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return float32(v)
}

func lookupKey(name string) event.Key {
	// Navigation keys.
	switch name {
	case "Up":
		return event.KeyUp
	case "Down":
		return event.KeyDown
	case "Left":
		return event.KeyLeft
	case "Right":
		return event.KeyRight
	case "Home":
		return event.KeyHome
	case "End":
		return event.KeyEnd
	case "PageUp":
		return event.KeyPageUp
	case "PageDown":
		return event.KeyPageDown
	case "Enter":
		return event.KeyEnter
	case "Backspace":
		return event.KeyBackspace
	case "Delete":
		return event.KeyDelete
	case "Tab":
		return event.KeyTab
	case "Escape":
		return event.KeyEscape
	case "Space":
		return event.KeySpace
	}
	// Letters A-Z.
	if len(name) == 1 && name[0] >= 'A' && name[0] <= 'Z' {
		return event.Key(name[0] - 'A' + 1) // KeyA = 1
	}
	// Letters a-z.
	if len(name) == 1 && name[0] >= 'a' && name[0] <= 'z' {
		return event.Key(name[0] - 'a' + 1)
	}
	// Digits 0-9.
	if len(name) == 1 && name[0] >= '0' && name[0] <= '9' {
		return event.Key(name[0] - '0' + 100)
	}
	// F-keys: KeyF1 = 200, KeyF2 = 201, ...
	if len(name) > 1 && name[0] == 'F' {
		n := 0
		for _, ch := range name[1:] {
			if ch >= '0' && ch <= '9' {
				n = n*10 + int(ch-'0')
			}
		}
		if n >= 1 && n <= 24 {
			return event.Key(199 + n)
		}
	}
	return event.KeyUnknown
}
