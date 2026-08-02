package editor

import "github.com/gogpu/ui/widget"

// VS Code dark theme palette.
var (
	// Editor colors.
	backgroundColor         = widget.RGBA8(0x1E, 0x1E, 0x1E, 0xFF)
	foregroundColor         = widget.RGBA8(0xD4, 0xD4, 0xD4, 0xFF)
	currentLineColor        = widget.RGBA8(0x2A, 0x2A, 0x2A, 0xFF)
	selectionColor          = widget.RGBA8(0x26, 0x4F, 0x78, 0xFF)
	cursorColor             = widget.RGBA8(0xAE, 0xAF, 0xAD, 0xFF)
	cursorUnfocusedColor    = widget.RGBA8(0xAE, 0xAF, 0xAD, 0x60)
	lineNumberColor         = widget.RGBA8(0x85, 0x85, 0x85, 0xFF)
	activeLineNumberColor   = widget.RGBA8(0xC6, 0xC6, 0xC6, 0xFF)
	scrollbarSliderColor    = widget.RGBA8(0x79, 0x79, 0x79, 0x55)
	scrollbarSliderHover    = widget.RGBA8(0x79, 0x79, 0x79, 0x88)
	// breakpointColor matches VS Code's debug breakpoint red (E51400).
	breakpointColor = widget.RGBA8(0xE5, 0x14, 0x00, 0xFF)
)

// ForegroundColor returns the editor's default text color. It is the fallback
// token color when the host does not supply a syntax-highlighting span for a
// range (or supplies an unparseable color).
func ForegroundColor() widget.Color { return foregroundColor }
