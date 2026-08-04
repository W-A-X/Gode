package editor

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// TabInfo holds the data for a single editor tab.
type TabInfo struct {
	// Label is the display name (typically filename).
	Label string
	// Description is the full path or description shown in tooltip.
	Description string
	// IconName is an optional icon identifier (e.g., "file-type-ts").
	IconName string
	// IsDirty shows a dot indicator when true (unsaved changes).
	IsDirty bool
	// IsActive is the currently focused tab.
	IsActive bool
	// IsPinned means the tab is pinned/sticky.
	IsPinned bool
	// IsHovered for hover highlight state.
	IsHovered bool
}

// TabBar is a gogpu/ui widget that renders a JetBrains-style horizontal tab
// bar: the active tab merges with the editor background and carries a 2px
// accent underline at the bottom; inactive tabs sit on the darker tab-bar
// surface with no separators; the close button appears on hover only.
// It handles tab layout, rendering, and mouse interaction (click, close, scroll).
type TabBar struct {
	widget.WidgetBase

	// Tabs is the list of tabs to render.
	Tabs []TabInfo

	// ActiveTabIdx is the index of the active/focused tab (-1 if none).
	ActiveTabIdx int

	// HoveredTabIdx is the index of the hovered tab (-1 if none).
	HoveredTabIdx int

	// OnTabSelected is called when a tab is clicked.
	OnTabSelected func(idx int)

	// OnTabClose is called when the close button (x) is clicked.
	OnTabClose func(idx int)

	// OnTabContextMenu is called when a tab is right-clicked.
	OnTabContextMenu func(idx int, x, y float32)

	// Options carries visual configuration.
	TabOptions TabOptions

	// layout cache
	tabRects []geometry.Rect
	scrollX  float32
	dragTab  int // index of tab being dragged, -1 if none
}

// TabOptions configures the appearance of TabBar.
type TabOptions struct {
	Height         float32
	TabHeight      float32
	TabMinWidth    float32
	TabMaxWidth    float32
	TabPadding     float32
	FontSize       float32
	IconSize       float32
	CloseButtonPad float32
	ScrollWidth    float32
}

// DefaultTabOptions returns sensible JetBrains-like defaults.
func DefaultTabOptions() TabOptions {
	return TabOptions{
		Height:         35,
		TabHeight:      35,
		TabMinWidth:    60,
		TabMaxWidth:    200,
		TabPadding:     10,
		FontSize:       13,
		IconSize:       16,
		CloseButtonPad: 16,
		ScrollWidth:    20,
	}
}

// NewTabBar creates a new TabBar widget.
func NewTabBar() *TabBar {
	tb := &TabBar{
		ActiveTabIdx:  -1,
		HoveredTabIdx: -1,
		dragTab:       -1,
		TabOptions:    DefaultTabOptions(),
	}
	return tb
}

// Bounds implements widget.Widget.
func (tb *TabBar) Bounds() geometry.Rect {
	return tb.WidgetBase.Bounds()
}

// SetBounds implements widget.Widget.
func (tb *TabBar) SetBounds(r geometry.Rect) {
	tb.WidgetBase.SetBounds(r)
	tb.recalcLayout()
}

// Update updates the tab list and active index, triggering redraw if changed.
func (tb *TabBar) Update(tabs []TabInfo, activeIdx int) {
	tb.Tabs = tabs
	tb.ActiveTabIdx = activeIdx
	tb.recalcLayout()
	tb.SetNeedsRedraw(true)
}

// recalcLayout computes per-tab rectangles based on current scroll position.
func (tb *TabBar) recalcLayout() {
	opts := tb.TabOptions
	n := len(tb.Tabs)
	if n == 0 {
		tb.tabRects = nil
		return
	}

	b := tb.Bounds()
	availableW := b.Width() - opts.ScrollWidth

	tb.tabRects = make([]geometry.Rect, n)
	x := tb.scrollX * -1 // negative because we scroll left
	for i := range tb.Tabs {
		w := tb.tabWidth(i)
		if w > availableW-x {
			w = max(availableW-x, 0)
		}
		tb.tabRects[i] = geometry.NewRect(x, 0, w, opts.TabHeight)
		x += w
		if x >= availableW {
			break
		}
	}
}

