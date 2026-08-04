package editor

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// InputBar is a gogpu/ui widget that renders a VS Code-style bottom input bar
// for sending messages to an AI agent, with placeholder text and icon.
type InputBar struct {
	widget.WidgetBase

	// Placeholder is the hint text shown when input is empty.
	Placeholder string

	// Value is the current input text.
	Value string

	// CursorPos is the cursor position within the text (0-based).
	CursorPos int

	// IsFocused indicates whether the input bar has keyboard focus.
	IsFocused bool

	// OnSubmit is called when Enter is pressed.
	OnSubmit func(text string)

	// OnChange is called when the text changes.
	OnChange func(text string)

	// Options carries visual configuration.
	Options InputBarOptions

	// scrollX is horizontal scroll offset for long text.
	scrollX float32
}

// InputBarOptions configures the appearance of InputBar.
type InputBarOptions struct {
	Height       float32
	FontSize     float32
	PaddingLeft  float32
	PaddingRight float32
	IconSize     float32
	IconGap      float32
}

// DefaultInputBarOptions returns sensible VS Code-like defaults.
func DefaultInputBarOptions() InputBarOptions {
	return InputBarOptions{
		Height:       44,
		FontSize:     13,
		PaddingLeft:  12,
		PaddingRight: 12,
		IconSize:     18,
		IconGap:      8,
	}
}

// NewInputBar creates a new InputBar widget.
func NewInputBar() *InputBar {
	return &InputBar{
		Placeholder: "Message the Zed Agent, @ to include context, / for commands",
		Options:     DefaultInputBarOptions(),
	}
}

// SetValue sets the input text and updates cursor position.
func (ib *InputBar) SetValue(text string) {
	ib.Value = text
	ib.CursorPos = len(text)
	if ib.OnChange != nil {
		ib.OnChange(text)
	}
	ib.SetNeedsRedraw(true)
}

// Layout implements widget.Widget.
func (ib *InputBar) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	return geometry.Size{Width: c.MaxWidth, Height: ib.Options.Height}
}

// Children implements widget.Widget.
func (ib *InputBar) Children() []widget.Widget { return nil }

// Draw implements widget.Widget.
func (ib *InputBar) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := ib.Bounds()
	size := bounds.Size()
	if size.Width <= 0 || size.Height <= 0 {
		return
	}

	// Background
	canvas.DrawRect(geometry.Rect{Min: geometry.Pt(0, 0), Max: size.ToPoint()}, InputBarBackgroundColor)

	// Left icon (chat bubble)
	iconX := ib.Options.PaddingLeft
	iconY := (size.Height - ib.Options.IconSize) / 2
	ib.drawChatIcon(canvas, iconX, iconY, ib.Options.IconSize, InputBarIconColor)

	// Input field area
	fieldX := iconX + ib.Options.IconSize + ib.Options.IconGap
	fieldW := size.Width - fieldX - ib.Options.PaddingRight

	// Input field background (subtle border)
	fieldRect := geometry.Rect{
		Min: geometry.Pt(fieldX, 4),
		Max: geometry.Pt(fieldX+fieldW, size.Height-4),
	}
	canvas.StrokeRect(fieldRect, InputBarBorderColor, 1.0)

	// Text or placeholder
	textX := fieldX + 8
	textW := fieldW - 16

	if ib.Value != "" {
		// Actual text
		canvas.DrawText(ib.Value, geometry.Rect{
			Min: geometry.Pt(textX-ib.scrollX, fieldRect.Min.Y),
			Max: geometry.Pt(textX+textW, fieldRect.Max.Y),
		}, ib.Options.FontSize, InputBarTextColor, false, widget.TextAlignLeft)

		// Cursor (blinking effect simulated with focus state)
		if ib.IsFocused {
			cursorX := textX + float32(ib.CursorPos)*ib.Options.FontSize*0.6 - ib.scrollX
			if cursorX >= textX && cursorX <= textX+textW {
				canvas.DrawRect(geometry.Rect{
					Min: geometry.Pt(cursorX, fieldRect.Min.Y+2),
					Max: geometry.Pt(cursorX+1, fieldRect.Max.Y-2),
				}, InputBarCursorColor)
			}
		}
	} else {
		// Placeholder text
		canvas.DrawText(ib.Placeholder, geometry.Rect{
			Min: geometry.Pt(textX, fieldRect.Min.Y),
			Max: geometry.Pt(textX+textW, fieldRect.Max.Y),
		}, ib.Options.FontSize, InputBarPlaceholderColor, false, widget.TextAlignLeft)
	}

	// Right side icons (send button when text is present)
	if ib.Value != "" {
		sendX := fieldX + fieldW - ib.Options.IconSize - 8
		sendY := (size.Height - ib.Options.IconSize) / 2
		ib.drawSendIcon(canvas, sendX, sendY, ib.Options.IconSize, InputBarSendColor)
	}
}

