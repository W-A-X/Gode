package editor

import (
	"math"
)

// ViewOptions configures how the editor renders text. It mirrors the subset
// of VS Code's editor options that affect the view layer.
type ViewOptions struct {
	// FontSize is the font size in logical pixels used for rendering.
	FontSize float32

	// LineHeight is the height of a single rendered line in logical pixels.
	LineHeight float32

	// TabSize is the number of character cells a tab advances.
	TabSize int

	// PaddingLeft / PaddingTop are the insets of the text area.
	PaddingLeft float32
	PaddingTop  float32

	// GlyphMarginWidth is the width of the left margin column (used by VS Code
	// for breakpoints / folding controls). A width of 0 disables it.
	GlyphMarginWidth float32

	// LineNumbers renders line numbers in a column to the left of the text.
	LineNumbers bool

	// CursorWidth is the caret width in logical pixels.
	CursorWidth float32
}

// DefaultOptions returns VS Code dark-theme defaults: 14px font, 19px lines.
func DefaultOptions() ViewOptions {
	return ViewOptions{
		FontSize:         14,
		LineHeight:       19,
		TabSize:          4,
		PaddingLeft:      6,
		PaddingTop:       4,
		GlyphMarginWidth: 0,
		LineNumbers:      true,
		CursorWidth:      2,
	}
}

// ViewModel is the pure geometry/layout layer of the editor. It maps between
// buffer coordinates (lines, columns) and view coordinates (pixels), and owns
// the scroll state. It has no dependency on the rendering toolkit, which keeps
// it unit-testable and lets any backend drive it.
type ViewModel struct {
	Model   ITextModel
	Options ViewOptions

	// CharWidth is the width of one monospace character cell in logical
	// pixels. The view measures it from the real font every frame and stores
	// it here; all column<->pixel conversions go through it. This matches
	// VS Code, where the view layout uses a "typical char width" while glyph
	// rendering uses the real font.
	CharWidth float32

	viewportWidth  float32
	viewportHeight float32
	scrollTop      float32
	scrollLeft     float32
}

// NewViewModel creates a view model over the given model and options.
func NewViewModel(m ITextModel, opts ViewOptions) *ViewModel {
	return &ViewModel{Model: m, Options: opts}
}

// SetCharWidth updates the character-cell width (called by the view each frame).
func (vm *ViewModel) SetCharWidth(w float32) {
	if w > 0 {
		vm.CharWidth = w
	}
}

// SetViewport updates the size of the visible area in logical pixels.
func (vm *ViewModel) SetViewport(w, h float32) {
	vm.viewportWidth = w
	vm.viewportHeight = h
	vm.clampScroll()
}

// LineCount returns the number of lines in the model.
func (vm *ViewModel) LineCount() int { return vm.Model.LineCount() }

// LineHeight returns the height of one line.
func (vm *ViewModel) LineHeight() float32 {
	if vm.Options.LineHeight > 0 {
		return vm.Options.LineHeight
	}
	return 19
}

// ContentHeight returns the total height of all lines.
func (vm *ViewModel) ContentHeight() float32 {
	return float32(vm.LineCount()) * vm.LineHeight()
}

// LineNumbersWidth returns the width of the line-number column, or 0 when
// line numbers are disabled.
func (vm *ViewModel) LineNumbersWidth() float32 {
	if !vm.Options.LineNumbers {
		return 0
	}
	digits := 1
	for n := vm.LineCount(); n >= 10; n /= 10 {
		digits++
	}
	return float32(digits)*vm.CharWidth + 2*vm.Options.PaddingLeft + 8
}

// TextLeft returns the x offset where the text area begins (after line numbers).
func (vm *ViewModel) TextLeft() float32 {
	return vm.LineNumbersWidth() + vm.Options.GlyphMarginWidth
}

// VisibleLines returns the inclusive 1-based range of lines currently visible.
func (vm *ViewModel) VisibleLines() (int, int) {
	lh := vm.LineHeight()
	first := int(math.Floor(float64(vm.scrollTop/lh))) + 1
	last := int(math.Ceil(float64((vm.scrollTop + vm.viewportHeight) / lh)))
	if first < 1 {
		first = 1
	}
	if last > vm.LineCount() {
		last = vm.LineCount()
	}
	if last < first {
		last = first
	}
	return first, last
}

// ViewportLines returns how many full lines fit in the viewport.
func (vm *ViewModel) ViewportLines() int {
	n := int(math.Floor(float64(vm.viewportHeight / vm.LineHeight())))
	if n < 1 {
		n = 1
	}
	return n
}

// LineToViewTop returns the y pixel of the top of the 1-based line.
func (vm *ViewModel) LineToViewTop(line int) float32 {
	return (float32(line) - 1) * vm.LineHeight()
}

// ViewTopToLine converts a y pixel (view-local, unscrolled) to a 1-based line.
func (vm *ViewModel) ViewTopToLine(y float32) int {
	line := int(math.Floor(float64(y/vm.LineHeight()))) + 1
	if line < 1 {
		line = 1
	}
	if line > vm.LineCount() {
		line = vm.LineCount()
	}
	return line
}