// tabWidth returns the width for a given tab index based on label length.
func (tb *TabBar) tabWidth(i int) float32 {
	if i < 0 || i >= len(tb.Tabs) {
		return tb.TabOptions.TabMinWidth
	}
	opts := tb.TabOptions
	labelLen := float32(len(tb.Tabs[i].Label))
	// Estimate width based on character count + padding + icon + close button
	charW := opts.FontSize * 0.6 // approximate char width
	w := opts.TabPadding*2 + charW*labelLen + opts.IconSize + opts.CloseButtonPad
	return min(max(w, opts.TabMinWidth), opts.TabMaxWidth)
}

// Draw implements widget.Widget.
func (tb *TabBar) Draw(ctx widget.Canvas) {
	opts := tb.TabOptions
	b := tb.Bounds()

	// Background
	ctx.DrawRect(b, TabBackgroundColor)

	if len(tb.Tabs) == 0 || len(tb.tabRects) == 0 {
		return
	}

	for i, tr := range tb.tabRects {
		if tr.Width() <= 0 || tr.Min.X+tr.Width() > b.Width()-opts.ScrollWidth {
			continue
		}
		tb.drawTab(ctx, i, geometry.NewRect(tr.Min.X+b.Min.X, tr.Min.Y+b.Min.Y, tr.Width(), tr.Height()))
	}

	// Draw scroll buttons if needed
	totalW := tb.totalTabsWidth()
	if totalW > b.Width() {
		tb.drawScrollButtons(ctx, b)
	}
}

// drawTab renders a single tab at the given rect.
func (tb *TabBar) drawTab(ctx widget.Canvas, idx int, r geometry.Rect) {
	tab := tb.Tabs[idx]
	opts := tb.TabOptions

	isActive := idx == tb.ActiveTabIdx
	isHovered := idx == tb.HoveredTabIdx && tb.dragTab < 0

	// Tab background: active merges with the editor, hover overlays the surface.
	var bg widget.Color
	switch {
	case isActive:
		bg = TabActiveBackground
	case isHovered:
		bg = TabHoverBackground
	default:
		bg = TabInactiveBackground
	}
	ctx.DrawRect(r, bg)

	// Icon area (left side of tab)
	iconSize := opts.IconSize
	iconX := r.Min.X + opts.TabPadding
	iconY := r.Min.Y + (r.Height()-iconSize)/2

	iconColor := TabInactiveForeground
	if isActive {
		iconColor = TabActiveForeground
	}
	tb.drawDocumentIcon(ctx, iconX, iconY, iconSize, iconColor)

	// Label text (JetBrains: close button only appears on hover)
	textX := iconX + iconSize + 4
	closeW := float32(0)
	if isHovered && !tab.IsPinned {
		closeW = opts.IconSize*0.75 + opts.CloseButtonPad
	}
	maxTextW := r.Width() - (textX - r.Min.X) - closeW - opts.TabPadding

	labelColor := TabInactiveForeground
	if isActive {
		labelColor = TabActiveForeground
	}

	// Truncate label if needed
	label := tab.Label
	if maxTextW > 0 {
		label = truncateString(label, int(maxTextW/opts.FontSize*2))
	}

	// Dirty indicator (dot before label)
	if tab.IsDirty {
		dotX := textX + 3
		dotY := r.Min.Y + r.Height()/2
		dotR := opts.FontSize / 7
		ctx.DrawCircle(geometry.Pt(dotX, dotY), dotR, DirtyIndicatorColor)
		textX += dotR*2 + 5
	}

	// Draw label (active tab is bold, matching the DevTools tabview painter)
	if len(label) > 0 {
		ctx.DrawText(label, geometry.NewRect(textX, r.Min.Y, maxTextW, r.Height()),
			opts.FontSize, labelColor, isActive, widget.TextAlignLeft)
	}

	// Active tab indicator: 2px bottom accent line (JetBrains/DevTools style).
	if isActive {
		ctx.DrawRect(geometry.NewRect(r.Min.X, r.Max.Y-2, r.Width(), 2), TabActiveBorderColor)
	}

	// Close button (hover only)
	if isHovered && !tab.IsPinned {
		closeSize := opts.IconSize * 0.75
		closeX := r.Min.X + r.Width() - opts.CloseButtonPad - opts.TabPadding - closeSize
		closeY := r.Min.Y + (r.Height()-closeSize)/2
		tb.drawCloseButton(ctx, closeX, closeY, closeSize, true)
	}
}

