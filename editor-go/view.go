package editor

import (
	"strings"
	"unicode"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// EditorView is a gogpu/ui widget that renders an ITextModel with VS Code-style
// text layout: line numbers, current-line highlight, selection, caret and
// scrollbars, plus mouse/keyboard navigation.
//
// It is the "view" layer of the editor: it owns no text state of its own —
// everything comes from the ITextModel — so the same widget can be driven by a
// native Go buffer or by a remote model bridged over IPC.
type EditorView struct {
	widget.WidgetBase

	// Model is the text source. Replace this (or the ITextModel impl) to point
	// the renderer at a different backend.
	Model ITextModel

	// OnDidChange is called after any editing operation. The callback should
	// synchronize the edit to the external model (e.g. VS Code mirror).
	OnDidChange func(range_ Range, text string)

	// Options carries the view configuration.
	Options ViewOptions

	// VM is the geometry layer; the view updates its viewport and char width
	// every frame and reads scroll/coordinate conversions from it.
	VM *ViewModel

	// tokens holds the syntax-highlighting spans per 1-based line, supplied by
	// the host from VS Code's TextMate tokenization + theme color map. Lines
	// without entries render in the default foreground color.
	tokens map[int][]TokenSpan

	// breakpoints holds the set of 1-based lines that carry a breakpoint marker,
	// supplied by the host from VS Code's debug breakpoints. The engine renders
	// a filled circle in the glyph margin for each line.
	breakpoints map[int]bool

	selection     Selection
	desiredColumn float32
	dragging      bool

	// lastCanvas caches the canvas from the most recent Draw so input
	// handlers (which run without a canvas) can measure glyph widths when
	// mapping between pixels and columns. The default font is proportional,
	// so real glyph widths differ from the CharWidth grid used for layout;
	// input mapping must use real measurements or the caret overlaps glyphs
	// and Backspace/click delete the visually wrong character.
	lastCanvas widget.Canvas
}

// TokenSpan is a colored, half-open column range [Start, End) on a line.
// Columns are 1-based. It is the editor-package representation of the
// syntax-highlighting spans supplied by the host.
type TokenSpan struct {
	Start int
	End   int
	Color widget.Color
}

// SetTokens merges token spans for the given lines. A line mapped to an empty
// slice is cleared; lines absent from the map keep their existing spans.
func (v *EditorView) SetTokens(spans map[int][]TokenSpan) {
	if v.tokens == nil {
		v.tokens = make(map[int][]TokenSpan)
	}
	for line, ss := range spans {
		if len(ss) == 0 {
			delete(v.tokens, line)
		} else {
			v.tokens[line] = ss
		}
	}
	v.SetNeedsRedraw(true)
}

// SetBreakpoints replaces the set of 1-based lines with a breakpoint marker.
func (v *EditorView) SetBreakpoints(bp map[int]bool) {
	v.breakpoints = bp
	v.SetNeedsRedraw(true)
}

// NewEditorView creates an editor view over the given model and options.
func NewEditorView(m ITextModel, opts ViewOptions) *EditorView {
	v := &EditorView{
		Model:         m,
		Options:       opts,
		VM:            NewViewModel(m, opts),
		selection:     Selection{Anchor: Position{Line: 1, Column: 1}, Active: Position{Line: 1, Column: 1}},
		desiredColumn: 0,
	}
	// The embedded WidgetBase is zero-valued here, which means visible and
	// enabled are both false. That makes IsFocusable() false, so the focus
	// manager never tracks the editor: pressing Tab is then treated as
	// "focus navigation with nothing focusable", and the framework's
	// syncManagerFocusToContext releases the editor's focus — all subsequent
	// keystrokes are silently dropped until the user clicks again. Mark the
	// widget visible+enabled so focus is tracked and preserved.
	v.SetVisible(true)
	v.SetEnabled(true)
	return v
}

// Selection returns the current selection (anchor + cursor).
func (v *EditorView) Selection() Selection { return v.selection }

// Cursor returns the current caret position.
func (v *EditorView) Cursor() Position { return v.selection.Active }

// SetCursor moves the caret (and collapses the selection) to the position.
func (v *EditorView) SetCursor(pos Position) {
	v.selection = Selection{Anchor: pos, Active: pos}
	v.desiredColumn = v.VM.ColumnToX(pos.Line, pos.Column)
	v.VM.RevealPosition(pos)
}

// SetSelection sets anchor and caret independently.
func (v *EditorView) SetSelection(anchor, active Position) {
	v.selection = Selection{Anchor: anchor, Active: active}
	v.desiredColumn = v.VM.ColumnToX(active.Line, active.Column)
	v.VM.RevealPosition(active)
}

// IsFocusable implements widget.Focusable.
func (v *EditorView) IsFocusable() bool { return v.IsVisible() && v.IsEnabled() }

// Layout implements widget.Widget: the editor fills all available space.
func (v *EditorView) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	return c.Biggest()
}