// ScrollTop returns the current vertical scroll offset in pixels.
func (vm *ViewModel) ScrollTop() float32 { return vm.scrollTop }

// ScrollLeft returns the current horizontal scroll offset in pixels.
func (vm *ViewModel) ScrollLeft() float32 { return vm.scrollLeft }

// ScrollBy scrolls vertically by dy and horizontally by dx pixels.
func (vm *ViewModel) ScrollBy(dy, dx float32) {
	vm.scrollTop += dy
	vm.scrollLeft += dx
	vm.clampScroll()
}

// ScrollToLine scrolls so that the given 1-based line is visible.
func (vm *ViewModel) ScrollToLine(line int) {
	if line < 1 {
		line = 1
	}
	if line > vm.LineCount() {
		line = vm.LineCount()
	}
	top := vm.LineToViewTop(line)
	bottom := top + vm.LineHeight()
	if top < vm.scrollTop {
		vm.scrollTop = top
	} else if bottom > vm.scrollTop+vm.viewportHeight {
		vm.scrollTop = bottom - vm.viewportHeight
	}
	vm.clampScroll()
}

// RevealPosition scrolls so that the given position is visible, both vertically
// and horizontally.
func (vm *ViewModel) RevealPosition(pos Position) {
	top := vm.LineToViewTop(pos.Line)
	bottom := top + vm.LineHeight()
	if top < vm.scrollTop {
		vm.scrollTop = top
	} else if bottom > vm.scrollTop+vm.viewportHeight {
		vm.scrollTop = bottom - vm.viewportHeight
	}

	x := vm.ColumnToX(pos.Line, pos.Column) + vm.TextLeft()
	if x < vm.scrollLeft {
		vm.scrollLeft = x - 4
	} else if x > vm.scrollLeft+vm.viewportWidth {
		vm.scrollLeft = x - vm.viewportWidth + 16
	}
	vm.clampScroll()
}

func (vm *ViewModel) clampScroll() {
	maxTop := vm.ContentHeight() - vm.viewportHeight
	if maxTop < 0 {
		maxTop = 0
	}
	if vm.scrollTop < 0 {
		vm.scrollTop = 0
	}
	if vm.scrollTop > maxTop {
		vm.scrollTop = maxTop
	}

	maxLeft := vm.ContentWidth() - vm.viewportWidth
	if maxLeft < 0 {
		maxLeft = 0
	}
	if vm.scrollLeft < 0 {
		vm.scrollLeft = 0
	}
	if vm.scrollLeft > maxLeft {
		vm.scrollLeft = maxLeft
	}
}

// tabStop returns the pixel width of one tab.
func (vm *ViewModel) tabStop() float32 { return float32(vm.Options.TabSize) * vm.CharWidth }

// advanceToTabStop returns the next tab stop position strictly greater than cur.
func (vm *ViewModel) advanceToTabStop(cur float32) float32 {
	tab := vm.tabStop()
	return (float32(math.Floor(float64(cur/tab))) + 1) * tab
}

// LineWidth returns the pixel width of the 1-based line with tabs expanded.
func (vm *ViewModel) LineWidth(line int) float32 {
	content := []rune(vm.Model.LineContent(line))
	x := float32(0)
	for _, ch := range content {
		if ch == '\t' {
			x = vm.advanceToTabStop(x)
		} else {
			x += vm.CharWidth
		}
	}
	return x
}

// ContentWidth returns the widest line, plus the horizontal padding.
func (vm *ViewModel) ContentWidth() float32 {
	maxWidth := float32(0)
	for line := 1; line <= vm.LineCount(); line++ {
		if w := vm.LineWidth(line); w > maxWidth {
			maxWidth = w
		}
	}
	return maxWidth + vm.Options.PaddingLeft
}

// ColumnToX converts a 1-based column of a line to a pixel x inside the text
// area (i.e. the offset where the caret before that column sits). Tabs advance
// to the next multiple of the tab stop.
func (vm *ViewModel) ColumnToX(line, col int) float32 {
	content := []rune(vm.Model.LineContent(line))
	n := clamp(col-1, 0, len(content))
	x := float32(0)
	for _, ch := range content[:n] {
		if ch == '\t' {
			x = vm.advanceToTabStop(x)
		} else {
			x += vm.CharWidth
		}
	}
	return x
}

// XToColumn converts a pixel x inside the text area back to a 1-based column,
// snapping to the nearest character boundary.
func (vm *ViewModel) XToColumn(line int, x float32) int {
	content := []rune(vm.Model.LineContent(line))
	col := 1
	cur := float32(0)
	for _, ch := range content {
		var w float32
		if ch == '\t' {
			w = vm.advanceToTabStop(cur) - cur
		} else {
			w = vm.CharWidth
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

// LineVisible reports whether the 1-based line intersects the viewport.
func (vm *ViewModel) LineVisible(line int) bool {
	top := vm.LineToViewTop(line)
	return top < vm.scrollTop+vm.viewportHeight && top+vm.LineHeight() > vm.scrollTop
}
