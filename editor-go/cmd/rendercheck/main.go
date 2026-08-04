// Temporary visual check for the JetBrains restyle. Renders the editor and
// the tab bar to PNGs so the new palette can be inspected. Not committed.
package main

import (
	"fmt"
	"os"

	"github.com/gogpu/gg"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/render"

	"gode/editor"
)

type fixedWindowProvider struct{ w, h int }

func (p *fixedWindowProvider) Size() (int, int)     { return p.w, p.h }
func (p *fixedWindowProvider) ScaleFactor() float64 { return 1.0 }
func (p *fixedWindowProvider) RequestRedraw()       {}

func main() {
	renderEditor()
	renderTabs()
}

func renderEditor() {
	const (
		width  = 720
		height = 320
	)

	model := editor.NewTextModel(sample)
	view := editor.NewEditorView(model, editor.DefaultOptions())

	uiApp := app.New(app.WithWindowProvider(&fixedWindowProvider{width, height}))
	uiApp.SetRoot(view)

	cc := gg.NewContext(width, height)
	canvas := render.NewCanvas(cc, width, height)

	// Caret on line 4 with a selection across part of it: shows the current-line
	// highlight, the selection, and the caret together.
	view.SetSelection(editor.Position{Line: 4, Column: 1}, editor.Position{Line: 4, Column: 10})

	uiApp.Frame()
	win := uiApp.Window()
	drawn := win.DrawTo(canvas)

	if err := cc.SavePNG("/tmp/jb-editor.png"); err != nil {
		fmt.Fprintln(os.Stderr, "save:", err)
		os.Exit(1)
	}
	fmt.Printf("editor drawn=%v size=%dx%d\n", drawn, width, height)
}

func renderTabs() {
	const (
		width  = 720
		height = 35
	)

	tabs := []editor.TabInfo{
		{Label: "main.go", IsActive: true, IsDirty: true},
		{Label: "server.go", IsDirty: true},
		{Label: "go.mod"},
		{Label: "viewmodel.go"},
		{Label: "model_test.go"},
	}

	draw := func(path string, hoverX float32) {
		tb := editor.NewTabBar()
		tb.SetBounds(geometry.NewRect(0, 0, width, height))
		tb.Update(tabs, 0)
		if hoverX >= 0 {
			tb.HandleEvent(&event.MouseEvent{MouseType: event.MouseMove, Position: geometry.Pt(hoverX, 17)})
		}
		cc := gg.NewContext(width, height)
		canvas := render.NewCanvas(cc, width, height)
		canvas.PushClip(geometry.NewRect(0, 0, width, height))
		tb.Draw(canvas)
		canvas.PopClip()
		if err := cc.SavePNG(path); err != nil {
			fmt.Fprintln(os.Stderr, "save:", err)
			os.Exit(1)
		}
	}

	draw("/tmp/jb-tabs.png", -1)
	draw("/tmp/jb-tabs-hover.png", 185)
	fmt.Println("tabs done")
}

const sample = `package main

import (
	"fmt"
	"net/http"
)

// handler serves the gode demo.
func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, gode!")
}

func main() {
	http.HandleFunc("/", handler)
	fmt.Println("Server starting on :8080")
	http.ListenAndServe(":8080", nil)
}
`
