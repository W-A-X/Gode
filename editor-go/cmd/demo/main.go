// Command demo launches a minimal Go editor window built on gogpu/ui that
// renders an ITextModel with the VS Code-style view layer in this module.
//
// Usage:
//
//	go run ./cmd/demo [file]
package main

import (
	"log"
	"os"
	"path/filepath"

	_ "github.com/gogpu/gg/gpu"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/desktop"
	"github.com/gogpu/ui/primitives"

	"gode/editor"
)

func main() {
	// Load the file to display (or a small built-in sample).
	var text string
	if len(os.Args) > 1 {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			log.Fatalf("cannot read %s: %v", os.Args[1], err)
		}
		text = string(data)
	} else {
		text = sample
	}

	model := editor.NewTextModel(text)
	opts := editor.DefaultOptions()
	editorView := editor.NewEditorView(model, opts)

	// The editor is the replaceable surface: NewEditorView accepts any
	// ITextModel, so a bridge to the VS Code extension host over IPC can be
	// swapped in here without touching the rendering code.

	title := "Gode Editor"
	if len(os.Args) > 1 {
		title = filepath.Base(os.Args[1])
	}

	gogpuApp := gogpu.NewApp(gogpu.DefaultConfig().
		WithTitle(title).
		WithSize(960, 640))

	uiApp := app.New(
		app.WithWindowProvider(gogpuApp),
		app.WithPlatformProvider(gogpuApp),
		app.WithEventSource(gogpuApp.EventSource()),
	)

	uiApp.SetRoot(primitives.Box(editorView))

	if err := desktop.Run(gogpuApp, uiApp); err != nil {
		log.Fatal(err)
	}
}

const sample = `package main

import (
	"fmt"
	"os"
)

// main is the entry point of the demo program.
func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		args = []string{"world"}
	}
	for _, name := range args {
		fmt.Printf("Hello, %s!\n", name)
	}
}

func add(a, b int) int {
	return a + b
}
`
