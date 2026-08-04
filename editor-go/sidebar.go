package editor

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// TreeItem represents a node in the project tree.
type TreeItem struct {
	Label      string
	IsDir      bool
	Children   []*TreeItem
	IsExpanded bool
	Depth      int
	FilePath   string
}

// Sidebar is a gogpu/ui widget that renders a VS Code-style left sidebar
// with a project tree view, section headers, and expandable/collapsible nodes.
type Sidebar struct {
	widget.WidgetBase

	// Items is the list of tree items to display.
	Items []*TreeItem

	// SelectedItem is the currently selected tree item (-1 if none).
	SelectedItem int

	// HoveredItem is the hovered tree item (-1 if none).
	HoveredItem int

	// OnItemSelected is called when a tree item is clicked.
	OnItemSelected func(item *TreeItem)

	// OnItemExpanded is called when a directory is expanded/collapsed.
	OnItemExpanded func(item *TreeItem, expanded bool)

	// Options carries visual configuration.
	Options SidebarOptions

	// scrollY is the vertical scroll offset.
	scrollY float32

	// itemRects stores computed layout rectangles for each visible item.
	itemRects []geometry.Rect

	// flattenedItems is the visible items after expansion state is applied.
	flattenedItems []*TreeItem
}

// SidebarOptions configures the appearance of Sidebar.
type SidebarOptions struct {
	Width         float32
	FontSize      float32
	RowHeight     float32
	IndentWidth   float32
	IconSize      float32
	PaddingLeft   float32
	PaddingRight  float32
	HeaderHeight  float32
	SectionGap    float32
}

// DefaultSidebarOptions returns sensible VS Code-like defaults.
func DefaultSidebarOptions() SidebarOptions {
	return SidebarOptions{
		Width:        240,
		FontSize:     13,
		RowHeight:    22,
		IndentWidth:  12,
		IconSize:     16,
		PaddingLeft:  8,
		PaddingRight: 8,
		HeaderHeight: 22,
		SectionGap:   8,
	}
}

// NewSidebar creates a new Sidebar widget.
func NewSidebar() *Sidebar {
	return &Sidebar{
		SelectedItem: -1,
		HoveredItem:  -1,
		Options:      DefaultSidebarOptions(),
	}
}

// SetItems replaces the tree items and triggers relayout.
func (s *Sidebar) SetItems(items []*TreeItem) {
	s.Items = items
	s.flattenItems()
	s.SetNeedsRedraw(true)
}

// flattenItems builds the visible list by walking the tree and respecting
// expansion state.
func (s *Sidebar) flattenItems() {
	s.flattenedItems = nil
	var walk func(items []*TreeItem, depth int)
	walk = func(items []*TreeItem, depth int) {
		for _, item := range items {
			item.Depth = depth
			s.flattenedItems = append(s.flattenedItems, item)
			if item.IsDir && item.IsExpanded && len(item.Children) > 0 {
				walk(item.Children, depth+1)
			}
		}
	}
	walk(s.Items, 0)
}

// Layout implements widget.Widget.
func (s *Sidebar) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	return geometry.Size{Width: s.Options.Width, Height: c.MaxHeight}
}

// Children implements widget.Widget.
func (s *Sidebar) Children() []widget.Widget { return nil }

