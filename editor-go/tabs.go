// Package editor implements the core rendering layer of a VS Code-style
// code editor in Go, rendered with the gogpu/ui toolkit.
//
// This file provides the tab bar widget for displaying multiple open editors.
package editor

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// TabInfo describes a single tab in the tab bar.
type TabInfo struct {
	ID       string
	Title    string
	Subtitle string // Optional description (e.g., file path)
	Dirty    bool
	Active   bool
	Selected bool // Part of multi-selection
	Pinned   bool   // Sticky/pinned tab
	Icon     rune   // Optional icon rune
}

// TabBarAction represents an action button on a tab (e.g., close button).
type TabBarAction struct {
	ID   string
	Icon rune
	Hint string
}

// TabBarOptions configures the appearance of the tab bar.
type TabBarOptions struct {
	// FontSize is the font size in logical pixels.
	FontSize float32

	// TabHeight is the height of each tab in logical pixels.
	TabHeight float32

	// TabMinWidth is the minimum width of a tab.
	TabMinWidth float32

	// TabMaxWidth is the maximum width of a tab (for truncation).
	TabMaxWidth float32

	// PaddingLeft / PaddingRight are horizontal insets.
	PaddingLeft  float32
	PaddingRight float32

	// Gap between tabs.
	TabGap float32

	// ShowCloseButton shows a close button on each tab when true.
	ShowCloseButton bool

	// CloseButtonIcon is the rune for the close button (default '×').
	CloseButtonIcon rune
}

// DefaultTabBarOptions returns VS Code-like defaults.
func DefaultTabBarOptions() TabBarOptions {
	return TabBarOptions{
		FontSize:        13,
		TabHeight:       35,
		TabMinWidth:     100,
		TabMaxWidth:     200,
		PaddingLeft:     10,
		PaddingRight:    10,
		TabGap:          2,
		ShowCloseButton: true,
		CloseButtonIcon: '×',
	}
}

// TabBar is a gogpu/ui widget that renders a row of editor tabs.
type TabBar struct {
	widget.WidgetBase

	// Tabs holds the list of tabs to display.
	Tabs []TabInfo

	// Options carries the view configuration.
	Options TabBarOptions

	// OnTabClick is called when a tab is clicked.
	OnTabClick func(id string)

	// OnTabClose is called when a tab's close button is clicked.
	OnTabClose func(id string)

	// OnTabContextMenu is called when a tab is right-clicked.
	OnTabContextMenu func(id string, x, y float32)

	// OnDragDrop is called when a tab is dragged and dropped.
	OnDragDrop func(fromID, toID string, before bool)

	// ScrollOffset is the horizontal scroll offset for overflow.
	ScrollOffset float32

	// hoveredTab is the ID of the currently hovered tab.
	hoveredTab string

	// hoveredClose is the ID of the tab whose close button is hovered.
	hoveredClose string

	// dragging is the ID of the tab being dragged.
	dragging string

	// dragStartX is the X position where the drag started.
	dragStartX float32

	// tabRects caches the computed rectangles for each tab.
	tabRects map[string]geometry.Rect

	// closeRects caches the computed rectangles for each close button.
	closeRects map[string]geometry.Rect

	lastCanvas widget.Canvas
}

// NewTabBar creates a new tab bar widget.
func NewTabBar(opts TabBarOptions) *TabBar {
	tb := &TabBar{
		Options:    opts,
		tabRects:   make(map[string]geometry.Rect),
		closeRects: make(map[string]geometry.Rect),
	}
	tb.SetVisible(true)
	tb.SetEnabled(true)
	return tb
}

// SetTabs updates the list of tabs and triggers a redraw.
func (tb *TabBar) SetTabs(tabs []TabInfo) {
	tb.Tabs = tabs
	tb.tabRects = make(map[string]geometry.Rect)
	tb.closeRects = make(map[string]geometry.Rect)
	tb.SetNeedsRedraw(true)
}

