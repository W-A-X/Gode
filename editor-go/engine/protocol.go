// Package engine wraps the gogpu/ui headless rendering pipeline and the
// EditorView widget into a frame-based offscreen engine. The host sends
// JSON-line commands on stdin and reads JSON-line events on stdout.
package engine

import "gode/editor"

// Pos is a 1-based position in the text model.
type Pos struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Range is a half-open [Start, End) range.
type Range struct {
	Start Pos `json:"start"`
	End   Pos `json:"end"`
}

// InputKey describes a keyboard event forwarded from the host.
type InputKey struct {
	KeyType string `json:"key_type"` // "press"|"release"|"repeat"
	Key     string `json:"key"`      // gogpu key name, e.g. "A","Up","Enter"
	Rune    string `json:"rune"`     // typed character, or ""
	Shift   bool   `json:"shift"`
	Ctrl    bool   `json:"ctrl"`
	Alt     bool   `json:"alt"`
	Super   bool   `json:"super"`
}

// InputMouse describes a mouse event forwarded from the host.
type InputMouse struct {
	MouseType string  `json:"mouse_type"` // "press"|"release"|"move"|"drag"|"double_click"
	Button    string  `json:"button"`     // "left"|"right"|"middle"
	X         float32 `json:"x"`
	Y         float32 `json:"y"`
	Shift     bool    `json:"shift"`
	Ctrl      bool    `json:"ctrl"`
	Alt       bool    `json:"alt"`
	Super     bool    `json:"super"`
}

// InputWheel describes a scroll/wheel event.
type InputWheel struct {
	DX    float32 `json:"dx"`
	DY    float32 `json:"dy"`
	Shift bool    `json:"shift"`
	Ctrl  bool    `json:"ctrl"`
}

// TokenSpan is a colored column range on a single line. Start and End are
// 1-based columns; the range is [Start, End) and End is exclusive. Color is a
// CSS string ("#rrggbb" or "rgba(r,g,b,a)") resolved by the host from the VS
// Code theme color map.
type TokenSpan struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Color string `json:"color"`
}

// TokenLine carries the token spans for one 1-based line.
type TokenLine struct {
	Line  int         `json:"line"`
	Spans []TokenSpan `json:"spans"`
}

// Command is a JSON-line message sent from the host to the engine.
type Command struct {
	Cmd string `json:"cmd"`

	// open_document
	Text string `json:"text,omitempty"`

	// set_viewport
	Width  int     `json:"width,omitempty"`
	Height int     `json:"height,omitempty"`
	Scale  float32 `json:"scale,omitempty"`

	// set_glyph_margin_width
	GlyphMarginWidth float32 `json:"glyph_margin_width,omitempty"`

	// set_breakpoints
	Breakpoints []int `json:"breakpoints,omitempty"`

	// set_selection
	Anchor *Pos `json:"anchor,omitempty"`
	Active *Pos `json:"active,omitempty"`

	// input
	Type  string      `json:"type,omitempty"` // "key"|"mouse"|"wheel"
	Key   *InputKey   `json:"key,omitempty"`
	Mouse *InputMouse `json:"mouse,omitempty"`
	Wheel *InputWheel `json:"wheel,omitempty"`

	// set_tokens
	Tokens []TokenLine `json:"tokens,omitempty"`

	// get_content
	ID int64 `json:"id,omitempty"`
}

// Event is a JSON-line message sent from the engine to the host.
type Event struct {
	Evt string `json:"evt"`

	// frame
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Data   []byte `json:"data,omitempty"` // RGBA pixels, row-major

	// selection_changed
	Anchor *Pos `json:"anchor,omitempty"`
	Active *Pos `json:"active,omitempty"`

	// edited
	Range    *Range `json:"range,omitempty"`
	EditText string `json:"edit_text,omitempty"`

	// scrolled
	ScrollTop  float32 `json:"scroll_top,omitempty"`
	ScrollLeft float32 `json:"scroll_left,omitempty"`

	// get_content response
	ID      int64  `json:"id,omitempty"`
	Content string `json:"content,omitempty"`
}

// posToEditor converts a protocol Pos to an editor.Position.
func posToEditor(p Pos) editor.Position {
	return editor.Position{Line: p.Line, Column: p.Column}
}

// posFromEditor converts an editor.Position to a protocol Pos.
func posFromEditor(p editor.Position) Pos {
	return Pos{Line: p.Line, Column: p.Column}
}
