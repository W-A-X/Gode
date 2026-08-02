package editor

import "testing"

func newTestVM(text string) *ViewModel {
	m := NewTextModel(text)
	vm := NewViewModel(m, DefaultOptions())
	vm.SetCharWidth(7) // typical 14px font cell width
	return vm
}

func TestVisibleLines(t *testing.T) {
	vm := newTestVM(makeLines(50))
	vm.SetViewport(400, 190) // 19px lines => 10 visible
	first, last := vm.VisibleLines()
	if first != 1 || last != 10 {
		t.Fatalf("VisibleLines = %d..%d, want 1..10", first, last)
	}

	vm.ScrollBy(19*10, 0) // scroll down 10 lines
	first, last = vm.VisibleLines()
	if first != 11 || last != 20 {
		t.Errorf("after scroll VisibleLines = %d..%d, want 11..20", first, last)
	}
}

func TestScrollClamp(t *testing.T) {
	vm := newTestVM(makeLines(10))
	vm.SetViewport(400, 190) // 10 lines content, viewport shows 10
	vm.ScrollBy(1000, 0)
	if got := vm.ScrollTop(); got != 0 {
		t.Errorf("ScrollTop = %v, want 0 (content fits viewport)", got)
	}

	vm = newTestVM(makeLines(100))
	vm.SetViewport(400, 190)
	maxTop := float32(90) * 19 // 100 lines, 10 visible => max scroll 90*19
	vm.ScrollBy(1e6, 0)
	if got := vm.ScrollTop(); got != maxTop {
		t.Errorf("ScrollTop = %v, want %v", got, maxTop)
	}
}

func TestColumnToXWithTabs(t *testing.T) {
	vm := newTestVM("a\tb")
	// tab stop = 4 cells * 7px = 28px
	// 'a' at 0..7, tab advances to 28, 'b' at 28..35
	if got := vm.ColumnToX(1, 2); got != 7 {
		t.Errorf("ColumnToX(1,2) = %v, want 7", got)
	}
	if got := vm.ColumnToX(1, 3); got != 28 {
		t.Errorf("ColumnToX(1,3) = %v, want 28", got)
	}
	if got := vm.ColumnToX(1, 4); got != 35 {
		t.Errorf("ColumnToX(1,4) = %v, want 35", got)
	}
}

func TestXToColumnRoundTrip(t *testing.T) {
	vm := newTestVM("hello world\tend")
	line := 1
	maxCol := vm.Model.LineMaxColumn(line)
	for col := 1; col <= maxCol; col++ {
		x := vm.ColumnToX(line, col)
		back := vm.XToColumn(line, x)
		if back != col {
			t.Errorf("round-trip col %d -> x %v -> col %d", col, x, back)
		}
	}
}

func TestRevealPosition(t *testing.T) {
	vm := newTestVM(makeLines(100))
	vm.SetViewport(400, 190)

	vm.RevealPosition(Position{Line: 50, Column: 1})
	first, last := vm.VisibleLines()
	if first > 50 || last < 50 {
		t.Errorf("after reveal 50: visible %d..%d, want to contain 50", first, last)
	}

	// Revealing line 2 from far below should scroll back up.
	vm.RevealPosition(Position{Line: 2, Column: 1})
	if got := vm.ScrollTop(); got != 19 {
		t.Errorf("ScrollTop = %v, want 19 (line 2 at top)", got)
	}
}

func TestViewTopToLine(t *testing.T) {
	vm := newTestVM(makeLines(10))
	for line := 1; line <= 10; line++ {
		y := vm.LineToViewTop(line)
		if got := vm.ViewTopToLine(y); got != line {
			t.Errorf("ViewTopToLine(%v) = %d, want %d", y, got, line)
		}
	}
}

func TestContentHeight(t *testing.T) {
	vm := newTestVM(makeLines(20))
	if got := vm.ContentHeight(); got != 20*19 {
		t.Errorf("ContentHeight = %v, want %v", got, 20*19)
	}
}

func makeLines(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			s += "\n"
		}
		s += "line"
	}
	return s
}
