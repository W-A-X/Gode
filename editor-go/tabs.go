package editor

import (
	"fmt"
	"math"
	"strings"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/render"
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

// TabBar is a gogpu/ui widget that renders a VS Code-style horizontal tab bar.
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

// DefaultTabOptions returns sensible VS Code-like defaults.
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
		ActiveTabIdx: -1,
		HoveredTabIdx: -1,
		dragTab:      -1,
		TabOptions:   DefaultTabOptions(),
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
	tb.MarkForRedraw()
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
	availableW := b.W - opts.ScrollWidth

	tb.tabRects = make([]geometry.Rect, n)
	x := tb.scrollX * -1 // negative because we scroll left
	for i := range tb.Tabs {
		w := tb.tabWidth(i)
		if w > availableW-x {
			w = max(availableW-x, 0)
		}
		tb.tabRects[i] = geometry.Rect{
			X: x,
			Y: 0,
			W: w,
			H: opts.TabHeight,
		}
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
func (tb *TabBar) Draw(ctx render.Context) {
	opts := tb.TabOptions
	b := tb.Bounds()

	// Background
	ctx.Fill(geometry.Rt(b.X, b.Y, b.W, b.H), TabBackgroundColor)

	if len(tb.Tabs) == 0 || len(tb.tabRects) == 0 {
		return
	}

	for i, tr := range tb.tabRects {
		if tr.W <= 0 || tr.X+tr.W > b.W-opts.ScrollWidth {
			continue
		}
		tb.drawTab(ctx, i, geometry.Rt(tr.X+b.X, tr.Y+b.Y, tr.W, tr.H))
	}

	// Draw scroll buttons if needed
	totalW := tb.totalTabsWidth()
	if totalW > b.W {
		tb.drawScrollButtons(ctx, b)
	}
}

// drawTab renders a single tab at the given rect.
func (tb *TabBar) drawTab(ctx render.Context, idx int, r geometry.Rect) {
	tab := tb.Tabs[idx]
	opts := tb.TabOptions

	isActive := idx == tb.ActiveTabIdx
	isHovered := idx == tb.HoveredTabIdx && tb.dragTab < 0

	// Tab background
	var bg widget.Color
	switch {
	case isActive:
		bg = TabActiveBackground
	case isHovered:
		bg = TabHoverBackground
	default:
		bg = TabInactiveBackground
	}
	ctx.Fill(r, bg)

	// Active tab bottom border
	if isActive {
		borderRect := geometry.Rt(r.X, r.Y+r.H-2, r.W, 2)
		ctx.Fill(borderRect, TabActiveBorderColor)
	} else {
		// Separator line between inactive tabs
		sepRect := geometry.Rt(r.X+r.W-1, r.Y+8, 1, r.H-16)
		ctx.Fill(sepRect, TabSeparatorColor)
	}

	// Icon area (left side of tab)
	iconX := r.X + opts.TabPadding
	iconY := r.Y + (r.H - opts.IconSize) / 2

	// For now, draw a simple document icon placeholder
	// In production, this would use actual file-type icons
	iconColor := TabInactiveForeground
	if isActive {
		iconColor = TabActiveForeground
	}
	tb.drawDocumentIcon(ctx, iconX, iconY, opts.IconSize, iconColor)

	// Label text
	textX := iconX + opts.IconSize + 4
	textY := r.Y + (r.H - opts.FontSize) / 2
	maxTextW := r.W - (textX - r.X) - opts.CloseButtonPad - opts.TabPadding

	labelColor := TabInactiveForeground
	if isActive {
		labelColor = TabActiveForeground
	}

	// Truncate label if needed
	label := tab.Label
	if maxTextW > 0 {
		label = truncateString(label, int(maxTextW/opts.FontSize*2))
	}

	// Draw dirty indicator (dot before label)
	if tab.IsDirty {
		dotX := textX
		dotY := textY + opts.FontSize/3
		dotR := opts.FontSize / 6
		ctx.Fill(geometry.Circle(dotX, dotY, dotR), DirtyIndicatorColor)
		textX += dotR*2 + 4
	}

	// Draw label
	if len(label) > 0 {
		ctx.Text(label, geometry.Pt(textX, textY), labelColor, opts.FontSize)
	}

	// Close button (shown on hover or active tab)
	if isActive || isHovered {
		closeX := r.X + r.W - opts.CloseButtonPad - opts.TabPadding
		closeY := r.Y + (r.H - opts.IconSize) / 2
		closeSize := opts.IconSize * 0.75
		tb.drawCloseButton(ctx, closeX, closeY, closeSize, idx == tb.HoveredTabIdx)
	}

	// Pinned indicator
	if tab.IsPinned {
		pinX := r.X + r.W - opts.CloseButtonPad - opts.TabPadding - 12
		pinY := r.Y + 8
		ctx.Text("📌", geometry.Pt(pinX, pinY), widget.Color{R: 0.5, G: 0.5, B: 0.5, A: 1}, 10)
	}
}

// drawDocumentIcon draws a simple document icon placeholder.
func (tb *TabBar) drawDocumentIcon(ctx render.Context, x, y, size float32, c widget.Color) {
	// Simple folded-corner document shape
	padding := size * 0.15
	body := geometry.Rt(x+padding, y, size-padding*1.5, size-padding)
	ctx.Stroke(body, c, 1.0)
	// Fold corner
	fold := geometry.Rt(x+size-padding*1.5, y+padding, padding*1.5, padding*1.5)
	ctx.Fill(fold, c.MulAlpha(0.5))
}

// drawCloseButton draws an X close button.
func (tb *TabBar) drawCloseButton(ctx render.Context, x, y, size float32, hovered bool) {
	c := TabCloseButtonColor
	if hovered {
		c = TabCloseButtonHoverColor
	}
	half := size / 2
	// X drawn as two lines
	lw := 1.5
	ctx.Line(geometry.Pt(x, y), geometry.Pt(x+size, y+size), c, lw)
	ctx.Line(geometry.Pt(x+size, y), geometry.Pt(x, y+size), c, lw)
}

// drawScrollButtons draws left/right scroll arrows when tabs overflow.
func (tb *TabBar) drawScrollButtons(ctx render.Context, b geometry.Rect) {
	opts := tb.TabOptions
	btnH := opts.Height
	btnW := opts.ScrollWidth

	// Left scroll button
	leftBtn := geometry.Rt(b.X+b.W-btnW*2, b.Y, btnW, btnH)
	ctx.Fill(leftBtn, TabBackgroundColor)
	ctx.Stroke(leftBtn, TabSeparatorColor, 1.0)
	// Left arrow
	ax := leftBtn.X + btnW/3
	ay := leftBtn.Y + btnH/2
	ctx.Text("◀", geometry.Pt(ax, ay-opts.FontSize/2), TabInactiveForeground, opts.FontSize*0.8)

	// Right scroll button
	rightBtn := geometry.Rt(b.X+b.W-btnW, b.Y, btnW, btnH)
	ctx.Fill(rightBtn, TabBackgroundColor)
	ctx.Stroke(rightBtn, TabSeparatorColor, 1.0)
	// Right arrow
	ax = rightBtn.X + btnW/3
	ay = rightBtn.Y + btnH/2
	ctx.Text("▶", geometry.Pt(ax, ay-opts.FontSize/2), TabInactiveForeground, opts.FontSize*0.8)
}

// HandleEvent implements mouse event handling for tabs.
func (tb *TabBar) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}

	pos := me.Position()
	b := tb.Bounds()
	localPos := geometry.Pt(pos.X-b.X, pos.Y-b.Y)

	switch me.Type() {
	case event.MousePress:
		return tb.handlePress(localPos, me.Button())
	case event.MouseRelease:
		return tb.handleRelease(localPos, me.Button())
	case event.MouseMove:
		return tb.handleMove(localPos)
	case event.MouseDoubleClick:
		return tb.handleDoubleClick(localPos)
	}
	return false
}

