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

// TabInfo describes a single editor tab for the protocol.
type TabInfo struct {
	Label       string `json:"label"`        // Display name (filename)
	Description string `json:"description"` // Full path or tooltip
	IconName    string `json:"icon_name"`   // File type icon identifier
	IsDirty     bool   `json:"is_dirty"`    // Unsaved changes indicator
	IsActive    bool   `json:"is_active"`   // Currently focused tab
	IsPinned    bool   `json:"is_pinned"`   // Pinned/sticky state
}

// TabCommand is the payload for set_tabs command.
type TabCommand struct {
	Tabs      []TabInfo `json:"tabs"`      // List of tabs
	ActiveIdx int       `json:"active_idx"` // Active tab index (-1 if none)
}

// TabViewportCmd sets the tab bar rendering area.
type TabViewportCmd struct {
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Scale  float32 `json:"scale"`
}

// --- Full IDE Layout Protocol ---

// ActivityBarItem represents an extension view container in the activity bar.
type ActivityBarItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Icon       string `json:"icon"`
	BadgeCount int    `json:"badge_count"`
	IsVisible  bool   `json:"is_visible"`
}

// SidebarItem represents a file or directory in the sidebar tree.
type SidebarItem struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Path        string        `json:"path"`
	IsDirectory bool          `json:"is_directory"`
	IsExpanded  bool          `json:"is_expanded"`
	Children    []SidebarItem `json:"children,omitempty"`
	Icon        string        `json:"icon"`
}

// PanelTab represents a panel tab (terminal, output, problems, etc.).
type PanelTab struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	IsActive    bool   `json:"is_active"`
	ContentType string `json:"content_type"`
}

// AuxiliaryTab represents an auxiliary bar tab (chat, agents, etc.).
type AuxiliaryTab struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	IsActive bool   `json:"is_active"`
}

// StatusItem represents a status bar item.
type StatusItem struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	Icon       string `json:"icon,omitempty"`
	Alignment  string `json:"alignment"` // "left" | "right"
	Tooltip    string `json:"tooltip,omitempty"`
}

// TitleState represents the title bar state.
type TitleState struct {
	Title           string `json:"title"`
	Subtitle        string `json:"subtitle"`
	SidebarVisible  bool   `json:"sidebar_visible"`
	PanelVisible    bool   `json:"panel_visible"`
	AuxiliaryVisible bool  `json:"auxiliary_visible"`
}

// WorkbenchViewport represents the full window rendering dimensions.
type WorkbenchViewport struct {
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Scale  float32 `json:"scale"`
}

// LayoutState represents which parts are visible.
type LayoutState struct {
	ActivitybarVisible  bool `json:"activitybar_visible"`
	SidebarVisible      bool `json:"sidebar_visible"`
	PanelVisible        bool `json:"panel_visible"`
	AuxiliarybarVisible bool `json:"auxiliarybar_visible"`
	StatusbarVisible    bool `json:"statusbar_visible"`
}

// CommandItem represents a command in the command palette.
type CommandItem struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Shortcut string `json:"shortcut,omitempty"`
	Category string `json:"category,omitempty"`
}

// ExecuteCommandPayload represents a command execution request.
type ExecuteCommandPayload struct {
	CommandID string        `json:"command_id"`
	Args      []interface{} `json:"args,omitempty"`
}

