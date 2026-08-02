package engine

import (
	"strings"
	"testing"

	"github.com/gogpu/ui/event"
)

func TestRenderProducesPixels(t *testing.T) {
	e := New(320, 200)
	e.SetText("hello\nworld")
	data, ok := e.Render()
	if !ok {
		t.Fatal("Render returned ok=false")
	}
	if len(data) != 320*200*4 {
		t.Fatalf("pixel buffer = %d bytes, want %d", len(data), 320*200*4)
	}
	// Background is opaque dark: first pixel alpha must be 255.
	if data[3] != 0xFF {
		t.Errorf("first pixel alpha = %02x, want ff (opaque background)", data[3])
	}
}

func TestKeyNavigationChangesSelection(t *testing.T) {
	e := New(400, 200)
	e.SetText("hello world")

	// Arrow right moves the cursor.
	e.HandleEvent(BuildKeyEvent(InputKey{KeyType: "press", Key: "Right"}))
	e.HandleEvent(BuildKeyEvent(InputKey{KeyType: "press", Key: "Right"}))
	anchor, active := e.Selection()
	if anchor != (Pos{Line: 1, Column: 3}) || active != (Pos{Line: 1, Column: 3}) {
		t.Errorf("after 2x right: anchor=%+v active=%+v, want {1,3}", anchor, active)
	}

	// Shift+Right extends the selection.
	e.HandleEvent(BuildKeyEvent(InputKey{KeyType: "press", Key: "Right", Shift: true}))
	anchor, active = e.Selection()
	if anchor != (Pos{Line: 1, Column: 3}) || active != (Pos{Line: 1, Column: 4}) {
		t.Errorf("after shift+right: anchor=%+v active=%+v, want {1,3}..{1,4}", anchor, active)
	}
}

func TestTypingInsertsAndNotifies(t *testing.T) {
	e := New(400, 200)
	e.SetText("ab\ncd")

	var gotRange Range
	var gotText string
	e.SetOnDidChange(func(r Range, text string) {
		gotRange = r
		gotText = text
	})

	e.HandleEvent(BuildKeyEvent(InputKey{KeyType: "press", Key: "A", Rune: "x"}))
	e.Render()

	if gotText != "x" {
		t.Errorf("edited text = %q, want %q", gotText, "x")
	}
	if gotRange.Start != (Pos{Line: 1, Column: 1}) || gotRange.End != (Pos{Line: 1, Column: 1}) {
		t.Errorf("edited range = %+v, want insert at {1,1}", gotRange)
	}

	if got := e.GetContent(); got != "xab\ncd" {
		t.Errorf("content = %q, want %q", got, "xab\ncd")
	}

	// Cursor should have moved past the inserted char.
	anchor, active := e.Selection()
	if active != (Pos{Line: 1, Column: 2}) {
		t.Errorf("cursor after insert = %+v, want {1,2}", active)
	}
	_ = anchor
}

func TestEnterSplitsLine(t *testing.T) {
	e := New(400, 200)
	e.SetText("ab\ncd")
	e.SetSelection(Pos{Line: 1, Column: 2}, Pos{Line: 1, Column: 2})

	e.HandleEvent(BuildKeyEvent(InputKey{KeyType: "press", Key: "Enter", Rune: "\n"}))
	e.Render()

	want := "a\nb\ncd"
	if got := e.GetContent(); got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
	_, active := e.Selection()
	if active != (Pos{Line: 2, Column: 1}) {
		t.Errorf("cursor after enter = %+v, want {2,1}", active)
	}
}

func TestBackspaceDeletes(t *testing.T) {
	e := New(400, 200)
	e.SetText("hello")
	e.SetSelection(Pos{Line: 1, Column: 4}, Pos{Line: 1, Column: 4})

	e.HandleEvent(BuildKeyEvent(InputKey{KeyType: "press", Key: "Backspace"}))
	e.Render()

	if got := e.GetContent(); got != "helo" {
		t.Errorf("content = %q, want %q", got, "helo")
	}
}

// TestBackspaceDeletesCharBeforeCursor verifies Backspace removes the
// character BEFORE the caret and the caret lands exactly on the previous
// glyph position (pixel<->column alignment is covered in editor/view_test.go).
func TestBackspaceDeletesCharBeforeCursor(t *testing.T) {
	e := New(400, 200)
	e.SetText("abcd")
	e.SetSelection(Pos{Line: 1, Column: 3}, Pos{Line: 1, Column: 3}) // caret after 'b'
	e.Render()

	e.HandleEvent(BuildKeyEvent(InputKey{KeyType: "press", Key: "Backspace"}))
	e.Render()

	if got := e.GetContent(); got != "acd" {
		t.Fatalf("content = %q, want %q", got, "acd")
	}
	_, active := e.Selection()
	if active != (Pos{Line: 1, Column: 2}) {
		t.Fatalf("caret after backspace = %+v, want {1,2}", active)
	}
}

