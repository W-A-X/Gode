// Command gode launches the complete Gode Editor with:
// - Tauri-style window management (custom title bar)
// - Go-based editor rendering (gogpu/ui)
// - Bottom activity bar (replacing traditional left sidebar)
// - Status bar
package main

import (
	"log"
	"os"
	"path/filepath"

	_ "github.com/gogpu/gg/gpu"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/desktop"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"gode/editor"
)

// TitleBar is a custom title bar widget
type TitleBar struct {
	widget.WidgetBase
	title    string
	onClose  func()
	onMinimize func()
	onMaximize func()
}

func NewTitleBar(title string) *TitleBar {
	return &TitleBar{
		title: title,
	}
}

func (t *TitleBar) SetTitle(title string) {
	t.title = title
	t.SetNeedsRedraw(true)
}

func (t *TitleBar) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	size := c.Biggest()
	return geometry.Size{Width: size.Width, Height: 30}
}

func (t *TitleBar) Children() []widget.Widget {
	return nil
}

func (t *TitleBar) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := t.Bounds()
	size := bounds.Size()
	
	// Background
	bgColor := widget.RGBA8(0x3C, 0x3C, 0x3C, 0xFF)
	canvas.DrawRect(bounds, bgColor)
	
	// Title text
	textColor := widget.RGBA8(0xCC, 0xCC, 0xCC, 0xFF)
	canvas.DrawText(
		t.title,
		geometry.Rect{
			Min: geometry.Pt(10, 8),
			Max: geometry.Pt(size.Width-60, 22),
		},
		12,
		textColor,
		false,
		widget.TextAlignLeft,
	)
	
	// Window controls (visual only - actual handling by window manager)
	// Close button (red)
	closeBtn := geometry.Rect{
		Min: geometry.Pt(size.Width-36, 9),
		Max: geometry.Pt(size.Width-24, 21),
	}
	canvas.DrawRect(closeBtn, widget.RGBA8(0xFF, 0x5F, 0x56, 0xFF))
	
	// Minimize button (yellow)
	minBtn := geometry.Rect{
		Min: geometry.Pt(size.Width-52, 9),
		Max: geometry.Pt(size.Width-40, 21),
	}
	canvas.DrawRect(minBtn, widget.RGBA8(0xFF, 0xBD, 0x2E, 0xFF))
	
	// Maximize button (green)
	maxBtn := geometry.Rect{
		Min: geometry.Pt(size.Width-68, 9),
		Max: geometry.Pt(size.Width-56, 21),
	}
	canvas.DrawRect(maxBtn, widget.RGBA8(0x27, 0xC9, 0x3F, 0xFF))
}

func (t *TitleBar) Event(ctx widget.Context, e event.Event) bool {
	// Title bar is draggable region - events pass through to window manager
	return false
}

// StatusBar is the status bar widget
type StatusBar struct {
	widget.WidgetBase
	lineCol      string
	encoding     string
	language     string
	gitBranch    string
	errors       int
	warnings     int
}

func NewStatusBar() *StatusBar {
	return &StatusBar{
		lineCol:   "Ln 1, Col 1",
		encoding:  "UTF-8",
		language:  "Plain Text",
		gitBranch: "main",
		errors:    0,
		warnings:  0,
	}
}

func (s *StatusBar) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	size := c.Biggest()
	return geometry.Size{Width: size.Width, Height: 22}
}

func (s *StatusBar) Children() []widget.Widget {
	return nil
}

func (s *StatusBar) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := s.Bounds()
	size := bounds.Size()
	
	// Background (VS Code blue)
	bgColor := widget.RGBA8(0x00, 0x7A, 0xCC, 0xFF)
	canvas.DrawRect(bounds, bgColor)
	
	textColor := widget.RGBA8(0xFF, 0xFF, 0xFF, 0xFF)
	
	// Left side: Git branch and errors/warnings
	leftX := float32(10)
	canvas.DrawText(
		"🌿 "+s.gitBranch,
		geometry.Rect{
			Min: geometry.Pt(leftX, 4),
			Max: geometry.Pt(leftX+80, 18),
		},
		11,
		textColor,
		false,
		widget.TextAlignLeft,
	)
	
	leftX += 90
	canvas.DrawText(
		"⊗ "+itoa(s.errors)+" ⚠ "+itoa(s.warnings),
		geometry.Rect{
			Min: geometry.Pt(leftX, 4),
			Max: geometry.Pt(leftX+80, 18),
		},
		11,
		textColor,
		false,
		widget.TextAlignLeft,
	)
	
	// Right side: Line/Col, encoding, language
	rightItems := []string{s.lineCol, "Spaces: 4", s.encoding, s.language}
	totalRightWidth := float32(0)
	for _, item := range rightItems {
		totalRightWidth += float32(len(item)*7 + 20)
	}
	
	x := size.Width - totalRightWidth
	for _, item := range rightItems {
		canvas.DrawText(
			item,
			geometry.Rect{
				Min: geometry.Pt(x, 4),
				Max: geometry.Pt(x+100, 18),
			},
			11,
			textColor,
			false,
			widget.TextAlignLeft,
		)
		x += 100
	}
}