// Children implements widget.Widget. The editor is a leaf widget.
func (v *EditorView) Children() []widget.Widget { return nil }

// --- Drawing -------------------------------------------------------------

// Draw implements widget.Widget.
func (v *EditorView) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := v.Bounds()
	size := bounds.Size()
	if size.Width <= 0 || size.Height <= 0 {
		return
	}

	local := geometry.Rect{Min: geometry.Pt(0, 0), Max: size.ToPoint()}
	canvas.PushClip(local)

	// Cache the canvas for input handlers that run outside Draw and need to
	// measure glyph widths (pixel <-> column mapping).
	v.lastCanvas = canvas

	// Keep the geometry layer in sync with the real font metrics.
	v.VM.SetCharWidth(canvas.MeasureText(" ", v.Options.FontSize, false))
	v.VM.SetViewport(size.Width, size.Height)

	canvas.DrawRect(local, backgroundColor)

	lh := v.VM.LineHeight()
	textLeft := v.VM.TextLeft() + v.Options.PaddingLeft
	scrollTop := v.VM.ScrollTop()
	scrollLeft := v.VM.ScrollLeft()
	contentTop := v.Options.PaddingTop

	first, last := v.VM.VisibleLines()
	cursor := v.selection.Active
	sel := v.selection.Sorted()

	// Current-line highlight (full width, only when the caret has no selection).
	if v.selection.IsEmpty() && cursor.Line >= first && cursor.Line <= last {
		top := v.VM.LineToViewTop(cursor.Line) - scrollTop + contentTop
		canvas.DrawRect(geometry.Rect{Min: geometry.Pt(0, top), Max: geometry.Pt(size.Width, top+lh)}, currentLineColor)
	}

	for line := first; line <= last; line++ {
		lineTop := v.VM.LineToViewTop(line) - scrollTop + contentTop

		// Breakpoint marker in the glyph margin (if enabled and present on this line).
		if v.Options.GlyphMarginWidth > 0 && v.breakpoints[line] {
			cx := v.Options.GlyphMarginWidth / 2
			cy := lineTop + lh/2
			r := lh * 0.18
			if r < 3 {
				r = 3
			}
			canvas.DrawCircle(geometry.Pt(cx, cy), r, breakpointColor)
		}

		// Selection highlight for this line.
		if s, ok := v.selectionRangeOnLine(sel, line); ok {
			x0 := v.renderXForColumn(canvas, line, s.Start.Column) + textLeft - scrollLeft
			x1 := v.renderXForColumn(canvas, line, s.End.Column) + textLeft - scrollLeft
			if x1 > x0 {
				canvas.DrawRect(geometry.Rect{Min: geometry.Pt(x0, lineTop), Max: geometry.Pt(x1, lineTop+lh)}, selectionColor)
			}
		}

		// Line number, right-aligned in the margin (after the glyph margin).
		if v.Options.LineNumbers {
			numColor := lineNumberColor
			if line == cursor.Line {
				numColor = activeLineNumberColor
			}
			num := itoa(line)
			canvas.DrawText(
				num,
				geometry.Rect{
					Min: geometry.Pt(v.Options.GlyphMarginWidth, lineTop),
					Max: geometry.Pt(v.Options.GlyphMarginWidth+v.VM.LineNumbersWidth()-v.Options.PaddingLeft, lineTop+lh),
				},
				v.Options.FontSize,
				numColor,
				false,
				widget.TextAlignRight,
			)
		}

		// Line text, split into colored token spans (tab-aware).
		content := v.Model.LineContent(line)
		if content != "" {
			v.drawLineText(canvas, line, content, lineTop, textLeft-scrollLeft, size, lh)
		}
	}

	// Caret.
	if cursor.Line >= first && cursor.Line <= last {
		cursorColor_ := cursorColor
		if !v.IsFocused() {
			cursorColor_ = cursorUnfocusedColor
		}
		x := v.renderXForColumn(canvas, cursor.Line, cursor.Column) + textLeft - scrollLeft
		caretTop := v.VM.LineToViewTop(cursor.Line) - scrollTop + contentTop
		canvas.DrawRect(
			geometry.Rect{
				Min: geometry.Pt(x, caretTop),
				Max: geometry.Pt(x+v.Options.CursorWidth, caretTop+lh),
			},
			cursorColor_,
		)
	}

	v.drawScrollbars(canvas, size)
	canvas.PopClip()
}