// drawDocumentIcon draws a simple document icon placeholder.
func (tb *TabBar) drawDocumentIcon(ctx widget.Canvas, x, y, size float32, c widget.Color) {
	// Simple folded-corner document shape
	padding := size * 0.15
	body := geometry.NewRect(x+padding, y, size-padding*1.5, size-padding)
	ctx.StrokeRect(body, c, 1.0)
	// Fold corner
	fold := geometry.NewRect(x+size-padding*1.5, y+padding, padding*1.5, padding*1.5)
	ctx.DrawRect(fold, c.WithAlpha(0.5))
}

// drawCloseButton draws an X close button.
func (tb *TabBar) drawCloseButton(ctx widget.Canvas, x, y, size float32, hovered bool) {
	c := TabCloseButtonColor
	if hovered {
		c = TabCloseButtonHoverColor
	}
	// X drawn as two lines
	lw := float32(1.5)
	ctx.DrawLine(geometry.Pt(x, y), geometry.Pt(x+size, y+size), c, lw)
	ctx.DrawLine(geometry.Pt(x+size, y), geometry.Pt(x, y+size), c, lw)
}

// drawScrollButtons draws left/right scroll arrows when tabs overflow.
func (tb *TabBar) drawScrollButtons(ctx widget.Canvas, b geometry.Rect) {
	opts := tb.TabOptions
	btnH := opts.Height
	btnW := opts.ScrollWidth

	// Left scroll button
	leftBtn := geometry.NewRect(b.Min.X+b.Width()-btnW*2, b.Min.Y, btnW, btnH)
	ctx.DrawRect(leftBtn, TabBackgroundColor)
	ctx.StrokeRect(leftBtn, TabSeparatorColor, 1.0)
	ctx.DrawText("◀", leftBtn, opts.FontSize*0.8, TabInactiveForeground, false, widget.TextAlignCenter)

	// Right scroll button
	rightBtn := geometry.NewRect(b.Min.X+b.Width()-btnW, b.Min.Y, btnW, btnH)
	ctx.DrawRect(rightBtn, TabBackgroundColor)
	ctx.StrokeRect(rightBtn, TabSeparatorColor, 1.0)
	ctx.DrawText("▶", rightBtn, opts.FontSize*0.8, TabInactiveForeground, false, widget.TextAlignCenter)
}

// HandleEvent implements mouse event handling for tabs.
func (tb *TabBar) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}

	pos := me.Position
	switch me.MouseType {
	case event.MousePress:
		return tb.handlePress(pos, me.Button)
	case event.MouseRelease:
		return tb.handleRelease(pos, me.Button)
	case event.MouseMove:
		return tb.handleMove(pos)
	case event.MouseDoubleClick:
		return tb.handleDoubleClick(pos)
	}
	return false
}