// Layout implements widget.Widget: the tab bar fills available width and uses fixed height.
func (tb *TabBar) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	height := tb.Options.TabHeight
	if height <= 0 {
		height = DefaultTabBarOptions().TabHeight
	}
	width := c.Max.Width
	if width < 0 {
		width = 500
	}
	return geometry.Size{Width: width, Height: height}
}

// Children implements widget.Widget. The tab bar is a leaf widget.
func (tb *TabBar) Children() []widget.Widget { return nil }

// Draw implements widget.Widget.
func (tb *TabBar) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := tb.Bounds()
	size := bounds.Size()
	if size.Width <= 0 || size.Height <= 0 {
		return
	}

	local := geometry.Rect{Min: geometry.Pt(0, 0), Max: size.ToPoint()}
	canvas.PushClip(local)

	tb.lastCanvas = canvas

	// Background
	canvas.DrawRect(local, tabBarBackgroundColor)

	x := tb.Options.PaddingLeft - tb.ScrollOffset
	tabHeight := size.Height

	tb.tabRects = make(map[string]geometry.Rect)
	tb.closeRects = make(map[string]geometry.Rect)

	for i, tab := range tb.Tabs {
		// Calculate tab width based on content
		titleWidth := canvas.MeasureText(tab.Title, tb.Options.FontSize, false)
		subtitleWidth := float32(0)
		if tab.Subtitle != "" {
			subtitleWidth = canvas.MeasureText(" • "+tab.Subtitle, tb.Options.FontSize-2, false)
		}
		iconWidth := float32(0)
		if tab.Icon != 0 {
			iconWidth = canvas.MeasureText(string(tab.Icon), tb.Options.FontSize, false) + 4
		}
		closeWidth := float32(0)
		if tb.Options.ShowCloseButton {
			closeWidth = canvas.MeasureText(string(tb.Options.CloseButtonIcon), tb.Options.FontSize, false) + 8
		}

		contentWidth := iconWidth + titleWidth + subtitleWidth + closeWidth
		tabWidth := contentWidth + tb.Options.PaddingLeft*2
		if tabWidth < tb.Options.TabMinWidth {
			tabWidth = tb.Options.TabMinWidth
		}
		if tabWidth > tb.Options.TabMaxWidth {
			tabWidth = tb.Options.TabMaxWidth
		}

		// Determine colors based on state
		bgColor := tabInactiveBackgroundColor
		textColor := tabInactiveForegroundColor
		borderColor := tabBorderColor

		if tab.Active {
			bgColor = tabActiveBackgroundColor
			textColor = tabActiveForegroundColor
			borderColor = tabActiveBorderColor
		} else if tab.Selected {
			bgColor = tabSelectedBackgroundColor
		}

		if tb.hoveredTab == tab.ID && !tab.Active {
			bgColor = tabHoverBackgroundColor
		}

		// Draw tab background
		tabRect := geometry.Rect{
			Min: geometry.Pt(x, 0),
			Max: geometry.Pt(x+tabWidth, tabHeight),
		}
		canvas.DrawRect(tabRect, bgColor)
		tb.tabRects[tab.ID] = tabRect

		// Draw top border for active tab
		if tab.Active {
			borderRect := geometry.Rect{
				Min: geometry.Pt(x, 0),
				Max: geometry.Pt(x+tabWidth, 2),
			}
			canvas.DrawRect(borderRect, borderColor)
		}

		// Draw separator between tabs
		if i > 0 && !tab.Active {
			sepRect := geometry.Rect{
				Min: geometry.Pt(x, 6),
				Max: geometry.Pt(x+1, tabHeight-6),
			}
			canvas.DrawRect(sepRect, tabSeparatorColor)
		}

		contentX := x + tb.Options.PaddingLeft
		textY := (tabHeight - tb.Options.FontSize) / 2

		// Draw icon
		if tab.Icon != 0 {
			canvas.DrawText(
				string(tab.Icon),
				geometry.Rect{Min: geometry.Pt(contentX, textY), Max: geometry.Pt(size.Width, textY+tb.Options.FontSize)},
				tb.Options.FontSize,
				textColor,
				false,
				widget.TextAlignLeft,
			)
			contentX += iconWidth
		}

		// Draw title (with truncation if needed)
		titleColor := textColor
		if tab.Dirty {
			// Draw dot indicator for dirty state
			dotX := contentX + titleWidth + 2
			dotY := textY + tb.Options.FontSize/2 - 2
			canvas.DrawCircle(geometry.Pt(dotX, dotY), 3, dirtyIndicatorColor)
		}

		// Truncate title if needed
		displayTitle := tab.Title
		maxTitleWidth := tabWidth - tb.Options.PaddingLeft*2 - iconWidth - subtitleWidth - closeWidth - 10
		if canvas.MeasureText(displayTitle, tb.Options.FontSize, false) > maxTitleWidth {
			for len(displayTitle) > 3 && canvas.MeasureText(displayTitle+"...", tb.Options.FontSize, false) > maxTitleWidth {
				displayTitle = displayTitle[:len(displayTitle)-1]
			}
			displayTitle += "..."
		}

		canvas.DrawText(
			displayTitle,
			geometry.Rect{Min: geometry.Pt(contentX, textY), Max: geometry.Pt(size.Width, textY+tb.Options.FontSize)},
			tb.Options.FontSize,
			titleColor,
			false,
			widget.TextAlignLeft,
		)
		contentX += titleWidth

		// Draw subtitle
		if tab.Subtitle != "" {
			canvas.DrawText(
				" • "+tab.Subtitle,
				geometry.Rect{Min: geometry.Pt(contentX, textY), Max: geometry.Pt(size.Width, textY+tb.Options.FontSize)},
				tb.Options.FontSize-2,
				tabSubtitleColor,
				false,
				widget.TextAlignLeft,
			)
		}

		// Draw close button
		if tb.Options.ShowCloseButton {
			closeX := x + tabWidth - tb.Options.PaddingRight - canvas.MeasureText(string(tb.Options.CloseButtonIcon), tb.Options.FontSize, false) - 4
			closeRect := geometry.Rect{
				Min: geometry.Pt(closeX, (tabHeight-tb.Options.FontSize)/2),
				Max: geometry.Pt(closeX+canvas.MeasureText(string(tb.Options.CloseButtonIcon), tb.Options.FontSize, false)+4, (tabHeight+tb.Options.FontSize)/2),
			}
			tb.closeRects[tab.ID] = closeRect

			closeBtnColor := tabCloseButtonColor
			if tb.hoveredClose == tab.ID {
				closeBtnColor = tabCloseButtonHoverColor
			}

			canvas.DrawText(
				string(tb.Options.CloseButtonIcon),
				closeRect,
				tb.Options.FontSize,
				closeBtnColor,
				false,
				widget.TextAlignCenter,
			)
		}

		x += tabWidth + tb.Options.TabGap
	}

	canvas.PopClip()
}