// drawLineText renders a line's text split into colored token spans. It walks
// the runes once, tracking the text-area-relative x (tabs advance to the next
// tab stop via VM.advanceToTabStop, matching ColumnToX), and flushes maximal
// runs of equal color with a single DrawText. Runes not covered by any span
// use the default foreground color; this is what restores VS Code's built-in
// syntax highlighting, which the host resolves from the theme color map.
func (v *EditorView) drawLineText(canvas widget.Canvas, line int, content string, lineTop, baseX float32, size geometry.Size, lh float32) {
	runes := []rune(content)
	spans := v.tokens[line]

	// textX tracks the layout position (CharWidth-based, matching ColumnToX)
	// for cursor/selection alignment and tab stops. renderX tracks the actual
	// on-screen position measured from the real font. The two diverge because
	// the default font is proportional: a run whose glyphs are wider than the
	// space cell would overlap the next run if positions were advanced by
	// CharWidth alone (the exact bug this fixes). Each run is drawn at
	// renderX, then renderX advances by the measured width of that run.
	textX := float32(0)   // layout x, CharWidth-based
	renderX := float32(0) // actual rendering x, measured width-based
	runStartX := float32(0)
	runColor := foregroundColor
	var runBuf []rune

	flush := func() {
		if len(runBuf) == 0 {
			return
		}
		text := string(runBuf)
		canvas.DrawText(
			text,
			geometry.Rect{
				Min: geometry.Pt(baseX+runStartX, lineTop),
				Max: geometry.Pt(size.Width, lineTop+lh),
			},
			v.Options.FontSize,
			runColor,
			false,
			widget.TextAlignLeft,
		)
		// The next run starts where this run actually ended on screen, so
		// adjacent colored spans never overlap or leave gaps.
		renderX = runStartX + canvas.MeasureText(text, v.Options.FontSize, false)
		runBuf = runBuf[:0]
	}

	for i, r := range runes {
		col := i + 1
		color := tokenColorAt(spans, col, foregroundColor)

		if r == '\t' {
			flush()
			textX = v.VM.advanceToTabStop(textX)
			// A tab renders as blank up to the next tab stop; resume measuring
			// from that layout position.
			renderX = textX
			runStartX = renderX
			runColor = color
			continue
		}

		if len(runBuf) == 0 {
			runStartX = renderX
			runColor = color
		} else if color != runColor {
			flush()
			runStartX = renderX
			runColor = color
		}
		runBuf = append(runBuf, r)
		textX += v.VM.CharWidth
	}
	flush()
}