// handlePress handles mouse press events.
func (tb *TabBar) handlePress(pos geometry.Point, btn event.Button) bool {
	idx := tb.tabAt(pos)
	if idx >= 0 {
		if btn == event.ButtonRight && tb.OnTabContextMenu != nil {
			tb.OnTabContextMenu(idx, pos.X, pos.Y)
			return true
		}
		// Check if click is on close button
		if tb.isOnCloseButton(pos, idx) && tb.OnTabClose != nil {
			tb.OnTabClose(idx)
			return true
		}
		// Select tab
		if tb.OnTabSelected != nil {
			tb.OnTabSelected(idx)
			return true
		}
	}
	// Check scroll buttons
	return tb.handleScrollButtonClick(pos)
}

// handleRelease handles mouse release events.
func (tb *TabBar) handleRelease(pos geometry.Point, btn event.Button) bool {
	if tb.dragTab >= 0 {
		tb.dragTab = -1
		return true
	}
	return false
}

// handleMove handles mouse move events for hover effect.
func (tb *TabBar) handleMove(pos geometry.Point) bool {
	newHovered := tb.tabAt(pos)
	if newHovered != tb.HoveredTabIdx {
		tb.HoveredTabIdx = newHovered
		tb.SetNeedsRedraw(true)
	}
	return tb.HoveredTabIdx >= 0
}

// handleDoubleClick handles double-click events.
func (tb *TabBar) handleDoubleClick(pos geometry.Point) bool {
	// Could be used for "pin tab" or other actions
	return false
}

// tabAt returns the tab index at the given local position, or -1 if none.
func (tb *TabBar) tabAt(pos geometry.Point) int {
	for i, tr := range tb.tabRects {
		if pos.X >= tr.Min.X && pos.X < tr.Min.X+tr.Width() &&
			pos.Y >= tr.Min.Y && pos.Y < tr.Min.Y+tr.Height() {
			return i
		}
	}
	return -1
}

// isOnCloseButton checks if position is on the close button of the given tab.
func (tb *TabBar) isOnCloseButton(pos geometry.Point, tabIdx int) bool {
	if tabIdx < 0 || tabIdx >= len(tb.tabRects) {
		return false
	}
	tr := tb.tabRects[tabIdx]
	opts := tb.TabOptions
	closeSize := opts.IconSize * 0.75
	closeX := tr.Min.X + tr.Width() - opts.CloseButtonPad - opts.TabPadding - closeSize
	closeY := float32(0)
	closeW := closeSize
	closeH := closeSize

	return pos.X >= closeX && pos.X < closeX+closeW &&
		pos.Y >= closeY && pos.Y < closeY+closeH
}

// handleScrollButtonClick handles clicks on scroll buttons.
func (tb *TabBar) handleScrollButtonClick(pos geometry.Point) bool {
	opts := tb.TabOptions
	b := tb.Bounds()
	btnW := opts.ScrollWidth

	// Left button
	leftBtn := geometry.NewRect(b.Width()-btnW*2, 0, btnW, opts.Height)
	if pos.X >= leftBtn.Min.X && pos.X < leftBtn.Min.X+leftBtn.Width() &&
		pos.Y >= leftBtn.Min.Y && pos.Y < leftBtn.Min.Y+leftBtn.Height() {
		tb.scrollLeft()
		return true
	}

	// Right button
	rightBtn := geometry.NewRect(b.Width()-btnW, 0, btnW, opts.Height)
	if pos.X >= rightBtn.Min.X && pos.X < rightBtn.Min.X+rightBtn.Width() &&
		pos.Y >= rightBtn.Min.Y && pos.Y < rightBtn.Min.Y+rightBtn.Height() {
		tb.scrollRight()
		return true
	}

	return false
}

// scrollLeft scrolls the tab bar to show tabs on the left.
func (tb *TabBar) scrollLeft() {
	tb.scrollX = max(tb.scrollX-100, 0)
	tb.recalcLayout()
	tb.SetNeedsRedraw(true)
}