// Draw implements widget.Widget.
func (s *Sidebar) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := s.Bounds()
	size := bounds.Size()
	if size.Width <= 0 || size.Height <= 0 {
		return
	}

	// Background
	canvas.DrawRect(geometry.Rect{Min: geometry.Pt(0, 0), Max: size.ToPoint()}, SidebarBackgroundColor)

	// Section header: "EXPLORER"
	headerH := s.Options.HeaderHeight
	canvas.DrawRect(geometry.Rect{
		Min: geometry.Pt(0, 0),
		Max: geometry.Pt(size.Width, headerH),
	}, SidebarHeaderBackgroundColor)
	canvas.DrawText("EXPLORER", geometry.Rect{
		Min: geometry.Pt(s.Options.PaddingLeft, 0),
		Max: geometry.Pt(size.Width-s.Options.PaddingRight, headerH),
	}, s.Options.FontSize-1, SidebarHeaderColor, false, widget.TextAlignLeft)

	// Tree content area
	contentTop := headerH + s.Options.SectionGap
	contentBottom := size.Height

	canvas.PushClip(geometry.Rect{
		Min: geometry.Pt(0, contentTop),
		Max: size.ToPoint(),
	})

	// Draw tree items
	y := contentTop - s.scrollY
	for i, item := range s.flattenedItems {
		if y+ s.Options.RowHeight < contentTop-s.Options.RowHeight {
			// Item is above visible area, skip
			y += s.Options.RowHeight
			continue
		}
		if y > contentBottom {
			break
		}

		rowRect := geometry.Rect{
			Min: geometry.Pt(0, y),
			Max: geometry.Pt(size.Width, y+s.Options.RowHeight),
		}

		// Store rect for hit testing
		if i < len(s.itemRects) {
			s.itemRects[i] = rowRect
		}

		// Selection/hover highlight
		if i == s.SelectedItem {
			canvas.DrawRect(rowRect, SidebarSelectedBackgroundColor)
		} else if i == s.HoveredItem {
			canvas.DrawRect(rowRect, SidebarHoverBackgroundColor)
		}

		// Indentation
		indentX := s.Options.PaddingLeft + float32(item.Depth)*s.Options.IndentWidth

		// Expand/collapse arrow for directories
		if item.IsDir {
			arrowX := indentX
			arrowY := y + s.Options.RowHeight/2
			arrowSize := float32(8)
			if item.IsExpanded {
				// Down arrow ▼
				canvas.DrawText("▼", geometry.Rect{
					Min: geometry.Pt(arrowX, y),
					Max: geometry.Pt(arrowX+arrowSize+4, y+s.Options.RowHeight),
				}, s.Options.FontSize-2, SidebarArrowColor, false, widget.TextAlignCenter)
			} else {
				// Right arrow ▶
				canvas.DrawText("▶", geometry.Rect{
					Min: geometry.Pt(arrowX, y),
					Max: geometry.Pt(arrowX+arrowSize+4, y+s.Options.RowHeight),
				}, s.Options.FontSize-2, SidebarArrowColor, false, widget.TextAlignCenter)
			}
			_ = arrowY
			indentX += arrowSize + 4
		}

		// Icon placeholder (simple document/folder shape)
		iconX := indentX
		iconY := y + (s.Options.RowHeight-s.Options.IconSize)/2
		iconColor := SidebarIconColor
		if item.IsDir {
			iconColor = SidebarFolderIconColor
		}
		s.drawIcon(canvas, iconX, iconY, s.Options.IconSize, iconColor, item.IsDir)
		indentX += s.Options.IconSize + 4

		// Label text
		textColor := SidebarTextColor
		if i == s.SelectedItem {
			textColor = SidebarSelectedTextColor
		}
		canvas.DrawText(item.Label, geometry.Rect{
			Min: geometry.Pt(indentX, y),
			Max: geometry.Pt(size.Width-s.Options.PaddingRight, y+s.Options.RowHeight),
		}, s.Options.FontSize, textColor, false, widget.TextAlignLeft)

		y += s.Options.RowHeight
	}

	canvas.PopClip()
}

// drawIcon draws a simple folder or file icon.
func (s *Sidebar) drawIcon(canvas widget.Canvas, x, y, size float32, c widget.Color, isDir bool) {
	if isDir {
		// Folder shape
		body := geometry.Rect{
			Min: geometry.Pt(x, y+size*0.2),
			Max: geometry.Pt(x+size, y+size),
		}
		canvas.DrawRect(body, c)
		// Folder tab
		tab := geometry.Rect{
			Min: geometry.Pt(x, y),
			Max: geometry.Pt(x+size*0.4, y+size*0.3),
		}
		canvas.DrawRect(tab, c)
	} else {
		// File shape (document with folded corner)
		body := geometry.Rect{
			Min: geometry.Pt(x+size*0.1, y),
			Max: geometry.Pt(x+size, y+size),
		}
		canvas.StrokeRect(body, c, 1.0)
		// Fold corner
		fold := geometry.Rect{
			Min: geometry.Pt(x+size*0.6, y),
			Max: geometry.Pt(x+size, y+size*0.3),
		}
		canvas.DrawRect(fold, SidebarBackgroundColor)
	}
}