// tokenColorAt returns the color of the token span covering the 1-based
// column, or fallback when no span covers it. Spans are sorted and
// non-overlapping; a span [Start, End) covers Start <= col < End, so spans
// whose End is <= col can no longer contain col and are skipped. Using
// `End < col` here would leave the span ending exactly at the previous
// column in place, making the first letter of the next token (col == that
// span's End) fall back to the default color — the "first letter" highlight
// anomaly.
func tokenColorAt(spans []TokenSpan, col int, fallback widget.Color) widget.Color {
	spanIdx := 0
	for spanIdx < len(spans) && spans[spanIdx].End <= col {
		spanIdx++
	}
	if spanIdx < len(spans) && col >= spans[spanIdx].Start && col < spans[spanIdx].End {
		return spans[spanIdx].Color
	}
	return fallback
}

// selectionRangeOnLine returns the column range of the selection restricted to
// the given 1-based line, and whether it is non-empty there.
func (v *EditorView) selectionRangeOnLine(sel Range, line int) (Range, bool) {
	lineMax := v.Model.LineMaxColumn(line)

	startCol, endCol := 1, lineMax
	if line == sel.Start.Line {
		startCol = sel.Start.Column
	}
	if line == sel.End.Line {
		endCol = sel.End.Column
	} else if line < sel.Start.Line || line > sel.End.Line {
		return Range{}, false
	}
	if startCol >= endCol {
		return Range{}, false
	}
	return Range{Start: Position{Line: line, Column: startCol}, End: Position{Line: line, Column: endCol}}, true
}

func (v *EditorView) drawScrollbars(canvas widget.Canvas, size geometry.Size) {
	const (
		barW      = 8
		barH      = 8
		thumbMin  = 24
		minScroll = 1
	)

	color := scrollbarSliderColor
	if v.dragging {
		color = scrollbarSliderHover
	}

	contentH := v.VM.ContentHeight()
	if contentH > size.Height+minScroll {
		thumbH := size.Height * size.Height / contentH
		if thumbH < thumbMin {
			thumbH = thumbMin
		}
		maxY := size.Height - thumbH
		y := maxY * v.VM.ScrollTop() / (contentH - size.Height)
		canvas.DrawRect(
			geometry.Rect{Min: geometry.Pt(size.Width-barW, y), Max: geometry.Pt(size.Width, y+thumbH)},
			color,
		)
	}

	contentW := v.VM.ContentWidth() + v.VM.TextLeft()
	if contentW > size.Width+minScroll {
		thumbW := size.Width * size.Width / contentW
		if thumbW < thumbMin {
			thumbW = thumbMin
		}
		maxX := size.Width - thumbW
		x := maxX * v.VM.ScrollLeft() / (contentW - size.Width)
		canvas.DrawRect(
			geometry.Rect{Min: geometry.Pt(x, size.Height-barH), Max: geometry.Pt(x+thumbW, size.Height)},
			color,
		)
	}
}

// --- Input ---------------------------------------------------------------

// Event implements widget.Widget.
func (v *EditorView) Event(ctx widget.Context, e event.Event) bool {
	switch ev := e.(type) {
	case *event.MouseEvent:
		return v.handleMouse(ctx, ev)
	case *event.WheelEvent:
		return v.handleWheel(ev)
	case *event.KeyEvent:
		if v.IsFocused() && ev.KeyType != event.KeyRelease {
			return v.handleKey(ev)
		}
	case *event.FocusEvent:
		// Redraw so the caret dims/brightens with focus.
		v.SetNeedsRedraw(true)
		ctx.InvalidateRect(v.Bounds())
		return false
	}
	return false
}