// Event implements widget.Widget.
func (tb *TabBar) Event(ctx widget.Context, e event.Event) bool {
	switch ev := e.(type) {
	case *event.MouseEvent:
		return tb.handleMouse(ctx, ev)
	}
	return false
}

func (tb *TabBar) handleMouse(ctx widget.Context, e *event.MouseEvent) bool {
	local := tb.GlobalToLocal(e.GlobalPosition)

	switch e.MouseType {
	case event.MouseMove:
		// Check which tab/close button is hovered
		newHoveredTab := ""
		newHoveredClose := ""

		for _, tab := range tb.Tabs {
			if rect, ok := tb.tabRects[tab.ID]; ok {
				if local.X >= rect.Min.X && local.X <= rect.Max.X &&
					local.Y >= rect.Min.Y && local.Y <= rect.Max.Y {
					newHoveredTab = tab.ID

					// Check if close button is hovered
					if closeRect, ok := tb.closeRects[tab.ID]; ok {
						if local.X >= closeRect.Min.X && local.X <= closeRect.Max.X &&
							local.Y >= closeRect.Min.Y && local.Y <= closeRect.Max.Y {
							newHoveredClose = tab.ID
						}
					}
					break
				}
			}
		}

		if newHoveredTab != tb.hoveredTab || newHoveredClose != tb.hoveredClose {
			tb.hoveredTab = newHoveredTab
			tb.hoveredClose = newHoveredClose
			tb.SetNeedsRedraw(true)
		}
		return tb.hoveredTab != ""

	case event.MousePress:
		if e.Button != event.ButtonLeft {
			return false
		}

		// Check if close button was clicked
		for _, tab := range tb.Tabs {
			if closeRect, ok := tb.closeRects[tab.ID]; ok {
				if local.X >= closeRect.Min.X && local.X <= closeRect.Max.X &&
					local.Y >= closeRect.Min.Y && local.Y <= closeRect.Max.Y {
					if tb.OnTabClose != nil {
						tb.OnTabClose(tab.ID)
					}
					return true
				}
			}
		}

		// Check if tab was clicked
		for _, tab := range tb.Tabs {
			if rect, ok := tb.tabRects[tab.ID]; ok {
				if local.X >= rect.Min.X && local.X <= rect.Max.X &&
					local.Y >= rect.Min.Y && local.Y <= rect.Max.Y {
					tb.dragging = tab.ID
					tb.dragStartX = local.X
					if tb.OnTabClick != nil {
						tb.OnTabClick(tab.ID)
					}
					return true
				}
			}
		}
		return false

	case event.MouseRelease:
		if tb.dragging != "" {
			// Check drop target
			dropTarget := ""
			dropBefore := false

			for _, tab := range tb.Tabs {
				if tab.ID == tb.dragging {
					continue
				}
				if rect, ok := tb.tabRects[tab.ID]; ok {
					midX := (rect.Min.X + rect.Max.X) / 2
					if local.X < midX {
						dropTarget = tab.ID
						dropBefore = true
						break
					} else if local.X < rect.Max.X {
						dropTarget = tab.ID
						dropBefore = false
						break
					}
				}
			}

			if dropTarget != "" && tb.OnDragDrop != nil {
				tb.OnDragDrop(tb.dragging, dropTarget, dropBefore)
			}
			tb.dragging = ""
			return true
		}
		return false

	case event.MouseDoubleClick:
		// Double-click on tab could trigger rename or other actions
		for _, tab := range tb.Tabs {
			if rect, ok := tb.tabRects[tab.ID]; ok {
				if local.X >= rect.Min.X && local.X <= rect.Max.X &&
					local.Y >= rect.Min.Y && local.Y <= rect.Max.Y {
					// Could trigger rename here
					return true
				}
			}
		}
		return false
	}
	return false
}