// handlePress handles mouse press events.
func (tb *TabBar) handlePress(pos geometry.Pos, btn event.Button) bool {
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
func (tb *TabBar) handleRelease(pos geometry.Pos, btn event.Button) bool {
	if tb.dragTab >= 0 {
		tb.dragTab = -1
		return true
	}
	return false
}

// handleMove handles mouse move events for hover effect.
func (tb *TabBar) handleMove(pos geometry.Pos) bool {
	newHovered := tb.tabAt(pos)
	if newHovered != tb.HoveredTabIdx {
		tb.HoveredTabIdx = newHovered
		tb.MarkForRedraw()
	}
	return tb.HoveredTabIdx >= 0
}

// handleDoubleClick handles double-click events.
func (tb *TabBar) handleDoubleClick(pos geometry.Pos) bool {
	// Could be used for "pin tab" or other actions
	return false
}

// tabAt returns the tab index at the given local position, or -1 if none.
func (tb *TabBar) tabAt(pos geometry.Pos) int {
	for i, tr := range tb.tabRects {
		if pos.X >= tr.X && pos.X < tr.X+tr.W &&
			pos.Y >= tr.Y && pos.Y < tr.Y+tr.H {
			return i
		}
	}
	return -1
}

// isOnCloseButton checks if position is on the close button of the given tab.
func (tb *TabBar) isOnCloseButton(pos geometry.Pos, tabIdx int) bool {
	if tabIdx < 0 || tabIdx >= len(tb.tabRects) {
		return false
	}
	tr := tb.tabRects[tabIdx]
	opts := tb.TabOptions
	closeX := tr.X + tr.W - opts.CloseButtonPad - opts.TabPadding
	closeY := 0
	closeW := opts.IconSize * 0.75
	closeH := opts.IconSize * 0.75

	return pos.X >= closeX && pos.X < closeX+closeW &&
		pos.Y >= closeY && pos.Y < closeY+closeH
}

// handleScrollButtonClick handles clicks on scroll buttons.
func (tb *TabBar) handleScrollButtonClick(pos geometry.Pos) bool {
	opts := tb.TabOptions
	b := tb.Bounds()
	btnW := opts.ScrollWidth

	// Left button
	leftBtn := geometry.Rt(b.W-btnW*2, 0, btnW, opts.Height)
	if pos.X >= leftBtn.X && pos.X < leftBtn.X+leftBtn.W &&
		pos.Y >= leftBtn.Y && pos.Y < leftBtn.Y+leftBtn.H {
		tb.scrollLeft()
		return true
	}

	// Right button
	rightBtn := geometry.Rt(b.W-btnW, 0, btnW, opts.Height)
	if pos.X >= rightBtn.X && pos.X < rightBtn.X+rightBtn.W &&
		pos.Y >= rightBtn.Y && pos.Y < rightBtn.Y+rightBtn.H {
		tb.scrollRight()
		return true
	}

	return false
}

// scrollLeft scrolls the tab bar to show tabs on the left.
func (tb *TabBar) scrollLeft() {
	tb.scrollX = max(tb.scrollX-100, 0)
	tb.recalcLayout()
	tb.MarkForRedraw()
}

// scrollRight scrolls the tab bar to show tabs on the right.
func (tb *TabBar) scrollRight() {
	maxScroll := tb.maxScrollX()
	tb.scrollX = min(tb.scrollX+100, maxScroll)
	tb.recalcLayout()
	tb.MarkForRedraw()
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
	available := b.W - tb.TabOptions.ScrollWidth
	return max(0, total-available)
}

// EnsureVisible scrolls the tab bar to ensure the given tab index is visible.
func (tb *TabBar) EnsureVisible(idx int) {
	if idx < 0 || idx >= len(tb.tabRects) {
		return
	}
	tr := tb.tabRects[idx]
	b := tb.Bounds()
	available := b.W - tb.TabOptions.ScrollWidth

	// Scroll left if tab starts before visible area
	if tr.X < tb.scrollX*-1 {
		tb.scrollX = -tr.X
	}
	// Scroll right if tab ends after visible area
	if tr.X+tr.W > available+tb.scrollX*-1 {
		tb.scrollX = -(tr.X + tr.W - available)
	}
	tb.scrollX = max(0, min(tb.scrollX, tb.maxScrollX()))
	tb.recalcLayout()
	tb.MarkForRedraw()
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

// --- VS Code Dark+ Theme Colors for Tabs ---

var (
	// TabBackgroundColor is the tab bar background.
	TabBackgroundColor = widget.Color{R: 0.25, G: 0.25, B: 0.28, A: 1.0}
	// TabActiveBackground is the active tab background color.
	TabActiveBackground = widget.Color{R: 0.30, G: 0.30, B: 0.33, A: 1.0}
	// TabInactiveBackground is the inactive tab background color.
	TabInactiveBackground = widget.Color{R: 0.25, G: 0.25, B: 0.28, A: 1.0}
	// TabHoverBackground is the hover state background.
	TabHoverBackground = widget.Color{R: 0.28, G: 0.28, B: 0.31, A: 1.0}
	// TabActiveBorderColor is the bottom border of the active tab.
	TabActiveBorderColor = widget.Color{R: 0.15, G: 0.65, B: 0.73, A: 1.0} // VS Code blue accent
	// TabSeparatorColor is the line between tabs.
	TabSeparatorColor = widget.Color{R: 0.35, G: 0.35, B: 0.38, A: 1.0}
	// TabActiveForeground is the text color for active tab.
	TabActiveForeground = widget.Color{R: 0.97, G: 0.97, B: 0.97, A: 1.0}
	// TabInactiveForeground is the text color for inactive tabs.
	TabInactiveForeground = widget.Color{R: 0.70, G: 0.70, B: 0.72, A: 1.0}
	// TabCloseButtonColor is the normal close button color.
	TabCloseButtonColor = widget.Color{R: 0.55, G: 0.55, B: 0.57, A: 1.0}
	// TabCloseButtonHoverColor is the close button color on hover.
	TabCloseButtonHoverColor = widget.Color{R: 0.97, G: 0.97, B: 0.97, A: 1.0}
	// DirtyIndicatorColor is the circle indicating unsaved changes.
	DirtyIndicatorColor = widget.Color{R: 0.15, G: 0.65, B: 0.73, A: 1.0} // Accent blue
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

// String returns a debug representation of the TabBar.
func (tb *TabBar) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "TabBar{tabs=%d, active=%d", len(tb.Tabs), tb.ActiveTabIdx)
	if tb.HoveredTabIdx >= 0 {
		fmt.Fprintf(&sb, ", hovered=%d", tb.HoveredTabIdx)
	}
	sb.WriteByte('}')
	return sb.String()
}

// WheelEvent handles horizontal scrolling via wheel events.
func (tb *TabBar) WheelEvent(deltaX, deltaY float32) bool {
	if math.Abs(float64(deltaX)) > 0.1 || math.Abs(float64(deltaY)) > 0.1 {
		// Horizontal scroll or shift+vertical scroll
		scrollAmt := deltaX + deltaY
		tb.scrollX = max(0, min(tb.scrollX-scrollAmt, tb.maxScrollX()))
		tb.recalcLayout()
		tb.MarkForRedraw()
		return true
	}
	return false
}