func (v *EditorView) handleMouse(ctx widget.Context, e *event.MouseEvent) bool {
	// GlobalPosition is always window-space; ScreenOrigin is stamped during
	// Draw. This conversion works for both tree dispatch and pointer capture.
	local := v.GlobalToLocal(e.GlobalPosition)

	switch e.MouseType {
	case event.MousePress:
		if e.Button != event.ButtonLeft {
			return false
		}
		ctx.RequestFocus(v)
		v.dragging = true

		// Capture the pointer (ADR-031) so MouseMove events keep arriving while
		// the user drags outside the editor bounds, extending the selection.
		if pc, ok := ctx.(widget.PointerCapturer); ok {
			pc.CapturePointer(v)
		}

		if local.X < v.VM.TextLeft() {
			// Click in the line-number margin: select the whole line.
			line := v.VM.ViewTopToLine(local.Y - v.Options.PaddingTop + v.VM.ScrollTop())
			v.selectLine(line)
		} else {
			pos := v.positionFromPixel(local)
			v.selection = Selection{Anchor: pos, Active: pos}
			v.desiredColumn = v.VM.ColumnToX(pos.Line, pos.Column)
			v.VM.RevealPosition(pos)
		}
		v.SetNeedsRedraw(true)
		ctx.InvalidateRect(v.Bounds())
		return true

	case event.MouseDrag, event.MouseMove:
		// While the left button is held, drags extend the selection. The press
		// handler captured the pointer, so moves outside the bounds also arrive.
		if !v.dragging || !e.Buttons.IsLeftPressed() {
			return false
		}
		pos := v.positionFromPixel(local)
		if pos != v.selection.Active {
			v.selection.Active = pos
			v.VM.RevealPosition(pos)
			v.SetNeedsRedraw(true)
			ctx.InvalidateRect(v.Bounds())
		}
		return true

	case event.MouseRelease:
		if e.Button != event.ButtonLeft {
			return false
		}
		wasDragging := v.dragging
		v.dragging = false
		if pc, ok := ctx.(widget.PointerCapturer); ok {
			pc.ReleasePointer(v)
		}
		v.SetNeedsRedraw(true)
		ctx.InvalidateRect(v.Bounds())
		return wasDragging

	case event.MouseDoubleClick:
		if e.Button != event.ButtonLeft {
			return false
		}
		pos := v.positionFromPixel(local)
		v.selectWord(pos)
		v.SetNeedsRedraw(true)
		ctx.InvalidateRect(v.Bounds())
		return true
	}
	return false
}

func (v *EditorView) handleWheel(e *event.WheelEvent) bool {
	// The host normalizes all wheel deltas to pixels ahead of time — it converts
	// line/page delta modes to pixels and applies VS Code's
	// mouseWheelScrollSensitivity / fastScrollSensitivity multipliers — so scroll
	// directly by pixels. (Previously the engine guessed |delta|<=3 meant lines,
	// which mishandled macOS trackpad pixel deltas and felt unnatural.)
	v.VM.ScrollBy(e.DeltaY(), e.DeltaX())
	v.SetNeedsRedraw(true)
	return true
}

func (v *EditorView) handleKey(e *event.KeyEvent) bool {
	shift := e.IsShift()
	ctrl := e.IsCtrl() || e.IsSuper()

	if e.KeyType == event.KeyRelease {
		return false
	}

	key := e.Key

	// Text input keys: printable rune, Enter, Tab, Backspace, Delete.
	switch key {
	case event.KeyEnter:
		v.insertNewline()
	case event.KeyBackspace:
		v.deleteBackward()
	case event.KeyDelete:
		v.deleteForward()
	case event.KeyTab:
		v.InsertText("\t")
	default:
		// Printable rune (no modifier combo that should not insert text).
		if e.Rune != 0 && !ctrl && !isNonPrintableKey(key) {
			v.InsertText(string(e.Rune))
		} else {
			// Navigation keys.
			switch key {
			case event.KeyUp, event.KeyDown:
				v.moveVertical(key == event.KeyDown, ctrl, shift)
			case event.KeyLeft, event.KeyRight:
				v.moveHorizontal(key == event.KeyRight, ctrl, shift)
			case event.KeyHome:
				if ctrl {
					v.moveToDocumentStart(shift)
				} else {
					v.moveToLineStart(shift)
				}
			case event.KeyEnd:
				if ctrl {
					v.moveToDocumentEnd(shift)
				} else {
					v.moveToLineEnd(shift)
				}
			case event.KeyPageUp, event.KeyPageDown:
				v.movePage(key == event.KeyPageDown, shift)
			default:
				return false
			}
		}
	}

	v.SetNeedsRedraw(true)
	return true
}