// ScrollBy scrolls the tab bar horizontally.
func (tb *TabBar) ScrollBy(delta float32) {
	tb.ScrollOffset += delta
	if tb.ScrollOffset < 0 {
		tb.ScrollOffset = 0
	}
	tb.SetNeedsRedraw(true)
}

// Colors for the tab bar (VS Code dark theme inspired)
var (
	tabBarBackgroundColor      = widget.RGBA8(37, 37, 38, 255)
	tabInactiveBackgroundColor = widget.RGBA8(45, 45, 45, 255)
	tabActiveBackgroundColor   = widget.RGBA8(50, 50, 50, 255)
	tabSelectedBackgroundColor = widget.RGBA8(55, 55, 55, 255)
	tabHoverBackgroundColor    = widget.RGBA8(50, 50, 50, 255)

	tabInactiveForegroundColor = widget.RGBA8(190, 190, 190, 255)
	tabActiveForegroundColor   = widget.RGBA8(255, 255, 255, 255)
	tabSubtitleColor         = widget.RGBA8(140, 140, 140, 255)

	tabBorderColor         = widget.RGBA8(50, 50, 50, 255)
	tabActiveBorderColor   = widget.RGBA8(0, 122, 204, 255)
	tabSeparatorColor      = widget.RGBA8(60, 60, 60, 255)

	tabCloseButtonColor      = widget.RGBA8(190, 190, 190, 200)
	tabCloseButtonHoverColor = widget.RGBA8(255, 255, 255, 255)

	dirtyIndicatorColor = widget.RGBA8(255, 187, 0, 255)
)
