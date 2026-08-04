package editor

import "github.com/gogpu/ui/widget"

// JetBrains Int UI dark palette (devtools DarkScheme from gogpu/ui examples/ide).
var (
	// Editor colors.
	backgroundColor         = widget.RGBA8(0x1E, 0x1F, 0x22, 0xFF) // Gray1, editor area
	foregroundColor         = widget.RGBA8(0xDF, 0xE1, 0xE5, 0xFF) // Gray12, OnSurface
	currentLineColor        = widget.RGBA8(0x2B, 0x2D, 0x30, 0xFF) // Gray2, caret row
	selectionColor          = widget.RGBA8(0x2E, 0x43, 0x6E, 0xFF) // Blue2, selection
	cursorColor             = widget.RGBA8(0xDF, 0xE1, 0xE5, 0xFF)
	cursorUnfocusedColor    = widget.RGBA8(0xDF, 0xE1, 0xE5, 0x60)
	lineNumberColor         = widget.RGBA8(0x6F, 0x73, 0x7A, 0xFF) // Gray7, gutter
	activeLineNumberColor   = widget.RGBA8(0x9D, 0xA0, 0xA8, 0xFF) // Gray9, caret-row gutter
	scrollbarSliderColor    = widget.RGBA8(0x4E, 0x51, 0x57, 0x99) // Gray5
	scrollbarSliderHover    = widget.RGBA8(0x6F, 0x73, 0x7A, 0xCC) // Gray7
	// breakpointColor matches VS Code's debug breakpoint red (E51400).
	breakpointColor = widget.RGBA8(0xE5, 0x14, 0x00, 0xFF)
)

// ForegroundColor returns the editor's default text color. It is the fallback
// token color when the host does not supply a syntax-highlighting span for a
// range (or supplies an unparseable color).
func ForegroundColor() widget.Color { return foregroundColor }