// drawChatIcon draws a simple chat bubble icon.
func (ib *InputBar) drawChatIcon(canvas widget.Canvas, x, y, size float32, c widget.Color) {
	// Rounded rectangle body
	padding := size * 0.1
	body := geometry.Rect{
		Min: geometry.Pt(x+padding, y),
		Max: geometry.Pt(x+size-padding, y+size*0.75),
	}
	canvas.StrokeRect(body, c, 1.5)
	// Tail
	tail := []geometry.Point{
		geometry.Pt(x+size*0.3, y+size*0.75),
		geometry.Pt(x+size*0.2, y+size),
		geometry.Pt(x+size*0.5, y+size*0.75),
	}
	for i := 0; i < len(tail)-1; i++ {
		canvas.DrawLine(tail[i], tail[i+1], c, 1.5)
	}
}

// drawSendIcon draws a simple send/arrow icon.
func (ib *InputBar) drawSendIcon(canvas widget.Canvas, x, y, size float32, c widget.Color) {
	// Paper plane shape
	points := []geometry.Point{
		geometry.Pt(x, y+size*0.2),
		geometry.Pt(x+size, y+size*0.5),
		geometry.Pt(x, y+size*0.8),
		geometry.Pt(x+size*0.3, y+size*0.5),
	}
	// Draw as filled polygon (simplified with lines)
	for i := 0; i < len(points)-1; i++ {
		canvas.DrawLine(points[i], points[i+1], c, 1.5)
	}
	canvas.DrawLine(points[len(points)-1], points[0], c, 1.5)
}

// Event implements widget.Widget.
func (ib *InputBar) Event(ctx widget.Context, e event.Event) bool {
	switch ev := e.(type) {
	case *event.MouseEvent:
		return ib.handleMouse(ctx, ev)
	case *event.KeyEvent:
		if ib.IsFocused {
			return ib.handleKey(ctx, ev)
		}
	}
	return false
}

func (ib *InputBar) handleMouse(ctx widget.Context, e *event.MouseEvent) bool {
	local := ib.GlobalToLocal(e.GlobalPosition)

	switch e.MouseType {
	case event.MousePress:
		if e.Button == event.ButtonLeft {
			// Check if click is in the input field
			fieldX := ib.Options.PaddingLeft + ib.Options.IconSize + ib.Options.IconGap
			if local.X >= fieldX {
				ib.IsFocused = true
				// Simple cursor positioning (could be more sophisticated)
				relX := local.X - fieldX - 8 + ib.scrollX
				ib.CursorPos = int(relX / (ib.Options.FontSize * 0.6))
				if ib.CursorPos < 0 {
					ib.CursorPos = 0
				}
				if ib.CursorPos > len(ib.Value) {
					ib.CursorPos = len(ib.Value)
				}
				ib.SetNeedsRedraw(true)
				ctx.InvalidateRect(ib.Bounds())
				return true
			}
		}

	case event.MouseDoubleClick:
		// Select all on double click
		if e.Button == event.ButtonLeft {
			ib.CursorPos = len(ib.Value)
			ib.SetNeedsRedraw(true)
			ctx.InvalidateRect(ib.Bounds())
			return true
		}
	}

	return false
}

