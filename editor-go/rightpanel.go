package editor

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// FileItem represents a file or directory in the right panel explorer.
type FileItem struct {
	Name      string
	IsDir     bool
	Extension string
	FilePath  string
	Children  []*FileItem
}

// RightPanel is a gogpu/ui widget that renders a VS Code-style right sidebar
// with a file explorer showing the project structure in a flat or tree view.
type RightPanel struct {
	widget.WidgetBase

	// Items is the list of file items to display.
	Items []*FileItem

	// SelectedIndex is the currently selected item (-1 if none).
	SelectedIndex int

	// HoveredIndex is the hovered item (-1 if none).
	HoveredIndex int

	// OnFileSelected is called when a file is clicked.
	OnFileSelected func(item *FileItem)

	// OnFileDoubleClicked is called when a file is double-clicked.
	OnFileDoubleClicked func(item *FileItem)

	// Options carries visual configuration.
	Options RightPanelOptions

	// scrollY is the vertical scroll offset.
	scrollY float32

	// itemRects stores computed layout rectangles.
	itemRects []geometry.Rect
}

// RightPanelOptions configures the appearance of RightPanel.
type RightPanelOptions struct {
	Width        float32
	FontSize     float32
	RowHeight    float32
	IconSize     float32
	PaddingLeft  float32
	PaddingRight float32
	HeaderHeight float32
}

// DefaultRightPanelOptions returns sensible defaults.
func DefaultRightPanelOptions() RightPanelOptions {
	return RightPanelOptions{
		Width:        200,
		FontSize:     12,
		RowHeight:    20,
		IconSize:     14,
		PaddingLeft:  8,
		PaddingRight: 8,
		HeaderHeight: 22,
	}
}

// NewRightPanel creates a new RightPanel widget.
func NewRightPanel() *RightPanel {
	return &RightPanel{
		SelectedIndex: -1,
		HoveredIndex:  -1,
		Options:       DefaultRightPanelOptions(),
	}
}

// SetItems replaces the file items and triggers relayout.
func (p *RightPanel) SetItems(items []*FileItem) {
	p.Items = items
	p.recalcLayout()
	p.SetNeedsRedraw(true)
}

// recalcLayout computes item rectangles.
func (p *RightPanel) recalcLayout() {
	bounds := p.Bounds()
	size := bounds.Size()
	n := len(p.Items)
	p.itemRects = make([]geometry.Rect, n)

	y := p.Options.HeaderHeight - p.scrollY
	for i := range p.Items {
		p.itemRects[i] = geometry.Rect{
			Min: geometry.Pt(0, y),
			Max: geometry.Pt(size.Width, y+p.Options.RowHeight),
		}
		y += p.Options.RowHeight
	}
}

// Layout implements widget.Widget.
func (p *RightPanel) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	return geometry.Size{Width: p.Options.Width, Height: c.MaxHeight}
}

// Children implements widget.Widget.
func (p *RightPanel) Children() []widget.Widget { return nil }

// Draw implements widget.Widget.
func (p *RightPanel) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := p.Bounds()
	size := bounds.Size()
	if size.Width <= 0 || size.Height <= 0 {
		return
	}

	// Background
	canvas.DrawRect(geometry.Rect{Min: geometry.Pt(0, 0), Max: size.ToPoint()}, RightPanelBackgroundColor)

	// Section header: "Gode"
	headerH := p.Options.HeaderHeight
	canvas.DrawRect(geometry.Rect{
		Min: geometry.Pt(0, 0),
		Max: geometry.Pt(size.Width, headerH),
	}, RightPanelHeaderBackgroundColor)
	canvas.DrawText("Gode", geometry.Rect{
		Min: geometry.Pt(p.Options.PaddingLeft, 0),
		Max: geometry.Pt(size.Width-p.Options.PaddingRight, headerH),
	}, p.Options.FontSize-1, RightPanelHeaderColor, false, widget.TextAlignLeft)

	// File list content area
	contentTop := headerH

	canvas.PushClip(geometry.Rect{
		Min: geometry.Pt(0, contentTop),
		Max: size.ToPoint(),
	})

	// Draw file items
	for i, item := range p.Items {
		if i >= len(p.itemRects) {
			break
		}
		r := p.itemRects[i]

		// Skip items outside visible area
		if r.Max.Y < contentTop-p.Options.RowHeight || r.Min.Y > size.Height {
			continue
		}

		// Selection/hover highlight
		if i == p.SelectedIndex {
			canvas.DrawRect(r, RightPanelSelectedBackgroundColor)
		} else if i == p.HoveredIndex {
			canvas.DrawRect(r, RightPanelHoverBackgroundColor)
		}

		// Icon
		iconX := p.Options.PaddingLeft
		iconY := r.Min.Y + (r.Height()-p.Options.IconSize)/2
		iconColor := RightPanelFileIconColor
		if item.IsDir {
			iconColor = RightPanelFolderIconColor
		}
		p.drawItemIcon(canvas, iconX, iconY, p.Options.IconSize, iconColor, item)
		textX := iconX + p.Options.IconSize + 4

		// Label text
		textColor := RightPanelTextColor
		if i == p.SelectedIndex {
			textColor = RightPanelSelectedTextColor
		}
		canvas.DrawText(item.Name, geometry.Rect{
			Min: geometry.Pt(textX, r.Min.Y),
			Max: geometry.Pt(size.Width-p.Options.PaddingRight, r.Max.Y),
		}, p.Options.FontSize, textColor, false, widget.TextAlignLeft)
	}

	canvas.PopClip()
}