// Event implements widget.Widget.
func (s *Sidebar) Event(ctx widget.Context, e event.Event) bool {
	switch ev := e.(type) {
	case *event.MouseEvent:
		return s.handleMouse(ctx, ev)
	}
	return false
}

func (s *Sidebar) handleMouse(ctx widget.Context, e *event.MouseEvent) bool {
	local := s.GlobalToLocal(e.GlobalPosition)

	switch e.MouseType {
	case event.MouseMove:
		idx := s.itemAtPosition(local)
		if idx != s.HoveredItem {
			s.HoveredItem = idx
			s.SetNeedsRedraw(true)
			ctx.InvalidateRect(s.Bounds())
		}
		return idx >= 0

	case event.MousePress:
		if e.Button != event.ButtonLeft {
			return false
		}
		idx := s.itemAtPosition(local)
		if idx >= 0 && idx < len(s.flattenedItems) {
			item := s.flattenedItems[idx]
			if item.IsDir {
				// Toggle expansion
				item.IsExpanded = !item.IsExpanded
				s.flattenItems()
				if s.OnItemExpanded != nil {
					s.OnItemExpanded(item, item.IsExpanded)
				}
			} else {
				// Select file
				s.SelectedItem = idx
				if s.OnItemSelected != nil {
					s.OnItemSelected(item)
				}
			}
			s.SetNeedsRedraw(true)
			ctx.InvalidateRect(s.Bounds())
			return true
		}
	}

	return false
}

func (s *Sidebar) itemAtPosition(pos geometry.Point) int {
	headerH := s.Options.HeaderHeight + s.Options.SectionGap
	y := headerH - s.scrollY
	for i := range s.flattenedItems {
		if pos.Y >= y && pos.Y < y+s.Options.RowHeight {
			return i
		}
		y += s.Options.RowHeight
	}
	return -1
}

// ScrollBy scrolls the sidebar content vertically.
func (s *Sidebar) ScrollBy(deltaY float32) {
	s.scrollY += deltaY
	if s.scrollY < 0 {
		s.scrollY = 0
	}
	maxScroll := float32(len(s.flattenedItems))*s.Options.RowHeight - (s.Bounds().Height() - s.Options.HeaderHeight - s.Options.SectionGap)
	if maxScroll < 0 {
		maxScroll = 0
	}
	if s.scrollY > maxScroll {
		s.scrollY = maxScroll
	}
	s.SetNeedsRedraw(true)
}

// Sidebar Colors (JetBrains Dark)
var (
	SidebarBackgroundColor        = widget.RGBA8(0x2B, 0x2D, 0x30, 0xFF) // Gray2
	SidebarHeaderBackgroundColor  = widget.RGBA8(0x39, 0x3B, 0x40, 0xFF) // Gray3
	SidebarHeaderColor            = widget.RGBA8(0x9D, 0xA0, 0xA8, 0xFF) // Gray9
	SidebarTextColor              = widget.RGBA8(0xDF, 0xE1, 0xE5, 0xFF) // Gray12
	SidebarSelectedTextColor      = widget.RGBA8(0xFF, 0xFF, 0xFF, 0xFF)
	SidebarSelectedBackgroundColor = widget.RGBA8(0x2E, 0x43, 0x6E, 0xFF) // Blue2
	SidebarHoverBackgroundColor   = widget.RGBA8(0x39, 0x3B, 0x40, 0xFF) // Gray3
	SidebarIconColor              = widget.RGBA8(0x9D, 0xA0, 0xA8, 0xFF) // Gray9
	SidebarFolderIconColor        = widget.RGBA8(0xDC, 0xAA, 0x2F, 0xFF) // Yellow
	SidebarArrowColor             = widget.RGBA8(0x6F, 0x73, 0x7A, 0xFF) // Gray7
)