// NotificationPayload represents a notification from the engine.
type NotificationPayload struct {
	Message string `json:"message"`
	Level   string `json:"level"` // "info" | "warning" | "error"
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

	// set_tabs - update tab bar
	Tabs *TabCommand `json:"tabs,omitempty"`

	// tab_viewport - set tab bar rendering dimensions
	TabViewport *TabViewportCmd `json:"tab_viewport,omitempty"`

	// --- Full IDE layout commands ---

	// set_workbench_viewport - set full window rendering dimensions
	WorkbenchViewport *WorkbenchViewport `json:"workbench_viewport,omitempty"`

	// set_activitybar_items - update activity bar items
	ActivityBarItems []ActivityBarItem `json:"activitybar_items,omitempty"`

	// set_sidebar_items - update sidebar file tree
	SidebarItems []SidebarItem `json:"sidebar_items,omitempty"`

	// set_panel_tabs - update panel tabs
	PanelTabs []PanelTab `json:"panel_tabs,omitempty"`

	// set_auxiliary_tabs - update auxiliary bar tabs
	AuxiliaryTabs []AuxiliaryTab `json:"auxiliary_tabs,omitempty"`

	// set_status_items - update status bar items
	StatusItems []StatusItem `json:"status_items,omitempty"`

	// set_title - update title bar state
	TitleState *TitleState `json:"title_state,omitempty"`

	// set_layout_state - update which parts are visible
	LayoutState *LayoutState `json:"layout_state,omitempty"`

	// execute_command - execute a VS Code command from engine
	ExecuteCommand *ExecuteCommandPayload `json:"execute_command,omitempty"`

	// show_command_palette - request command palette
	ShowCommandPalette bool `json:"show_command_palette,omitempty"`

	// set_command_palette_items - populate command palette
	CommandPaletteItems []CommandItem `json:"command_palette_items,omitempty"`

	// notification from engine
	Notification *NotificationPayload `json:"notification,omitempty"`
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
	Range    *Range  `json:"range,omitempty"`
	EditText string  `json:"edit_text,omitempty"`

	// scrolled
	ScrollTop  float32 `json:"scroll_top,omitempty"`
	ScrollLeft float32 `json:"scroll_left,omitempty"`

	// get_content response
	ID      int64  `json:"id,omitempty"`
	Content string `json:"content,omitempty"`

	// tab_frame - rendered tab bar frame data
	TabWidth  int    `json:"tab_width,omitempty"`
	TabHeight int    `json:"tab_height,omitempty"`
	TabData   []byte `json:"tab_data,omitempty"` // RGBA pixels for tab bar

	// tab_selected - user clicked a tab
	TabSelectedIdx int `json:"tab_selected_idx,omitempty"`

	// tab_close - user clicked close button on a tab
	TabCloseIdx int `json:"tab_close_idx,omitempty"`

	// --- Full IDE layout events ---

	// activitybar_selected - user clicked an activity bar item
	ActivityBarSelectedID string `json:"activitybar_selected_id,omitempty"`

	// sidebar_item_selected - user selected a file in sidebar
	SidebarItemPath string `json:"sidebar_item_path,omitempty"`

	// sidebar_item_toggle - user expanded/collapsed a directory
	SidebarItemToggle *struct {
		Path     string `json:"path"`
		Expanded bool   `json:"expanded"`
	} `json:"sidebar_item_toggle,omitempty"`

	// panel_tab_selected - user selected a panel tab
	PanelTabSelectedID string `json:"panel_tab_selected_id,omitempty"`

	// auxiliary_tab_selected - user selected an auxiliary tab
	AuxiliaryTabSelectedID string `json:"auxiliary_tab_selected_id,omitempty"`

	// titlebar_action - user clicked a title bar action
	TitlebarAction *struct {
		Action string `json:"action"`
	} `json:"titlebar_action,omitempty"`

	// command_palette_requested - engine wants command palette
	CommandPaletteRequested bool `json:"command_palette_requested,omitempty"`

	// command_selected - user selected a command from palette
	CommandSelectedID string `json:"command_selected_id,omitempty"`

	// status_item_clicked - user clicked a status bar item
	StatusItemClickedID string `json:"status_item_clicked_id,omitempty"`

	// input_bar_submit - user submitted input in the input bar
	InputBarText string `json:"input_bar_text,omitempty"`

	// engine_ready - engine finished initialization
	EngineReady bool `json:"engine_ready,omitempty"`
}

// posToEditor converts a protocol Pos to an editor.Position.
func posToEditor(p Pos) editor.Position {
	return editor.Position{Line: p.Line, Column: p.Column}
}

// posFromEditor converts an editor.Position to a protocol Pos.
func posFromEditor(p editor.Position) Pos {
	return Pos{Line: p.Line, Column: p.Column}
}