// isNonPrintableKey returns true for keys that should not produce text input.
func isNonPrintableKey(key event.Key) bool {
	return key >= event.KeyUp && key <= event.KeyPageDown
}

// --- Editing operations ---------------------------------------------------

func (v *EditorView) replaceRange(r Range, text string) {
	m, ok := v.Model.(IEditableTextModel)
	if !ok {
		return // model is read-only
	}
	sel := r
	if sel.Start.Line > sel.End.Line || (sel.Start.Line == sel.End.Line && sel.Start.Column > sel.End.Column) {
		sel = Range{Start: r.End, End: r.Start}
	}
	m.Edit(sel, text)
	// Move cursor to end of inserted text.
	lines := strings.Split(text, "\n")
	var newPos Position
	if len(lines) == 1 {
		newPos = Position{Line: sel.Start.Line, Column: sel.Start.Column + len([]rune(lines[0]))}
	} else {
		newPos = Position{Line: sel.Start.Line + len(lines) - 1, Column: len([]rune(lines[len(lines)-1])) + 1}
	}
	v.selection = Selection{Anchor: newPos, Active: newPos}
	v.desiredColumn = v.VM.ColumnToX(newPos.Line, newPos.Column)
	v.VM.RevealPosition(newPos)
	v.VM.Model = v.Model // refresh line count
	if v.OnDidChange != nil {
		v.OnDidChange(sel, text)
	}
}

// InsertText inserts text at the current cursor/selection, mirroring what a
// keystroke would do. The engine uses this for Tab, which the framework's
// focus manager intercepts for widget navigation before the editor widget
// ever sees the key event.
func (v *EditorView) InsertText(text string) {
	v.replaceRange(v.selection.Sorted(), text)
}

func (v *EditorView) insertNewline() {
	v.replaceRange(v.selection.Sorted(), "\n")
}

func (v *EditorView) deleteBackward() {
	sel := v.selection
	if sel.IsEmpty() {
		// Delete the character before the cursor.
		cur := sel.Active
		if cur.Line == 1 && cur.Column <= 1 {
			return
		}
		if cur.Column <= 1 {
			// Merge with previous line end.
			prevLine := cur.Line - 1
			prevCol := v.Model.LineMaxColumn(prevLine)
			v.replaceRange(Range{Start: Position{Line: prevLine, Column: prevCol}, End: cur}, "")
			return
		}
		v.replaceRange(Range{Start: Position{Line: cur.Line, Column: cur.Column - 1}, End: cur}, "")
	} else {
		v.replaceRange(sel.Sorted(), "")
	}
}

func (v *EditorView) deleteForward() {
	sel := v.selection
	if sel.IsEmpty() {
		cur := sel.Active
		if cur.Column >= v.Model.LineMaxColumn(cur.Line) && cur.Line >= v.Model.LineCount() {
			return
		}
		if cur.Column >= v.Model.LineMaxColumn(cur.Line) {
			// Delete start of next line (merge).
			v.replaceRange(Range{Start: cur, End: Position{Line: cur.Line + 1, Column: 1}}, "")
			return
		}
		v.replaceRange(Range{Start: cur, End: Position{Line: cur.Line, Column: cur.Column + 1}}, "")
	} else {
		v.replaceRange(sel.Sorted(), "")
	}
}

// --- Cursor movement ------------------------------------------------------

func (v *EditorView) moveWithShift(newPos Position, shift bool) {
	if shift {
		v.selection.Active = newPos
	} else {
		v.selection = Selection{Anchor: newPos, Active: newPos}
	}
	v.VM.RevealPosition(newPos)
}

