// Package layout provides the Go-based window layout system for Gode Editor.
// It implements a bottom activity bar layout instead of the traditional left sidebar.
package layout

import (
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// LayoutType defines the type of layout region
type LayoutType int

const (
	LayoutEditor LayoutType = iota
	LayoutBottomBar
	LayoutStatusBar
	LayoutTitleBar
)

// LayoutRegion represents a named region in the layout
type LayoutRegion struct {
	Type     LayoutType
	Bounds   geometry.Rect
	Widget   widget.Widget
	Priority int // Higher priority regions are laid out first
}

// WindowLayout manages the overall window layout with bottom activity bar
type WindowLayout struct {
	widget.WidgetBase
	regions    []*LayoutRegion
	titleBarH  float32
	bottomBarH float32
	statusBarH float32
}

// NewWindowLayout creates a new window layout with default dimensions
func NewWindowLayout() *WindowLayout {
	return &WindowLayout{
		regions:    make([]*LayoutRegion, 0),
		titleBarH:  30,
		bottomBarH: 48,
		statusBarH: 22,
	}
}

// SetTitleBarHeight sets the height of the title bar
func (l *WindowLayout) SetTitleBarHeight(h float32) {
	l.titleBarH = h
	l.SetNeedsLayout(true)
}

// SetBottomBarHeight sets the height of the bottom activity bar
func (l *WindowLayout) SetBottomBarHeight(h float32) {
	l.bottomBarH = h
	l.SetNeedsLayout(true)
}

// SetStatusBarHeight sets the height of the status bar
func (l *WindowLayout) SetStatusBarHeight(h float32) {
	l.statusBarH = h
	l.SetNeedsLayout(true)
}

// AddRegion adds a new layout region
func (l *WindowLayout) AddRegion(typ LayoutType, w widget.Widget, priority int) {
	l.regions = append(l.regions, &LayoutRegion{
		Type:     typ,
		Widget:   w,
		Priority: priority,
	})
}

// GetRegion returns the region of the specified type
func (l *WindowLayout) GetRegion(typ LayoutType) *LayoutRegion {
	for _, r := range l.regions {
		if r.Type == typ {
			return r
		}
	}
	return nil
}

// Layout implements widget.Widget
func (l *WindowLayout) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	size := c.Biggest()
	
	y := float32(0)
	
	// Title bar at top
	if titleRegion := l.GetRegion(LayoutTitleBar); titleRegion != nil {
		titleRegion.Bounds = geometry.Rect{
			Min: geometry.Pt(0, y),
			Max: geometry.Pt(size.Width, y+l.titleBarH),
		}
		y += l.titleBarH
	}
	
	// Editor area fills remaining space minus bottom bars
	editorBottom := size.Height - l.bottomBarH - l.statusBarH
	
	if editorRegion := l.GetRegion(LayoutEditor); editorRegion != nil {
		editorRegion.Bounds = geometry.Rect{
			Min: geometry.Pt(0, y),
			Max: geometry.Pt(size.Width, editorBottom),
		}
		y = editorBottom
	}
	
	// Bottom activity bar
	if bottomRegion := l.GetRegion(LayoutBottomBar); bottomRegion != nil {
		bottomRegion.Bounds = geometry.Rect{
			Min: geometry.Pt(0, y),
			Max: geometry.Pt(size.Width, y+l.bottomBarH),
		}
		y += l.bottomBarH
	}
	
	// Status bar at very bottom
	if statusRegion := l.GetRegion(LayoutStatusBar); statusRegion != nil {
		statusRegion.Bounds = geometry.Rect{
			Min: geometry.Pt(0, y),
			Max: geometry.Pt(size.Width, y+l.statusBarH),
		}
	}
	
	// Layout child widgets
	for _, region := range l.regions {
		if region.Widget != nil {
			childConstraints := geometry.Constraints{
				Min: region.Bounds.Size().ToPoint(),
				Max: region.Bounds.Size().ToPoint(),
			}
			region.Widget.Layout(ctx, childConstraints)
			region.Widget.SetBounds(region.Bounds)
		}
	}
	
	return size
}

// Children implements widget.Widget
func (l *WindowLayout) Children() []widget.Widget {
	children := make([]widget.Widget, 0, len(l.regions))
	for _, r := range l.regions {
		if r.Widget != nil {
			children = append(children, r.Widget)
		}
	}
	return children
}

// Draw implements widget.Widget
func (l *WindowLayout) Draw(ctx widget.Context, canvas widget.Canvas) {
	// Background
	canvas.DrawRect(l.Bounds(), widget.RGBA8(0x1E, 0x1E, 0x1E, 0xFF))
	
	// Children are drawn automatically by the framework
}