// scrollRight scrolls the tab bar to show tabs on the right.
func (tb *TabBar) scrollRight() {
	maxScroll := tb.maxScrollX()
	tb.scrollX = min(tb.scrollX+100, maxScroll)
	tb.recalcLayout()
	tb.SetNeedsRedraw(true)
}

// totalTabsWidth returns the total width of all tabs.
func (tb *TabBar) totalTabsWidth() float32 {
	w := float32(0)
	for i := range tb.Tabs {
		w += tb.tabWidth(i)
	}
	return w
}

// maxScrollX returns the maximum scroll offset.
func (tb *TabBar) maxScrollX() float32 {
	b := tb.Bounds()
	total := tb.totalTabsWidth()
	available := b.Width() - tb.TabOptions.ScrollWidth
	return max(0, total-available)
}

// EnsureVisible scrolls the tab bar to ensure the given tab index is visible.
func (tb *TabBar) EnsureVisible(idx int) {
	if idx < 0 || idx >= len(tb.tabRects) {
		return
	}
	tr := tb.tabRects[idx]
	b := tb.Bounds()
	available := b.Width() - tb.TabOptions.ScrollWidth

	// Scroll left if tab starts before visible area
	if tr.Min.X < tb.scrollX*-1 {
		tb.scrollX = -tr.Min.X
	}
	// Scroll right if tab ends after visible area
	if tr.Min.X+tr.Width() > available+tb.scrollX*-1 {
		tb.scrollX = -(tr.Min.X + tr.Width() - available)
	}
	tb.scrollX = max(0, min(tb.scrollX, tb.maxScrollX()))
	tb.recalcLayout()
	tb.SetNeedsRedraw(true)
}

// truncateString shortens a string to fit within maxRunes, adding "...".
func truncateString(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
}

// --- JetBrains DevTools Dark Tab Colors (gogpu/ui examples/ide palette) ---

var (
	// TabBackgroundColor is the tab bar background (Surface Gray2).
	TabBackgroundColor = widget.RGBA8(0x2B, 0x2D, 0x30, 0xFF)
	// TabActiveBackground merges with the editor background (Gray1).
	TabActiveBackground = widget.RGBA8(0x1E, 0x1F, 0x22, 0xFF)
	// TabInactiveBackground matches the tab bar surface.
	TabInactiveBackground = widget.RGBA8(0x2B, 0x2D, 0x30, 0xFF)
	// TabHoverBackground is the hover state (SurfaceElevated Gray3).
	TabHoverBackground = widget.RGBA8(0x39, 0x3B, 0x40, 0xFF)
	// TabActiveBorderColor is the 2px bottom indicator of the active tab (Blue6).
	TabActiveBorderColor = widget.RGBA8(0x35, 0x74, 0xF0, 0xFF)
	// TabSeparatorColor is used for the scroll-button outline (Gray3).
	TabSeparatorColor = widget.RGBA8(0x39, 0x3B, 0x40, 0xFF)
	// TabActiveForeground is the text color for active tab (Gray12).
	TabActiveForeground = widget.RGBA8(0xDF, 0xE1, 0xE5, 0xFF)
	// TabInactiveForeground is the text color for inactive tabs (Gray9).
	TabInactiveForeground = widget.RGBA8(0x9D, 0xA0, 0xA8, 0xFF)
	// TabCloseButtonColor is the normal close button color (Gray9).
	TabCloseButtonColor = widget.RGBA8(0x9D, 0xA0, 0xA8, 0xFF)
	// TabCloseButtonHoverColor is the close button color on hover (Gray12).
	TabCloseButtonHoverColor = widget.RGBA8(0xDF, 0xE1, 0xE5, 0xFF)
	// DirtyIndicatorColor is the circle indicating unsaved changes (Gray9).
	DirtyIndicatorColor = widget.RGBA8(0x9D, 0xA0, 0xA8, 0xFF)
)

func min(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
