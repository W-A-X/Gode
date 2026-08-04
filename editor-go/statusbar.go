package editor

import (
	"fmt"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// StatusItem represents a single item in the status bar.
type StatusItem struct {
	Label    string
	Value    string
	Tooltip  string
	IsRight  bool
	IconName string
}

// StatusBar is a gogpu/ui widget that renders a VS Code-style bottom status bar
// with information about cursor position, language, encoding, and other state.
type StatusBar struct {
	widget.WidgetBase

	// Items is the list of status items to display.
	Items []StatusItem

	// CursorLine is the current cursor line (1-based).
	CursorLine int

	// CursorColumn is the current cursor column (1-based).
	CursorColumn int

	// Language is the current language mode (e.g., "JavaScript").
	Language string

	// Encoding is the file encoding (e.g., "UTF-8").
	Encoding string

	// LineEnding is the line ending style (e.g., "LF").
	LineEnding string

	// Indent is the indentation style (e.g., "Spaces: 2").
	Indent string

	// Options carries visual configuration.
	Options StatusBarOptions
}

// StatusBarOptions configures the appearance of StatusBar.
type StatusBarOptions struct {
	Height       float32
	FontSize     float32
	PaddingLeft  float32
	PaddingRight float32
	ItemGap      float32
}

// DefaultStatusBarOptions returns sensible VS Code-like defaults.
func DefaultStatusBarOptions() StatusBarOptions {
	return StatusBarOptions{
		Height:       24,
		FontSize:     12,
		PaddingLeft:  8,
		PaddingRight: 8,
		ItemGap:      16,
	}
}

// NewStatusBar creates a new StatusBar widget.
func NewStatusBar() *StatusBar {
	return &StatusBar{
		Options: DefaultStatusBarOptions(),
	}
}

// SetCursor updates the cursor position display.
func (s *StatusBar) SetCursor(line, column int) {
	s.CursorLine = line
	s.CursorColumn = column
	s.SetNeedsRedraw(true)
}

// SetLanguage updates the language mode display.
func (s *StatusBar) SetLanguage(lang string) {
	s.Language = lang
	s.SetNeedsRedraw(true)
}

// Layout implements widget.Widget.
func (s *StatusBar) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	return geometry.Size{Width: c.MaxWidth, Height: s.Options.Height}
}

// Children implements widget.Widget.
func (s *StatusBar) Children() []widget.Widget { return nil }

// Draw implements widget.Widget.
func (s *StatusBar) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := s.Bounds()
	size := bounds.Size()
	if size.Width <= 0 || size.Height <= 0 {
		return
	}

	// Background
	canvas.DrawRect(geometry.Rect{Min: geometry.Pt(0, 0), Max: size.ToPoint()}, StatusBarBackgroundColor)

	// Left side items (cursor position)
	leftX := s.Options.PaddingLeft
	cursorText := fmt.Sprintf("%d:%d", s.CursorLine, s.CursorColumn)
	canvas.DrawText(cursorText, geometry.Rect{
		Min: geometry.Pt(leftX, 0),
		Max: geometry.Pt(leftX+80, size.Height),
	}, s.Options.FontSize, StatusBarTextColor, false, widget.TextAlignLeft)

	// Right side items
	rightX := size.Width - s.Options.PaddingRight

	// Indent
	if s.Indent != "" {
		indentW := float32(len(s.Indent)) * s.Options.FontSize * 0.6
		rightX -= indentW
		canvas.DrawText(s.Indent, geometry.Rect{
			Min: geometry.Pt(rightX, 0),
			Max: geometry.Pt(rightX+indentW, size.Height),
		}, s.Options.FontSize, StatusBarTextColor, false, widget.TextAlignRight)
		rightX -= s.Options.ItemGap
	}

	// Encoding
	if s.Encoding != "" {
		encW := float32(len(s.Encoding)) * s.Options.FontSize * 0.6
		rightX -= encW
		canvas.DrawText(s.Encoding, geometry.Rect{
			Min: geometry.Pt(rightX, 0),
			Max: geometry.Pt(rightX+encW, size.Height),
		}, s.Options.FontSize, StatusBarTextColor, false, widget.TextAlignRight)
		rightX -= s.Options.ItemGap
	}

	// Line ending
	if s.LineEnding != "" {
		leW := float32(len(s.LineEnding)) * s.Options.FontSize * 0.6
		rightX -= leW
		canvas.DrawText(s.LineEnding, geometry.Rect{
			Min: geometry.Pt(rightX, 0),
			Max: geometry.Pt(rightX+leW, size.Height),
		}, s.Options.FontSize, StatusBarTextColor, false, widget.TextAlignRight)
		rightX -= s.Options.ItemGap
	}

	// Language
	if s.Language != "" {
		langW := float32(len(s.Language)) * s.Options.FontSize * 0.6
		rightX -= langW
		canvas.DrawText(s.Language, geometry.Rect{
			Min: geometry.Pt(rightX, 0),
			Max: geometry.Pt(rightX+langW, size.Height),
		}, s.Options.FontSize, StatusBarTextColor, false, widget.TextAlignRight)
	}

	// Custom items
	for _, item := range s.Items {
		if item.IsRight {
			continue // Already handled above
		}
		itemW := float32(len(item.Label)) * s.Options.FontSize * 0.6
		canvas.DrawText(item.Label, geometry.Rect{
			Min: geometry.Pt(leftX, 0),
			Max: geometry.Pt(leftX+itemW, size.Height),
		}, s.Options.FontSize, StatusBarTextColor, false, widget.TextAlignLeft)
		leftX += itemW + s.Options.ItemGap
	}
}

// Event implements widget.Widget. StatusBar does not handle events.
func (s *StatusBar) Event(ctx widget.Context, e event.Event) bool {
	return false
}

// StatusBar Colors (JetBrains Dark)
var (
	StatusBarBackgroundColor = widget.RGBA8(0x00, 0x7A, 0x33, 0xFF) // Green
	StatusBarTextColor       = widget.RGBA8(0xFF, 0xFF, 0xFF, 0xFF) // White
	StatusBarErrorColor      = widget.RGBA8(0xE5, 0x14, 0x00, 0xFF) // Red
	StatusBarWarningColor    = widget.RGBA8(0xDC, 0xAA, 0x2F, 0xFF) // Yellow
)