// moveVertical preserves the desired column: a vertical move keeps the visual
// column (clamped to each line's width), matching VS Code.
func (v *EditorView) moveVertical(down, ctrl, shift bool) {
	cur := v.selection.Active
	if ctrl {
		// Ctrl/Cmd+Up/Down: jump to document start/end.
		if down {
			v.moveToDocumentEnd(shift)
		} else {
			v.moveToDocumentStart(shift)
		}
		return
	}

	target := cur.Line
	if down {
		target++
	} else {
		target--
	}
	target = clamp(target, 1, v.Model.LineCount())

	col := v.VM.XToColumn(target, v.desiredColumn)
	if col > v.Model.LineMaxColumn(target) {
		col = v.Model.LineMaxColumn(target)
	}
	v.moveWithShift(Position{Line: target, Column: col}, shift)
}

func (v *EditorView) moveHorizontal(right, ctrl, shift bool) {
	cur := v.selection.Active
	var newPos Position

	if ctrl {
		// Ctrl/Cmd+Left/Right: move by word.
		line := v.Model.LineContent(cur.Line)
		if right {
			newPos = Position{Line: cur.Line, Column: nextWordEnd(line, cur.Column)}
		} else {
			newPos = Position{Line: cur.Line, Column: prevWordStart(line, cur.Column)}
		}
	} else {
		maxCol := v.Model.LineMaxColumn(cur.Line)
		if right {
			col := cur.Column + 1
			if col > maxCol {
				// Wrap to the next line, matching VS Code's line-boundary behavior.
				if cur.Line < v.Model.LineCount() {
					newPos = Position{Line: cur.Line + 1, Column: 1}
				} else {
					newPos = Position{Line: cur.Line, Column: maxCol}
				}
			} else {
				newPos = Position{Line: cur.Line, Column: col}
			}
		} else {
			col := cur.Column - 1
			if col < 1 {
				if cur.Line > 1 {
					newPos = Position{Line: cur.Line - 1, Column: v.Model.LineMaxColumn(cur.Line - 1)}
				} else {
					newPos = Position{Line: cur.Line, Column: 1}
				}
			} else {
				newPos = Position{Line: cur.Line, Column: col}
			}
		}
	}

	v.desiredColumn = v.VM.ColumnToX(newPos.Line, newPos.Column)
	v.moveWithShift(newPos, shift)
}

func (v *EditorView) moveToLineStart(shift bool) {
	pos := Position{Line: v.selection.Active.Line, Column: 1}
	v.desiredColumn = 0
	v.moveWithShift(pos, shift)
}

func (v *EditorView) moveToLineEnd(shift bool) {
	pos := Position{Line: v.selection.Active.Line, Column: v.Model.LineMaxColumn(v.selection.Active.Line)}
	v.desiredColumn = v.VM.ColumnToX(pos.Line, pos.Column)
	v.moveWithShift(pos, shift)
}

func (v *EditorView) moveToDocumentStart(shift bool) {
	v.desiredColumn = 0
	v.moveWithShift(Position{Line: 1, Column: 1}, shift)
}

func (v *EditorView) moveToDocumentEnd(shift bool) {
	last := v.Model.LineCount()
	pos := Position{Line: last, Column: v.Model.LineMaxColumn(last)}
	v.desiredColumn = v.VM.ColumnToX(pos.Line, pos.Column)
	v.moveWithShift(pos, shift)
}

func (v *EditorView) movePage(down, shift bool) {
	page := v.VM.ViewportLines() - 1
	if page < 1 {
		page = 1
	}
	cur := v.selection.Active
	target := cur.Line
	if down {
		target += page
	} else {
		target -= page
	}
	target = clamp(target, 1, v.Model.LineCount())
	col := v.VM.XToColumn(target, v.desiredColumn)
	if col > v.Model.LineMaxColumn(target) {
		col = v.Model.LineMaxColumn(target)
	}
	v.moveWithShift(Position{Line: target, Column: col}, shift)
}

// --- Mouse position mapping ----------------------------------------------

// positionFromPixel maps a widget-local pixel to a model position. The
// column uses real glyph measurements (via xToColumnReal) so clicks land on
// the character the user actually points at — the layout CharWidth grid
// diverges because the default font is proportional.
func (v *EditorView) positionFromPixel(p geometry.Point) Position {
	y := p.Y - v.Options.PaddingTop + v.VM.ScrollTop()
	line := v.VM.ViewTopToLine(y)
	x := p.X - v.VM.TextLeft() - v.Options.PaddingLeft + v.VM.ScrollLeft()
	col := v.xToColumnReal(line, x)
	return Position{Line: line, Column: col}
}

