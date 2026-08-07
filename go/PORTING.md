# Gode Porting Status Tracker

This document tracks the progress of porting VS Code (Code-OSS) from TypeScript/Node.js/Electron to Go.

## Architecture Overview

```
┌─────────────────────────────────────┐
│  Go binary (gode)                    │
├──────────────┬──────────────────────┤
│ App lifecycle│ internal/app         │
│ IPC protocol │ internal/ipc         │
│ Ext proc     │ internal/extproc     │
│ RPC dispatch │ internal/protocol    │
│ Editor model │ internal/editor      │
│ UI (gogpu)  │ internal/ui           │
│ Services     │ internal/services    │
├──────────────┴──────────────────────┤
│  IPC (named pipe / unix socket)      │
├─────────────────────────────────────┤
│  Node.js extension host (preserved)  │
│  out/vs/workbench/api/node/*.js      │
└─────────────────────────────────────┘
```

## Extension Host Protocol Channels

Status legend: 🟢 Done  🟡 WIP  🔴 Not started  ⚪ Shelved (null impl)

### MainThread Channels (Go → implements, ExtHost → calls)

| Channel | Status | Notes |
|---------|--------|-------|
| `MainThreadCommands` | 🔴 | Command registration/execution |
| `MainThreadWindow` | 🔴 | Window/title/showMessage dialogs |
| `MainThreadEditors` | 🔴 | Text editor management |
| `MainThreadDocuments` | 🔴 | Text document tracking |
| `MainThreadWorkspace` | 🔴 | Workspace/config access |
| `MainThreadLanguages` | ⚪ | Language feature registration |
| `MainThreadDebug` | ⚪ | Debug adapter host |
| `MainThreadSCM` | ⚪ | Source control |
| `MainThreadTerminal` | ⚪ | Terminal PTY |
| `MainThreadSearch` | ⚪ | Text search |
| `MainThreadTask` | ⚪ | Task execution |
| `MainThreadNotifications` | ⚪ | Notification handling |
| `MainThreadStatus` | ⚪ | Status bar items |
| `MainThreadProgress` | ⚪ | Progress reporting |
| `MainThreadTreeViews` | ⚪ | Tree view provider |
| `MainThreadWebviews` | ⚪ | Webview panels |
| `MainThreadExtension` | 🔴 | Extension lifecycle control |
| `MainThreadExtensionTests` | ⚪ | Extension test runner |

### Core Platform Services

| Service | Source File | Go Status |
|---------|-------------|-----------|
| Environment | `environmentMainService.ts` | 🟡 |
| Lifecycle | `lifecycleMainService.ts` | 🔴 |
| Configuration | `configurationService.ts` | 🟡 |
| State | `stateService.ts` | 🟡 |
| File I/O | `diskFileSystemProvider.ts` | 🔴 |
| Logging | `log.ts` | 🟡 |
| Telemetry | `telemetry.ts` | ⚪ |
| Policy | `policy.ts` | ⚪ |
| Product | `productService.ts` | ⚪ |

### Workbench Components

| Component | Source Path | Go Status |
|-----------|-------------|-----------|
| Workbench shell | `src/vs/workbench/workbench.desktop.main.ts` | ⚪ |
| Editor panes | `src/vs/workbench/contrib/editor/` | ⚪ |
| Command palette | `src/vs/workbench/contrib/commands/` | ⚪ |
| Activity bar | `src/vs/workbench/contrib/activitybar/` | ⚪ |
| Status bar | `src/vs/workbench/contrib/statusbar/` | ⚪ |
| Sidebar views | `src/vs/workbench/contrib/views/` | ⚪ |
| Title bar | `src/vs/workbench/contrib/title/` | ⚪ |

## MVP Scope (Phase 1)

The MVP goal is to get a minimal end-to-end chain working:
1. Go boots and creates a window (gogpu/ui)
2. Can open/edit/save a file
3. Launches the Node extension host
4. Handshake completes (init data + Ready + Initialized)
5. Extensions can register commands and call `vscode.commands.executeCommand`
6. Command palette shows registered commands

Channels needed for MVP:
- `MainThreadExtension` (lifecycle: start/fold extensions)
- `MainThreadCommands` (register/execute commands)
- `MainThreadWindow` (basic showMessage)
- `MainThreadEditors` + `MainThreadDocuments` (minimal for editor context)

## Notes
- The Node extension host (`out/vs/workbench/api/node/`) is compiled by the existing TS toolchain.
- Go launches Node with `VSCODE_EXTHOST_IPC_HOOK=<pipe>` env var.
- IPC protocol: VSCode PersistentProtocol framing (13-byte header + body).
- RPC protocol: channel-based JSON messages inside the protocol frames.
