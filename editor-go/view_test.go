package editor

import (
	"testing"

	"github.com/gogpu/gg"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/render"
	"github.com/gogpu/ui/widget"
)

func newTestView(text string) *EditorView {
	m := NewTextModel(text)
	opts := DefaultOptions()
	v := NewEditorView(m, opts)
	v.VM.SetCharWidth(7)
	v.VM.SetViewport(400, 190)
	return v
}

// newRealCanvasView builds a view whose lastCanvas is a real offscreen canvas,
// so pixel <-> column mapping uses true glyph measurements (proportional font)
// instead of the CharWidth layout grid.
func newRealCanvasView(text string) *EditorView {
	v := newTestView(text)
	cc := gg.NewContext(400, 190)
	v.lastCanvas = render.NewCanvas(cc, 400, 190)
	return v
}

func TestMoveRightLeft(t *testing.T) {
	v := newTestView("hello\nworld")
	v.moveHorizontal(true, false, false)
	if got := v.Cursor(); got != (Position{Line: 1, Column: 2}) {
		t.Errorf("right: cursor = %+v, want {1,2}", got)
	}
	v.moveHorizontal(false, false, false)
	if got := v.Cursor(); got != (Position{Line: 1, Column: 1}) {
		t.Errorf("left: cursor = %+v, want {1,1}", got)
	}
}

func TestMoveRightWrapsToNextLine(t *testing.T) {
	v := newTestView("ab\ncd")
	// Move right four times: 1->2->3 (end of line 1) -> next line start -> 2.
	v.SetCursor(Position{Line: 1, Column: 1})
	for i := 0; i < 4; i++ {
		v.moveHorizontal(true, false, false)
	}
	if got := v.Cursor(); got != (Position{Line: 2, Column: 2}) {
		t.Errorf("after wrapping right: cursor = %+v, want {2,2}", got)
	}
}

func TestMoveVerticalKeepsDesiredColumn(t *testing.T) {
	v := newTestView("abcdef\nxy")
	v.SetCursor(Position{Line: 1, Column: 5})
	v.moveVertical(true, false, false)
	// Desired column 5 exceeds line 2 (2 chars): clamp to line end.
	if got := v.Cursor(); got != (Position{Line: 2, Column: 3}) {
		t.Errorf("down from long line: cursor = %+v, want {2,3}", got)
	}
	// And back up: the short line's column is preserved on the way back.
	v.moveVertical(false, false, false)
	if got := v.Cursor(); got != (Position{Line: 1, Column: 5}) {
		t.Errorf("up back: cursor = %+v, want {1,5}", got)
	}
}

func TestMoveWord(t *testing.T) {
	v := newTestView("func fooBar(baz int)")
	v.SetCursor(Position{Line: 1, Column: 1})
	v.moveHorizontal(true, true, false) // ctrl+right: past "func"
	if got := v.Cursor(); got != (Position{Line: 1, Column: 5}) {
		t.Errorf("ctrl+right 1: cursor = %+v, want {1,5}", got)
	}
	v.moveHorizontal(true, true, false) // past "fooBar"
	if got := v.Cursor(); got != (Position{Line: 1, Column: 12}) {
		t.Errorf("ctrl+right 2: cursor = %+v, want {1,12}", got)
	}
	v.moveHorizontal(false, true, false) // ctrl+left: back to start of "fooBar"
	if got := v.Cursor(); got != (Position{Line: 1, Column: 6}) {
		t.Errorf("ctrl+left: cursor = %+v, want {1,6}", got)
	}
}

func TestShiftSelection(t *testing.T) {
	v := newTestView("hello world")
	v.SetCursor(Position{Line: 1, Column: 1})
	v.moveHorizontal(true, false, true) // shift+right
	v.moveHorizontal(true, false, true)
	sel := v.Selection()
	if sel.Anchor != (Position{Line: 1, Column: 1}) || sel.Active != (Position{Line: 1, Column: 3}) {
		t.Errorf("selection = anchor %+v active %+v, want {1,1}..{1,3}", sel.Anchor, sel.Active)
	}
}

func TestPositionFromPixel(t *testing.T) {
	v := newRealCanvasView("hello world")
	// Click just past the real start of column 6 ("hello " ends there) so the
	// mapping resolves to column 6 using true glyph widths.
	x := v.VM.TextLeft() + v.Options.PaddingLeft + v.renderXForColumn(v.lastCanvas, 1, 6) + 1
	y := v.Options.PaddingTop
	pos := v.positionFromPixel(geometry.Pt(x, y))
	if pos.Line != 1 {
		t.Errorf("pixel line = %d, want 1", pos.Line)
	}
	if pos.Column != 6 {
		t.Errorf("pixel column = %d, want 6", pos.Column)
	}
}

