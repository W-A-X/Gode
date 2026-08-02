// Command offscreen-experiment validates the offscreen rendering pipeline:
// a headless gogpu/ui app driving the editor widget into a software gg
// context, then reading the pixels back. This is the core of the VS Code
// embedding bridge.
package main

import (
	"fmt"
	"os"

	"github.com/gogpu/gg"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/render"

	"gode/editor"
)

// fixedWindowProvider satisfies gpucontext.WindowProvider with a fixed
// logical size, so the headless app uses our viewport instead of the
// 800x600 default.
type fixedWindowProvider struct{ w, h int }

func (p *fixedWindowProvider) Size() (int, int)     { return p.w, p.h }
func (p *fixedWindowProvider) ScaleFactor() float64 { return 1.0 }
func (p *fixedWindowProvider) RequestRedraw()       {}

func main() {
	const (
		width  = 480
		height = 300
	)

	model := editor.NewTextModel(sample)
	view := editor.NewEditorView(model, editor.DefaultOptions())

	uiApp := app.New(app.WithWindowProvider(&fixedWindowProvider{width, height}))
	uiApp.SetRoot(view)

	// Software-rendered offscreen context: no window, no GPU device needed.
	cc := gg.NewContext(width, height)
	canvas := render.NewCanvas(cc, width, height)

	uiApp.Frame() // layout
	win := uiApp.Window()
	drawn := win.DrawTo(canvas)

	if err := cc.SavePNG("/tmp/gode-offscreen.png"); err != nil {
		fmt.Fprintln(os.Stderr, "save:", err)
		os.Exit(1)
	}
	fmt.Printf("drawn=%v size=%dx%d\n", drawn, width, height)
}

const sample = `package main

import "fmt"

// hello
func main() {
	fmt.Println("gode offscreen")
}

var answer = 42
`
