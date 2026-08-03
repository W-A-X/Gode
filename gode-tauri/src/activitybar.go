// Package activitybar provides the bottom activity bar widget for Gode Editor.
package activitybar

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// ActivityItem represents a single item in the activity bar
type ActivityItem struct {
	ID       string
	Icon     string
	Tooltip  string
	OnClick  func()
	selected bool
	bounds   geometry.Rect
}

// ActivityBar is the bottom activity bar widget
type ActivityBar struct {
	widget.WidgetBase
	items      []*ActivityItem
	selectedID string
	itemWidth  float32
	onSelect   func(id string)
}

// NewActivityBar creates a new activity bar with default items
func NewActivityBar() *ActivityBar {
	bar := &ActivityBar{
		items:     make([]*ActivityItem, 0),
		itemWidth: 48,
	}
	
	// Add default activity items
	bar.AddItem(&ActivityItem{
		ID:      "explorer",
		Icon:    "📁",
		Tooltip: "Explorer",
	})
	bar.AddItem(&ActivityItem{
		ID:      "search",
		Icon:    "🔍",
		Tooltip: "Search",
	})
	bar.AddItem(&ActivityItem{
		ID:      "source-control",
		Icon:    "🌿",
		Tooltip: "Source Control",
	})
	bar.AddItem(&ActivityItem{
		ID:      "run",
		Icon:    "▶️",
		Tooltip: "Run and Debug",
	})
	bar.AddItem(&ActivityItem{
		ID:      "extensions",
		Icon:    "🧩",
		Tooltip: "Extensions",
	})
	bar.AddItem(&ActivityItem{
		ID:      "settings",
		Icon:    "⚙️",
		Tooltip: "Settings",
	})
	
	// Select first item by default
	if len(bar.items) > 0 {
		bar.selectedID = bar.items[0].ID
		bar.items[0].selected = true
	}
	
	return bar
}

// SetOnSelect sets the callback for when an item is selected
func (b *ActivityBar) SetOnSelect(fn func(id string)) {
	b.onSelect = fn
}

// AddItem adds an activity item to the bar
func (b *ActivityBar) AddItem(item *ActivityItem) {
	b.items = append(b.items, item)
	b.SetNeedsLayout(true)
}

// Select selects an item by ID
func (b *ActivityBar) Select(id string) {
	for _, item := range b.items {
		item.selected = item.ID == id
	}
	b.selectedID = id
	b.SetNeedsRedraw(true)
	
	if b.onSelect != nil {
		b.onSelect(id)
	}
}

// GetSelected returns the currently selected item ID
func (b *ActivityBar) GetSelected() string {
	return b.selectedID
}

// Layout implements widget.Widget
func (b *ActivityBar) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	size := c.Biggest()
	
	// Layout items horizontally
	x := float32(0)
	itemHeight := size.Height
	
	for _, item := range b.items {
		item.bounds = geometry.Rect{
			Min: geometry.Pt(x, 0),
			Max: geometry.Pt(x+b.itemWidth, itemHeight),
		}
		x += b.itemWidth
	}
	
	return size
}

// Children implements widget.Widget
func (b *ActivityBar) Children() []widget.Widget {
	return nil
}

// Draw implements widget.Widget
func (b *ActivityBar) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := b.Bounds()
	size := bounds.Size()
	
	// Background
	bgColor := widget.RGBA8(0x25, 0x25, 0x26, 0xFF)
	canvas.DrawRect(bounds, bgColor)
	
	// Border top
	borderColor := widget.RGBA8(0x3C, 0x3C, 0x3C, 0xFF)
	canvas.DrawLine(
		geometry.Pt(0, 0),
		geometry.Pt(size.Width, 0),
		borderColor,
		1,
	)
	
	// Draw items
	for _, item := range b.items {
		b.drawItem(canvas, item)
	}
}

func (b *ActivityBar) drawItem(canvas widget.Canvas, item *ActivityItem) {
	bounds := item.bounds
	
	// Hover background (simplified - would need mouse tracking for real hover)
	hoverColor := widget.RGBA8(0x2A, 0x2D, 0x2E, 0xFF)
	
	// Selection indicator
	if item.selected {
		// Selected border bottom
		selectColor := widget.RGBA8(0x00, 0x7A, 0xCC, 0xFF)
		canvas.DrawLine(
			geometry.Pt(bounds.Min.X, bounds.Max.Y-1),
			geometry.Pt(bounds.Max.X, bounds.Max.Y-1),
			selectColor,
			2,
		)
	}
	
	// Icon
	iconColor := widget.RGBA8(0x85, 0x85, 0x85, 0xFF)
	if item.selected {
		iconColor = widget.RGBA8(0xFF, 0xFF, 0xFF, 0xFF)
	}
	
	centerX := (bounds.Min.X + bounds.Max.X) / 2
	centerY := (bounds.Min.Y + bounds.Max.Y) / 2
	
	canvas.DrawText(
		item.Icon,
		geometry.Rect{
			Min: geometry.Pt(centerX-12, centerY-12),
			Max: geometry.Pt(centerX+12, centerY+12),
		},
		24,
		iconColor,
		false,
		widget.TextAlignCenter,
	)
}

// Event implements widget.Widget
func (b *ActivityBar) Event(ctx widget.Context, e event.Event) bool {
	switch ev := e.(type) {
	case *event.MouseEvent:
		return b.handleMouse(ev)
	}
	return false
}

func (b *ActivityBar) handleMouse(e *event.MouseEvent) bool {
	if e.KeyType != event.KeyRelease {
		return false
	}
	
	switch e.MouseType {
	case event.MousePress:
		if e.Button != event.ButtonLeft {
			return false
		}
		
		local := e.Position // Already in local coordinates
		
		for _, item := range b.items {
			if item.bounds.Contains(local) {
				b.Select(item.ID)
				if item.OnClick != nil {
					item.OnClick()
				}
				return true
			}
		}
	}
	
	return false
}
