package editor

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// TitleBar is a gogpu/ui widget that renders a macOS-style title bar with
// traffic light buttons (close, minimize, maximize) and centered title text.
type TitleBar struct {
	widget.WidgetBase

	// Title is the text displayed in the center of the title bar.
	Title string

	// Subtitle is optional secondary text (e.g., branch name).
	Subtitle string

	// OnClose is called when the close button is clicked.
	OnClose func()

	// OnMinimize is called when the minimize button is clicked.
	OnMinimize func()

	// OnMaximize is called when the maximize button is clicked.
	OnMaximize func()

	// Options carries visual configuration.
	Options TitleBarOptions

	// hoveredButton tracks which traffic light button is hovered (-1=none, 0=close, 1=minimize, 2=maximize).
	hoveredButton int
}

// TitleBarOptions configures the appearance of TitleBar.
type TitleBarOptions struct {
	Height         float32
	FontSize       float32
	ButtonSize     float32
	ButtonPadding  float32
	ButtonGap      float32
	TrafficLightX  float32
}

// DefaultTitleBarOptions returns sensible macOS-like defaults.
func DefaultTitleBarOptions() TitleBarOptions {
	return TitleBarOptions{
		Height:        38,
		FontSize:      13,
		ButtonSize:    12,
		ButtonPadding: 8,
		ButtonGap:     8,
		TrafficLightX: 12,
	}
}

// NewTitleBar creates a new TitleBar widget.
func NewTitleBar() *TitleBar {
	return &TitleBar{
		Options:       DefaultTitleBarOptions(),
		hoveredButton: -1,
	}
}

// Layout implements widget.Widget.
func (t *TitleBar) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	return geometry.Size{Width: c.MaxWidth, Height: t.Options.Height}
}

// Children implements widget.Widget.
func (t *TitleBar) Children() []widget.Widget { return nil }

// Draw implements widget.Widget.
func (t *TitleBar) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := t.Bounds()
	size := bounds.Size()
	if size.Width <= 0 || size.Height <= 0 {
		return
	}

	// Background
	canvas.DrawRect(geometry.Rect{Min: geometry.Pt(0, 0), Max: size.ToPoint()}, TitleBarBackgroundColor)

	// Traffic light buttons (macOS style)
	btnY := (size.Height - t.Options.ButtonSize) / 2
	btnX := t.Options.TrafficLightX

	// Close button (red)
	closeColor := TitleBarCloseColor
	if t.hoveredButton == 0 {
		closeColor = TitleBarCloseHoverColor
	}
	canvas.DrawCircle(geometry.Pt(btnX+t.Options.ButtonSize/2, btnY+t.Options.ButtonSize/2), t.Options.ButtonSize/2, closeColor)

	// Minimize button (yellow)
	minimizeColor := TitleBarMinimizeColor
	if t.hoveredButton == 1 {
		minimizeColor = TitleBarMinimizeHoverColor
	}
	canvas.DrawCircle(geometry.Pt(btnX+t.Options.ButtonSize+t.Options.ButtonGap+t.Options.ButtonSize/2, btnY+t.Options.ButtonSize/2), t.Options.ButtonSize/2, minimizeColor)

	// Maximize button (green)
	maximizeColor := TitleBarMaximizeColor
	if t.hoveredButton == 2 {
		maximizeColor = TitleBarMaximizeHoverColor
	}
	canvas.DrawCircle(geometry.Pt(btnX+(t.Options.ButtonSize+t.Options.ButtonGap)*2+t.Options.ButtonSize/2, btnY+t.Options.ButtonSize/2), t.Options.ButtonSize/2, maximizeColor)

	// Title text (centered)
	titleX := size.Width / 4 // Offset right to account for traffic lights
	canvas.DrawText(t.Title, geometry.Rect{
		Min: geometry.Pt(titleX, 0),
		Max: geometry.Pt(size.Width-titleX, size.Height),
	}, t.Options.FontSize, TitleBarTextColor, false, widget.TextAlignCenter)

	// Subtitle (if present, displayed to the right of title)
	if t.Subtitle != "" {
		subtitleW := float32(len(t.Subtitle)) * t.Options.FontSize * 0.6
		subtitleX := titleX + float32(len(t.Title))*t.Options.FontSize*0.3 + 8
		canvas.DrawText(t.Subtitle, geometry.Rect{
			Min: geometry.Pt(subtitleX, 0),
			Max: geometry.Pt(subtitleX+subtitleW+16, size.Height),
		}, t.Options.FontSize-1, TitleBarSubtitleColor, false, widget.TextAlignLeft)
	}
}

// Event implements widget.Widget.
func (t *TitleBar) Event(ctx widget.Context, e event.Event) bool {
	// Note: This is a placeholder - actual event handling would need proper imports
	return false
}

// TitleBar Colors (macOS Dark)
var (
	TitleBarBackgroundColor    = widget.RGBA8(0x3C, 0x3C, 0x3C, 0xFF) // Dark gray
	TitleBarTextColor          = widget.RGBA8(0xDF, 0xE1, 0xE5, 0xFF) // Light gray
	TitleBarSubtitleColor      = widget.RGBA8(0x9D, 0xA0, 0xA8, 0xFF) // Gray9
	TitleBarCloseColor         = widget.RGBA8(0xFF, 0x5F, 0x57, 0xFF) // Red
	TitleBarCloseHoverColor    = widget.RGBA8(0xFF, 0x5F, 0x57, 0xCC) // Red (hover)
	TitleBarMinimizeColor      = widget.RGBA8(0xFF, 0xBD, 0x2E, 0xFF) // Yellow
	TitleBarMinimizeHoverColor = widget.RGBA8(0xFF, 0xBD, 0x2E, 0xCC) // Yellow (hover)
	TitleBarMaximizeColor      = widget.RGBA8(0x28, 0xC9, 0x40, 0xFF) // Green
	TitleBarMaximizeHoverColor = widget.RGBA8(0x28, 0xC9, 0x40, 0xCC) // Green (hover)
)