func TestRenderXForColumnAndXToColumnReal(t *testing.T) {
	v := newRealCanvasView("hello world")

	// The caret before column 4 must sit at the real measured width of "hel".
	want := float32(0)
	for _, ch := range []rune("hel") {
		want += v.lastCanvas.MeasureText(string(ch), v.Options.FontSize, false)
	}
	if got := v.renderXForColumn(v.lastCanvas, 1, 4); got != want {
		t.Errorf("renderXForColumn(1,4) = %v, want %v", got, want)
	}

	// A click just past that position maps back to column 4.
	if got := v.xToColumnReal(1, want+1); got != 4 {
		t.Errorf("xToColumnReal(col4 start) = %d, want 4", got)
	}

	// The midpoint of the first glyph snaps to column 1 or 2.
	firstW := v.lastCanvas.MeasureText("h", v.Options.FontSize, false)
	col := v.xToColumnReal(1, firstW/2)
	if col < 1 || col > 2 {
		t.Errorf("xToColumnReal(first glyph midpoint) = %d, want 1..2", col)
	}

	// The end of the line maps to the max column (11 chars -> 12).
	endX := v.renderXForColumn(v.lastCanvas, 1, 12)
	if got := v.xToColumnReal(1, endX); got != 12 {
		t.Errorf("xToColumnReal(EOL) = %d, want 12", got)
	}

	// Fallback: without a cached canvas the layout grid is used (CharWidth 7).
	v2 := newTestView("hello world")
	if got := v2.xToColumnReal(1, 7*5+1); got != 6 {
		t.Errorf("fallback xToColumnReal = %d, want 6", got)
	}
}

func TestSelectWord(t *testing.T) {
	v := newTestView("foo bar_baz qux")
	v.selectWord(Position{Line: 1, Column: 7}) // inside bar_baz
	sel := v.Selection()
	if sel.Anchor != (Position{Line: 1, Column: 5}) || sel.Active != (Position{Line: 1, Column: 12}) {
		t.Errorf("word selection = %+v..%+v, want {1,5}..{1,12}", sel.Anchor, sel.Active)
	}
}

func TestMoveToDocumentEnd(t *testing.T) {
	v := newTestView("a\nb\nc")
	v.moveToDocumentEnd(false)
	if got := v.Cursor(); got != (Position{Line: 3, Column: 2}) {
		t.Errorf("doc end cursor = %+v, want {3,2}", got)
	}
}

func TestTokenColorAt(t *testing.T) {
	kw := widget.RGBA8(0x56, 0x9C, 0xD6, 0xFF) // keyword-ish
	fn := widget.RGBA8(0xD2, 0xDC, 0x8F, 0xFF) // function-ish
	fg := foregroundColor

	// "func main" -> tokenized as [1,5) keyword, [5,6) default, [6,10) function.
	spans := []TokenSpan{
		{Start: 1, End: 5, Color: kw},
		{Start: 5, End: 6, Color: fg},
		{Start: 6, End: 10, Color: fn},
	}

	// Inside the first token.
	if got := tokenColorAt(spans, 1, fg); got != kw {
		t.Errorf("col 1 = %v, want keyword %v", got, kw)
	}
	if got := tokenColorAt(spans, 4, fg); got != kw {
		t.Errorf("col 4 = %v, want keyword %v", got, kw)
	}
	// Boundary column of the whitespace token.
	if got := tokenColorAt(spans, 5, fg); got != fg {
		t.Errorf("col 5 = %v, want default %v", got, fg)
	}
	// First letter of the second token must take its token color, not the
	// default foreground (regression: End < col skipped the boundary span).
	if got := tokenColorAt(spans, 6, fg); got != fn {
		t.Errorf("col 6 (first letter of next token) = %v, want function %v", got, fn)
	}
	if got := tokenColorAt(spans, 9, fg); got != fn {
		t.Errorf("col 9 = %v, want function %v", got, fn)
	}
	// Column past the last token's end falls back.
	if got := tokenColorAt(spans, 10, fg); got != fg {
		t.Errorf("col 10 = %v, want default %v", got, fg)
	}

	// Columns in a gap between non-contiguous spans fall back.
	gap := []TokenSpan{{Start: 3, End: 4, Color: kw}}
	if got := tokenColorAt(gap, 1, fg); got != fg {
		t.Errorf("gap col 1 = %v, want default %v", got, fg)
	}
	if got := tokenColorAt(gap, 2, fg); got != fg {
		t.Errorf("gap col 2 = %v, want default %v", got, fg)
	}
	if got := tokenColorAt(gap, 3, fg); got != kw {
		t.Errorf("gap col 3 = %v, want keyword %v", got, kw)
	}
}