// drawItemIcon draws a file or folder icon.
func (p *RightPanel) drawItemIcon(canvas widget.Canvas, x, y, size float32, c widget.Color, item *FileItem) {
	if item.IsDir {
		// Folder
		body := geometry.Rect{
			Min: geometry.Pt(x, y+size*0.2),
			Max: geometry.Pt(x+size, y+size),
		}
		canvas.DrawRect(body, c)
		tab := geometry.Rect{
			Min: geometry.Pt(x, y),
			Max: geometry.Pt(x+size*0.4, y+size*0.3),
		}
		canvas.DrawRect(tab, c)
	} else {
		// File with extension-based color
		fileColor := RightPanelFileIconColor
		switch item.Extension {
		case "go":
			fileColor = RightPanelGoFileColor
		case "ts", "js":
			fileColor = RightPanelJSFileColor
		case "json":
			fileColor = RightPanelJSONFileColor
		case "md":
			fileColor = RightPanelMDFileColor
		}

		body := geometry.Rect{
			Min: geometry.Pt(x+size*0.1, y),
			Max: geometry.Pt(x+size, y+size),
		}
		canvas.StrokeRect(body, fileColor, 1.0)
		fold := geometry.Rect{
			Min: geometry.Pt(x+size*0.6, y),
			Max: geometry.Pt(x+size, y+size*0.3),
		}
		canvas.DrawRect(fold, RightPanelBackgroundColor)
	}
}

// Event implements widget.Widget.
func (p *RightPanel) Event(ctx widget.Context, e event.Event) bool {
	switch ev := e.(type) {
	case *event.MouseEvent:
		return p.handleMouse(ctx, ev)
	}
	return false
}

func (p *RightPanel) handleMouse(ctx widget.Context, e *event.MouseEvent) bool {
	local := p.GlobalToLocal(e.GlobalPosition)

	switch e.MouseType {
	case event.MouseMove:
		idx := p.itemAtPosition(local)
		if idx != p.HoveredIndex {
			p.HoveredIndex = idx
			p.SetNeedsRedraw(true)
			ctx.InvalidateRect(p.Bounds())
		}
		return idx >= 0

	case event.MousePress:
		if e.Button != event.ButtonLeft {
			return false
		}
		idx := p.itemAtPosition(local)
		if idx >= 0 && idx < len(p.Items) {
			p.SelectedIndex = idx
			if p.OnFileSelected != nil {
				p.OnFileSelected(p.Items[idx])
			}
			p.SetNeedsRedraw(true)
			ctx.InvalidateRect(p.Bounds())
			return true
		}

	case event.MouseDoubleClick:
		if e.Button != event.ButtonLeft {
			return false
		}
		idx := p.itemAtPosition(local)
		if idx >= 0 && idx < len(p.Items) {
			if p.OnFileDoubleClicked != nil {
				p.OnFileDoubleClicked(p.Items[idx])
			}
			return true
		}
	}

	return false
}

func (p *RightPanel) itemAtPosition(pos geometry.Point) int {
	headerH := p.Options.HeaderHeight
	y := headerH - p.scrollY
	for i := range p.Items {
		if pos.Y >= y && pos.Y < y+p.Options.RowHeight {
			return i
		}
		y += p.Options.RowHeight
	}
	return -1
}

// ScrollBy scrolls the panel content vertically.
func (p *RightPanel) ScrollBy(deltaY float32) {
	p.scrollY += deltaY
	if p.scrollY < 0 {
		p.scrollY = 0
	}
	maxScroll := float32(len(p.Items))*p.Options.RowHeight - (p.Bounds().Height() - p.Options.HeaderHeight)
	if maxScroll < 0 {
		maxScroll = 0
	}
	if p.scrollY > maxScroll {
		p.scrollY = maxScroll
	}
	p.SetNeedsRedraw(true)
}

// RightPanel Colors (JetBrains Dark)
var (
	RightPanelBackgroundColor       = widget.RGBA8(0x2B, 0x2D, 0x30, 0xFF) // Gray2
	RightPanelHeaderBackgroundColor = widget.RGBA8(0x39, 0x3B, 0x40, 0xFF) // Gray3
	RightPanelHeaderColor           = widget.RGBA8(0xDF, 0xE1, 0xE5, 0xFF) // Gray12
	RightPanelTextColor             = widget.RGBA8(0xDF, 0xE1, 0xE5, 0xFF) // Gray12
	RightPanelSelectedTextColor     = widget.RGBA8(0xFF, 0xFF, 0xFF, 0xFF)
	RightPanelSelectedBackgroundColor = widget.RGBA8(0x2E, 0x43, 0x6E, 0xFF) // Blue2
	RightPanelHoverBackgroundColor  = widget.RGBA8(0x39, 0x3B, 0x40, 0xFF) // Gray3
	RightPanelFileIconColor         = widget.RGBA8(0x9D, 0xA0, 0xA8, 0xFF) // Gray9
	RightPanelFolderIconColor       = widget.RGBA8(0xDC, 0xAA, 0x2F, 0xFF) // Yellow
	RightPanelGoFileColor           = widget.RGBA8(0x00, 0xAD, 0xD8, 0xFF) // Cyan
	RightPanelJSFileColor           = widget.RGBA8(0xF7, 0xDF, 0x1E, 0xFF) // Yellow
	RightPanelJSONFileColor         = widget.RGBA8(0xCB, 0xCB, 0x42, 0xFF) // Olive
	RightPanelMDFileColor           = widget.RGBA8(0x51, 0x9A, 0xBA, 0xFF) // Blue
)
