// Command gode-engine is the offscreen rendering service for the VS Code
// embedding bridge.
//
// Two transports are supported:
//
//   - stdin/stdout JSON lines (default): for tests and simple embedding.
//   - WebSocket server (--port N): for the VS Code renderer, which connects
//     from a browser context. Each text message is one JSON command; replies
//     are JSON events including the rendered frame.
package main

import (
        "bufio"
        "encoding/json"
        "flag"
        "fmt"
        "log"
        "net/http"
        "os"
        "sync"

        "github.com/gorilla/websocket"

        "gode/editor/engine"
)

var upgrader = websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool { return true },
}

var (
        wsConn    *websocket.Conn
        wsWriteMu sync.Mutex
)

func main() {
        port := flag.Int("port", 0, "listen on this port as a WebSocket server (0 = stdin/stdout mode)")
        flag.Parse()

        eng := engine.New(800, 600)
        eng.SetOnDidChange(func(r engine.Range, text string) {
                sendEvent(engine.Event{Evt: "edited", Range: &r, EditText: text})
        })

        if *port > 0 {
                runWebSocket(eng, *port)
        } else {
                runStdio(eng)
        }
}

// runStdio reads JSON-line commands from stdin and writes events to stdout.
//
// Unlike the WebSocket transport, stdio does not coalesce input events: each
// command produces an immediate frame.  This keeps the line-based protocol
// simple for tests and scripting.
func runStdio(eng *engine.Engine) {
        scanner := bufio.NewScanner(os.Stdin)
        scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)

        sendEvent(engine.Event{Evt: "ready"})
        for scanner.Scan() {
                line := scanner.Bytes()
                if len(line) == 0 {
                        continue
                }
                var cmd engine.Command
                if err := json.Unmarshal(line, &cmd); err != nil {
                        continue
                }
                // Flush any pending frame before non-input commands so ordering
                // stays consistent (matches WebSocket transport behaviour).
                if cmd.Cmd != "input" {
                        sendFrame(eng)
                }
                if handleCommand(eng, cmd) {
                        return
                }
        }
        // Flush any final pending frame before exiting.
        sendFrame(eng)
}

// runWebSocket serves commands over a WebSocket connection.
//
// Input commands (key / mouse / wheel) are coalesced: the main loop waits
// up to frameCoalesceWindow for more input after each keystroke, then renders
// a single frame covering the whole batch.  Non-input commands (document open,
// viewport change, …) flush any pending frame first so that ordering stays
// consistent.
func runWebSocket(eng *engine.Engine, port int) {
        http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
                c, err := upgrader.Upgrade(w, r, nil)
                if err != nil {
                        log.Printf("upgrade: %v", err)
                        return
                }
                wsWriteMu.Lock()
                wsConn = c
                wsWriteMu.Unlock()
                defer func() {
                        wsWriteMu.Lock()
                        wsConn = nil
                        wsWriteMu.Unlock()
                        c.Close()
                }()

                sendEvent(engine.Event{Evt: "ready"})

                for {
                        // Blocking read, one frame per command. A coalescing deadline was
                        // removed here: gorilla/websocket permanently remembers the first
                        // read error, so after a SetReadDeadline timeout the next
                        // ReadMessage returns the same error immediately, the loop kept
                        // spinning on it, and gorilla panicked after 1000 repeated reads
                        // on the failed connection ("repeated read on failed websocket
                        // connection"). Sending a frame per command is simpler and
                        // matches the stdio transport.
                        _, data, err := c.ReadMessage()
                        if err != nil {
                                return // connection error or close
                        }

                        var cmd engine.Command
                        if err := json.Unmarshal(data, &cmd); err != nil {
                                continue
                        }

                        if handleCommand(eng, cmd) {
                                return
                        }

                        // Input commands are rendered here (sendFrame skips no-ops via
                        // NeedsRedraw); non-input commands already flushed their frame.
                        sendFrame(eng)
                }
        })

        log.Printf("gode-engine listening on ws://127.0.0.1:%d", port)
        if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
                log.Fatal(err)
        }
}

