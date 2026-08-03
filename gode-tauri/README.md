# Gode Editor - Tauri + Go Implementation

A modern code editor built with Tauri for window management and Go (gogpu/ui) for high-performance editor rendering.

## Project Status: Phase 1 Complete ✅

### Completed Components

#### 1. Go Editor Renderer (`cmd/gode/main.go`)
- **Custom Title Bar**: macOS-style traffic light window controls (red/yellow/green)
- **Editor Area**: Full gogpu/ui based text rendering with VS Code compatibility
- **Status Bar**: VS Code blue status bar with git branch, errors/warnings, line/column info
- **Layout System**: Vertical split container managing all UI regions

#### 2. Reusable Widgets (`src/`)
- **`activitybar.go`**: Bottom activity bar widget with icons for Explorer, Search, Source Control, Run, Extensions, Settings
- **`layout.go`**: Window layout system supporting bottom activity bar configuration

#### 3. Tauri Backend (`src-tauri/`)
- **Rust IPC commands**: `open_file`, `save_file`, `show_plugins_window`
- **Plugin marketplace stub**: Ready for extension integration
- **Tauri v2 configuration**: Modern Tauri setup with custom protocol

#### 4. Frontend UI (`src/index.html`)
- **Complete HTML/CSS layout**: Title bar, editor canvas, bottom activity bar, status bar
- **Plugins marketplace overlay**: Modal window for extension management
- **Window control handlers**: Minimize, maximize, close functionality

## Architecture

```
gode-tauri/
├── cmd/gode/           # Main Go application
│   └── main.go         # TitleBar + EditorView + StatusBar
├── src/                # Reusable Go widgets
│   ├── activitybar.go  # Bottom activity bar widget
│   ├── layout.go       # Window layout system
│   └── index.html      # Tauri frontend (Phase 3)
├── src-tauri/          # Tauri (Rust) backend
│   ├── src/main.rs     # IPC commands
│   ├── Cargo.toml      # Rust dependencies
│   ├── build.rs        # Tauri build script
│   └── tauri.conf.json # Tauri configuration
├── go.mod              # Go module definition
└── package.json        # Node.js dependencies
```

## Layout Design

**Gode Editor Layout:**
```
┌─────────────────────────────────────┐
│ Custom Title Bar (30px)             │
│ [Title]                      🔴🟡🟢 │
├─────────────────────────────────────┤
│                                     │
│         Editor Area                 │
│      (Go gogpu/ui renderer)         │
│                                     │
├─────────────────────────────────────┤
│ Activity Bar (Bottom, 48px)         │
│ 📁 🔍 🌿 ▶️ 🧩 ⚙️                  │
├─────────────────────────────────────┤
│ Status Bar (22px, VS Code blue)     │
│ 🌿 main  ⊗ 0 ⚠ 0    Ln 1, Col 1    │
└─────────────────────────────────────┘
```

## Running the Application

### Prerequisites
- Go 1.21+
- Rust 1.70+ (for Tauri backend)
- Node.js 18+ (for Tauri CLI)

### Phase 1: Run Go Editor (Current)

```bash
cd /workspace/gode-tauri/cmd/gode
go run .

# Or open a specific file
go run . /path/to/file.go
```

### Phase 2: Add Bottom Activity Bar Integration

The activity bar widget is ready in `src/activitybar.go`. Integration steps:
1. Import `activitybar` package in `main.go`
2. Create `ActivityBar` instance
3. Add to split container between editor and status bar
4. Wire up view switching callbacks

### Phase 3: Full Tauri Integration

```bash
# Install dependencies
npm install

# Run in development mode (Tauri + Go renderer)
npm run dev

# Build production binary
npm run build
```

## IPC Communication

Tauri commands available from frontend:

```rust
// Open a file and return content
#[tauri::command]
fn open_file(path: String) -> Result<EditorResponse, String>

// Save content to a file
#[tauri::command]
fn save_file(path: String, content: String) -> Result<EditorResponse, String>

// Show plugins marketplace window
#[tauri::command]
fn show_plugins_window() -> Result<(), String>
```

## Key Differences from VS Code

| Feature | VS Code | Gode Editor |
|---------|---------|-------------|
| Window Management | Electron | Tauri (Rust) |
| Editor Rendering | DOM/Canvas | Go (gogpu/ui WebGPU) |
| Activity Bar | Left sidebar | Bottom bar |
| Plugin Marketplace | Webview | Tauri native window |
| Language Support | TypeScript | Go + Rust |

## Next Steps

### Phase 2 (In Progress)
- [ ] Integrate bottom activity bar into main editor
- [ ] Implement view switching (Explorer, Search, etc.)
- [ ] Add file explorer panel
- [ ] Connect activity bar icons to view panels

### Phase 3 (Planned)
- [ ] Enable Tauri window management
- [ ] Implement plugins marketplace with real data
- [ ] Add file dialog integration
- [ ] Settings synchronization

### Future Enhancements
- Syntax highlighting via TextMate grammars
- Extension API compatibility layer
- Multi-root workspace support
- Remote development support

## Development Notes

### Color Scheme
All colors match VS Code Dark+ theme:
- Background: `#1E1E1E`
- Editor background: `#1E1E1E`
- Status bar: `#007ACC`
- Activity bar: `#252526`
- Title bar: `#3C3C3C`

### Dependencies
- **gogpu/ui**: Pure Go WebGPU UI framework
- **Tauri v2**: Native window management
- **Zero CGO**: Full Go compilation without C dependencies

## License

MIT License - See LICENSE file for details