func (ib *InputBar) handleKey(ctx widget.Context, e *event.KeyEvent) bool {
	if e.KeyType == event.KeyRelease {
		return false
	}

	switch e.Key {
	case event.KeyEnter:
		if ib.Value != "" && ib.OnSubmit != nil {
			ib.OnSubmit(ib.Value)
			ib.Value = ""
			ib.CursorPos = 0
			ib.SetNeedsRedraw(true)
			ctx.InvalidateRect(ib.Bounds())
		}
		return true

	case event.KeyBackspace:
		if ib.CursorPos > 0 {
			ib.Value = ib.Value[:ib.CursorPos-1] + ib.Value[ib.CursorPos:]
			ib.CursorPos--
			if ib.OnChange != nil {
				ib.OnChange(ib.Value)
			}
			ib.SetNeedsRedraw(true)
			ctx.InvalidateRect(ib.Bounds())
		}
		return true

	case event.KeyDelete:
		if ib.CursorPos < len(ib.Value) {
			ib.Value = ib.Value[:ib.CursorPos] + ib.Value[ib.CursorPos+1:]
			if ib.OnChange != nil {
				ib.OnChange(ib.Value)
			}
			ib.SetNeedsRedraw(true)
			ctx.InvalidateRect(ib.Bounds())
		}
		return true

	case event.KeyLeft:
		if ib.CursorPos > 0 {
			ib.CursorPos--
			ib.SetNeedsRedraw(true)
			ctx.InvalidateRect(ib.Bounds())
		}
		return true

	case event.KeyRight:
		if ib.CursorPos < len(ib.Value) {
			ib.CursorPos++
			ib.SetNeedsRedraw(true)
			ctx.InvalidateRect(ib.Bounds())
		}
		return true

	case event.KeyHome:
		ib.CursorPos = 0
		ib.SetNeedsRedraw(true)
		ctx.InvalidateRect(ib.Bounds())
		return true

	case event.KeyEnd:
		ib.CursorPos = len(ib.Value)
		ib.SetNeedsRedraw(true)
		ctx.InvalidateRect(ib.Bounds())
		return true
	}

	// Printable characters
	if e.Rune != 0 && !e.IsCtrl() && !e.IsAlt() && !e.IsSuper() {
		ib.Value = ib.Value[:ib.CursorPos] + string(e.Rune) + ib.Value[ib.CursorPos:]
		ib.CursorPos++
		if ib.OnChange != nil {
			ib.OnChange(ib.Value)
		}
		ib.SetNeedsRedraw(true)
		ctx.InvalidateRect(ib.Bounds())
		return true
	}

	return false
}

// InputBar Colors (JetBrains Dark)
var (
	InputBarBackgroundColor  = widget.RGBA8(0x1E, 0x1F, 0x22, 0xFF) // Gray1
	InputBarBorderColor      = widget.RGBA8(0x4E, 0x51, 0x57, 0xFF) // Gray5
	InputBarTextColor        = widget.RGBA8(0xDF, 0xE1, 0xE5, 0xFF) // Gray12
	InputBarPlaceholderColor = widget.RGBA8(0x6F, 0x73, 0x7A, 0xFF) // Gray7
	InputBarIconColor        = widget.RGBA8(0x9D, 0xA0, 0xA8, 0xFF) // Gray9
	InputBarCursorColor      = widget.RGBA8(0xDF, 0xE1, 0xE5, 0xFF) // Gray12
	InputBarSendColor        = widget.RGBA8(0x35, 0x74, 0xF0, 0xFF) // Blue6
)
