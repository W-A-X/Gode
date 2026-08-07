/*
ui provides the Go-based UI layer using gogpu/ui.

Replaces src/vs/workbench/* (UI components, workbench contributions).
This is a thin wrapper around gogpu/ui that provides:
  - Main window
  - Editor area
  - Command palette
  - Status bar (placeholder)

For MVP, this is a minimal stub that logs UI actions.
The full UI implementation depends on confirming the gogpu/ui API.
*/

package ui

import (
	"fmt"
	"log"
	"sync"
)

// Mock gogpu/ui API for MVP
type Event int
type Key int
type Modifier int
type Color int
type Point struct { X, Y int }
type Size struct { Width, Height int }
type Rect struct { X, Y, Width, Height int }
type MouseButton int

const (
	MouseButtonLeft MouseButton = iota
	MouseButtonMiddle
	MouseButtonRight
)

type MouseEvent struct {
	X, Y int
	Button MouseButton
	Shift bool
	Ctrl bool
	Alt bool
	Type Event
}

type Window interface {
	Title(title string)
	SetSize(width, height int)
	SetPos(x, y int)
	Show()
	Close()
	OnMouseMove(func(MouseEvent))
	OnMouseDown(func(MouseEvent))
	OnMouseUp(func(MouseEvent))
	OnKeyDown(func(Key, Modifier))
	OnKeyUp(func(Key, Modifier))
	Loop()
}

type Font struct {
	Family string
	Size int
	Weight int
}

type Renderer interface {
	SetFont(Font)
	SetColor(Color)
	Clear()
	FillRect(Rect)
	DrawText(x, y int, text string)
	Present()
}

// App represents the UI application.
type App struct {
	mu    sync.Mutex
	win   Window
	renderer Renderer
	events chan Event
	shutdown chan struct{}
}

// NewApp creates a new UI application.
func NewApp() *App {
	return &App{
		events: make(chan Event),
		shutdown: make(chan struct{}),
	}
}

// Run starts the UI event loop.
func (a *App) Run() error {
	log.Println("[ui] UI app starting (MVP stub)")
	// TODO: Initialize gogpu/ui window, start render loop
	return nil
}

// Shutdown stops the UI.
func (a *App) Shutdown() {
	close(a.shutdown)
	log.Println("[ui] UI app shutting down")
}

// ShowEditor renders the editor view with the given content.
func (a *App) ShowEditor(content string) {
	log.Printf("[ui] showing editor with %d chars", len(content))
}

// ShowCommandPalette opens the command palette.
func (a *App) ShowCommandPalette(commands []string) {
	log.Printf("[ui] command palette: %d commands", len(commands))
}

// ShowMessage displays a message dialog.
func (a *App) ShowMessage(title, message string) {
	fmt.Printf("[ui] %s: %s\n", title, message)
}

// StatusItem represents a status bar item.
type StatusItem struct {
	Text    string
	Tooltip string
}