// xToColumnReal converts a pixel x inside the text area (the same origin as
// renderXForColumn) back to a 1-based column, using real glyph widths so it
// matches where the text is actually drawn. Falls back to the layout grid
// before the first Draw (no canvas cached yet).
func (v *EditorView) xToColumnReal(line int, x float32) int {
	c := v.lastCanvas
	if c == nil {
		return v.VM.XToColumn(line, x)
	}
	content := []rune(v.Model.LineContent(line))
	col := 1
	cur := float32(0)
	for _, ch := range content {
		var w float32
		if ch == '\t' {
			w = v.VM.advanceToTabStop(cur) - cur
		} else {
			w = c.MeasureText(string(ch), v.Options.FontSize, false)
		}
		if x < cur+w/2 {
			return col
		}
		if x < cur+w {
			return col + 1
		}
		cur += w
		col++
	}
	return len(content) + 1
}

// renderXForColumn returns the true on-screen x (relative to the text area,
// before textLeft/scrollLeft) of the caret before the 1-based column,
// measured with the real font. Tabs advance to tab stops. This matches
// drawLineText's renderX, so the caret/selection sit exactly on the rendered
// glyphs — the CharWidth grid diverges because the default font is
// proportional (this was the "caret overlapping characters" bug).
func (v *EditorView) renderXForColumn(canvas widget.Canvas, line, col int) float32 {
	content := []rune(v.Model.LineContent(line))
	n := clamp(col-1, 0, len(content))
	x := float32(0)
	for _, ch := range content[:n] {
		if ch == '\t' {
			x = v.VM.advanceToTabStop(x)
		} else {
			x += canvas.MeasureText(string(ch), v.Options.FontSize, false)
		}
	}
	return x
}

func (v *EditorView) selectLine(line int) {
	if line < 1 {
		line = 1
	}
	if line > v.Model.LineCount() {
		line = v.Model.LineCount()
	}
	v.selection = Selection{
		Anchor: Position{Line: line, Column: 1},
		Active: Position{Line: line, Column: v.Model.LineMaxColumn(line)},
	}
	v.desiredColumn = 0
	v.VM.RevealPosition(v.selection.Active)
}

func (v *EditorView) selectWord(pos Position) {
	content := []rune(v.Model.LineContent(pos.Line))
	if len(content) == 0 {
		v.selection = Selection{Anchor: pos, Active: pos}
		return
	}

	start := pos.Column - 1
	end := pos.Column - 1
	// Expand left while on a word character (or a run of non-space, non-word chars).
	wordChar := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }
	for start > 0 && wordChar(content[start-1]) {
		start--
	}
	for end < len(content) && wordChar(content[end]) {
		end++
	}
	v.selection = Selection{
		Anchor: Position{Line: pos.Line, Column: start + 1},
		Active: Position{Line: pos.Line, Column: end + 1},
	}
	v.desiredColumn = v.VM.ColumnToX(pos.Line, end+1)
}

// --- Word boundaries ------------------------------------------------------

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// nextWordEnd returns the column just past the next word to the right.
func nextWordEnd(line string, col int) int {
	content := []rune(line)
	i := clamp(col-1, 0, len(content))
	for i < len(content) && !isWordChar(content[i]) {
		i++
	}
	for i < len(content) && isWordChar(content[i]) {
		i++
	}
	return i + 1
}

// prevWordStart returns the column of the start of the word to the left.
func prevWordStart(line string, col int) int {
	content := []rune(line)
	i := clamp(col-1, 0, len(content))
	if i > len(content) {
		i = len(content)
	}
	for i > 0 && !isWordChar(content[i-1]) {
		i--
	}
	for i > 0 && isWordChar(content[i-1]) {
		i--
	}
	return i + 1
}

// --- Helpers --------------------------------------------------------------

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func abs32(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}