// handleCommand processes one command. Returns true when the engine should exit.
//
// For input commands ("input") the frame is intentionally NOT sent here — the
// WebSocket loop above coalesces multiple input events into a single frame via
// the 2 ms deadline window.  All other commands send their frame immediately.
func handleCommand(eng *engine.Engine, cmd engine.Command) bool {
        switch cmd.Cmd {
        case "open_document":
                eng.SetText(cmd.Text)
                sendFrame(eng)
        case "set_viewport":
                eng.Resize(cmd.Width, cmd.Height, cmd.Scale)
                sendFrame(eng)
        case "set_glyph_margin_width":
                eng.SetGlyphMarginWidth(cmd.GlyphMarginWidth)
                sendFrame(eng)
        case "set_breakpoints":
                eng.SetBreakpoints(cmd.Breakpoints)
                sendFrame(eng)
        case "set_selection":
                if cmd.Anchor != nil && cmd.Active != nil {
                        eng.SetSelection(*cmd.Anchor, *cmd.Active)
                        sendFrame(eng)
                }
        case "focus":
                eng.Focus()
                sendFrame(eng)
        case "set_tokens":
                eng.SetTokens(cmd.Tokens)
                sendFrame(eng)
        case "input":
                switch cmd.Type {
                case "key":
                        if cmd.Key != nil {
                                eng.HandleEvent(engine.BuildKeyEvent(*cmd.Key))
                        }
                case "mouse":
                        if cmd.Mouse != nil {
                                // Check if this is a tab bar event (y < 0 indicates tab area)
                                if cmd.Mouse.Y < 0 {
                                        // Transform to tab bar coordinates
                                        tabMouse := *cmd.Mouse
                                        tabMouse.Y = -tabMouse.Y
                                        if eng.HandleTabEvent(tabMouse) {
                                                // Tab event handled, may need to send tab frame
                                        }
                                } else {
                                        eng.HandleEvent(engine.BuildMouseEvent(*cmd.Mouse))
                                }
                        }
                case "wheel":
                        if cmd.Wheel != nil {
                                eng.HandleEvent(engine.BuildWheelEvent(*cmd.Wheel))
                        }
                }
                // Frame is sent by the WebSocket loop's coalesce logic.
        case "get_content":
                sendEvent(engine.Event{Evt: "content", ID: cmd.ID, Content: eng.GetContent()})
        case "ping":
                sendEvent(engine.Event{Evt: "pong"})
        case "set_tabs":
                if cmd.Tabs != nil {
                        activeIdx := cmd.Tabs.ActiveIdx
                        eng.SetTabs(cmd.Tabs.Tabs, activeIdx)
                        sendTabFrame(eng)
                }
        case "tab_viewport":
                if cmd.TabViewport != nil {
                        eng.SetTabViewport(cmd.TabViewport.Width, cmd.TabViewport.Height, cmd.TabViewport.Scale)
                        sendTabFrame(eng)
                }
        case "shutdown":
                return true
        }
        return false
}

// sendFrame renders and pushes a frame plus the current selection. If the
// engine hasn't changed since the last frame (e.g. a modifier key that
// doesn't redraw), rendering is skipped to save CPU and bandwidth.
func sendFrame(eng *engine.Engine) {
        if !eng.NeedsRedraw() {
                return
        }
        data, ok := eng.Render()
        if !ok {
                return
        }
        w, h := eng.ViewportSize()
        anchor, active := eng.Selection()
        sendEvent(engine.Event{
                Evt:    "frame",
                Width:  w,
                Height: h,
                Data:   data,
                Anchor: &anchor,
                Active: &active,
        })
}

// sendEvent delivers an event over the active transport. In WebSocket mode
// this must be called from the connection handler goroutine.
func sendEvent(ev engine.Event) {
        if wsConn != nil {
                wsWriteMu.Lock()
                defer wsWriteMu.Unlock()
                // Frame pixel data ([]byte) is base64-encoded automatically by
                // encoding/json; the renderer decodes it once with atob.
                _ = wsConn.WriteJSON(ev)
                return
        }
        data, _ := json.Marshal(ev)
        os.Stdout.Write(data)
        os.Stdout.Write([]byte("\n"))
}

// sendTabFrame renders and pushes a tab bar frame event.
func sendTabFrame(eng *engine.Engine) {
        data, ok := eng.RenderTabBar()
        if !ok {
                return
        }
        w, h := eng.TabViewportSize()
        sendEvent(engine.Event{
                Evt:      "tab_frame",
                TabWidth: w,
                TabHeight: h,
                TabData:  data,
        })
}