func (s *StatusBar) Event(ctx widget.Context, e event.Event) bool {
	return false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = '0' + byte(n%10)
		n /= 10
	}
	return string(buf[i:])
}

// MainEditor wraps the editor view with layout containers
type MainEditor struct {
	widget.WidgetBase
	titleBar    *TitleBar
	editorView  *editor.EditorView
	statusBar   *StatusBar
	container   *widget.SplitContainer
}

func NewMainEditor(model editor.ITextModel, title string) *MainEditor {
	opts := editor.DefaultOptions()
	editorView := editor.NewEditorView(model, opts)
	
	titleBar := NewTitleBar(title)
	statusBar := NewStatusBar()
	
	// Create a vertical split container
	container := widget.NewSplitContainer(widget.OrientationVertical)
	
	// Add title bar at top
	titleBox := primitives.Box(titleBar)
	container.AddChild(titleBox, 30) // Fixed height
	
	// Editor fills remaining space
	editorBox := primitives.Box(editorView)
	container.AddChild(editorBox, 1) // Flexible
	
	// Status bar at bottom
	statusBox := primitives.Box(statusBar)
	container.AddChild(statusBox, 22) // Fixed height
	
	return &MainEditor{
		titleBar:   titleBar,
		editorView: editorView,
		statusBar:  statusBar,
		container:  container,
	}
}

func (m *MainEditor) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	return m.container.Layout(ctx, c)
}

func (m *MainEditor) Children() []widget.Widget {
	return m.container.Children()
}

func (m *MainEditor) Draw(ctx widget.Context, canvas widget.Canvas) {
	m.container.Draw(ctx, canvas)
}

func (m *MainEditor) Event(ctx widget.Context, e event.Event) bool {
	return m.container.Event(ctx, e)
}

func (m *MainEditor) GetEditorView() *editor.EditorView {
	return m.editorView
}

func (m *MainEditor) GetStatusBar() *StatusBar {
	return m.statusBar
}

func main() {
	// Load the file to display (or a small built-in sample).
	var text string
	var fileName string
	
	if len(os.Args) > 1 {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			log.Fatalf("cannot read %s: %v", os.Args[1], err)
		}
		text = string(data)
		fileName = filepath.Base(os.Args[1])
	} else {
		text = sample
		fileName = "sample.go"
	}

	model := editor.NewTextModel(text)
	title := "Gode Editor - " + fileName
	
	mainEditor := NewMainEditor(model, title)
	
	// Create the application
	gogpuApp := gogpu.NewApp(gogpu.DefaultConfig().
		WithTitle(title).
		WithSize(1200, 800))

	uiApp := app.New(
		app.WithWindowProvider(gogpuApp),
		app.WithPlatformProvider(gogpuApp),
		app.WithEventSource(gogpuApp.EventSource()),
	)

	uiApp.SetRoot(primitives.Box(mainEditor))

	log.Println("Gode Editor starting...")
	log.Println("  - Custom title bar with window controls")
	log.Println("  - Go-based editor rendering (gogpu/ui)")
	log.Println("  - Status bar at bottom")
	log.Println("\nNote: This is Phase 1 of the migration.")
	log.Println("  - Activity bar will be added in Phase 2")
	log.Println("  - Tauri integration for plugins in Phase 3")

	if err := desktop.Run(gogpuApp, uiApp); err != nil {
		log.Fatal(err)
	}
}

const sample = `package main

import (
	"fmt"
	"os"
)

// main is the entry point of the Gode Editor.
func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("Gode Editor - A modern code editor")
		fmt.Println("Usage: gode [file]")
		return
	}
	
	for _, name := range args {
		fmt.Printf("Opening: %s\n", name)
	}
}

// add returns the sum of two integers.
func add(a, b int) int {
	return a + b
}

// multiply returns the product of two integers.
func multiply(a, b int) int {
	return a * b
}
`