func TestMouseClickMovesCursor(t *testing.T) {
	e := New(400, 200)
	e.SetText("hello world")
	e.Render() // measures the real char width and caches lastCanvas

	// Click just past the real start of column 4. The default font is
	// proportional, so positions must be computed with measured glyph widths
	// (matching xToColumnReal) rather than the CharWidth layout grid, or the
	// click would land on a different character.
	want := Pos{Line: 1, Column: 4}
	var rx float32
	for _, ch := range []rune("hel") {
		rx += e.canvas.MeasureText(string(ch), e.view.Options.FontSize, false)
	}
	x := rx + e.view.VM.TextLeft() + e.view.Options.PaddingLeft + 1
	y := float32(2)
	e.HandleEvent(BuildMouseEvent(InputMouse{MouseType: "press", Button: "left", X: x, Y: y}))
	e.HandleEvent(BuildMouseEvent(InputMouse{MouseType: "release", Button: "left", X: x, Y: y}))
	e.Render()

	anchor, active := e.Selection()
	if active.Line != 1 {
		t.Errorf("mouse click line = %d, want 1", active.Line)
	}
	if active.Column != want.Column {
		t.Errorf("mouse click column = %d, want %d", active.Column, want.Column)
	}
	_ = anchor
}

func TestWheelScrolls(t *testing.T) {
	e := New(400, 200)
	var b strings.Builder
	for i := 0; i < 50; i++ {
		b.WriteString("line\n")
	}
	e.SetText(b.String())
	e.Render()

	before := e.view.VM.ScrollTop()
	e.HandleEvent(BuildWheelEvent(InputWheel{DY: 40}))
	e.Render()
	after := e.view.VM.ScrollTop()
	if after <= before {
		t.Errorf("wheel scroll: top %v -> %v, want increase", before, after)
	}
}

func TestSetTextResetsCursor(t *testing.T) {
	e := New(400, 200)
	e.SetText("first")
	e.SetSelection(Pos{Line: 1, Column: 3}, Pos{Line: 1, Column: 3})
	e.SetText("second")
	anchor, active := e.Selection()
	if anchor != (Pos{Line: 1, Column: 1}) || active != (Pos{Line: 1, Column: 1}) {
		t.Errorf("cursor after SetText = %+v/%+v, want {1,1}", anchor, active)
	}
}

func TestTabInsertsIndentationAndKeepsFocus(t *testing.T) {
	e := New(400, 200)
	e.SetText("hello")

	// A plain Tab must insert indentation. The gogpu/ui focus manager would
	// normally consume Tab for widget navigation (and, without the editor
	// being tracked as focusable, release the editor's focus); the engine
	// handles it directly instead.
	e.HandleEvent(BuildKeyEvent(InputKey{KeyType: "press", Key: "Tab"}))
	e.Render()
	if got := e.GetContent(); got != "\thello" {
		t.Errorf("after Tab: content = %q, want %q", got, "\thello")
	}

	// Focus must survive the Tab press so the next keystroke still lands at
	// the cursor instead of being dropped.
	e.HandleEvent(BuildKeyEvent(InputKey{KeyType: "press", Key: "A", Rune: "x"}))
	e.Render()
	if got := e.GetContent(); got != "\txhello" {
		t.Errorf("after Tab then x: content = %q, want %q", got, "\txhello")
	}
	_, active := e.Selection()
	if active != (Pos{Line: 1, Column: 3}) {
		t.Errorf("cursor after Tab+x = %+v, want {1,3}", active)
	}
}

func TestLookupKey(t *testing.T) {
	cases := map[string]event.Key{
		"A":       event.KeyA,
		"a":       event.KeyA,
		"Up":      event.KeyUp,
		"Enter":   event.KeyEnter,
		"PageUp":  event.KeyPageUp,
		"0":       event.Key0,
		"F5":      event.KeyF5,
		"Escape":  event.KeyEscape,
		"unknown": event.KeyUnknown,
	}
	for name, want := range cases {
		if got := lookupKey(name); got != want {
			t.Errorf("lookupKey(%q) = %v, want %v", name, got, want)
		}
	}
}
