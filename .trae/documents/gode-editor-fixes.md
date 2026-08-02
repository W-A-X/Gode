# 修复 Go 版编辑器三大问题：只读、滚动不自然、语法高亮失效

## 背景（Context）

Gode 项目用 Go 离屏渲染引擎（`editor-go/`）替换 VS Code 原生 DOM 编辑器视图。活路径为：
`workbench.desktop.main.ts` → `gode.contribution.ts`（注册 `setGodeViewFactory`）→ `GodeView`（继承 `View`）→ `GodeEngineClient`（WebSocket）→ `gode-engine` 进程（`editor-go/engine` + `editor-go/view.go`）。

用户反馈该编辑器存在三个问题：① 编辑器表现为只读/编辑失效；② 滚动速度不自然；③ VS Code 自带语法高亮失效。经静态分析定位到 Go 引擎与 TS 集成层的多处根因，本方案在保持「Go 引擎拥有编辑、VS Code 模型为镜像」这一既有架构的前提下进行修复。

> 注：`godeEditorWidget.ts` / `godeService.ts` / `godeRenderer.ts` / `godeContribution.ts` 是未被 `workbench.desktop.main.ts` 接入的原型死代码，本次不动；如需清理另开任务。

***

## 根因分析

### 问题 1：只读 / 编辑失效

* **1a（主因）`engine.Resize()`** **每次视口变化都重建整个 gogpu/ui** **`app`** **并重跑** **`focusEditor()`。** `focusEditor` 合成一次鼠标按下，把选区重置为 `{1,1}`（`engine.go:71`、`view.go:297`）。而 `GodeView.render()` 在 VS Code 每次渲染都调用 `setViewport`（`godeView.ts:122-129`）。只要视口尺寸变化哪怕 1px（如滚动条显隐导致内容宽度变化），光标就被打回第 1 行第 1 列 → 字打在错误位置 → 体验为「只读/坏掉」。

* **1b** **`GodeView.focus()`** **只聚焦 DOM canvas，不通知引擎聚焦** **`EditorView`。** 引擎 `handleKey` 在 `!IsFocused()` 时丢弃所有按键（`view.go:261`）。当编辑器被程序化聚焦（切标签页等）且没有转发 mouse press 时，按键被静默丢弃 → 只读。

* **1c 缺少 VS Code 模型 → 引擎的内容回写。** 仅接了 engine→VS Code（`onDidEdit`→`pushEditOperations`，`godeView.ts:76`）。当 VS Code 自身改模型（撤销/重做、查找替换、外部改动、多光标）时，引擎镜像与 VS Code 模型漂移，后续引擎编辑基于过期位置 → 编辑错位/丢失。

* **1d 选区同步可能环路**：engine→VS Code `setSelections` → `onCursorStateChanged` → engine `setSelection`。需要重入保护。

### 问题 2：滚动不自然

* **2a** **`GodeView.sendWheel`** **直接转发原始** **`e.deltaY/e.deltaX`，未归一化** **`deltaMode`**（`godeEngineClient.ts:111`）。macOS 触控板是像素模式，值常 >3；引擎用 `|delta|<=3` 猜行/像素（`view.go:352`），猜错就把像素当 1x 像素直接用 → 滚动量微小/跳变。

* **2b 忽略 VS Code 的** **`mouseWheelScrollSensitivity`** **与** **`fastScrollSensitivity`（Alt 加速）**，所以手感与 VS Code 不一致。

### 问题 3：语法高亮失效

* **3 引擎每行用单次** **`DrawText`** **+ 硬编码** **`foregroundColor`** **渲染整行**（`view.go:154`、`colors.go`）。引擎无分词器，且从不接收 VS Code 的 TextMate 分词结果。VS Code 的 `model.tokenization` 已产出逐 token 主题色（`TokenizationRegistry.getColorMap()`），但跨不过 WebSocket 边界 → 全文单色。

***

## 实施方案

### Fix 1：可编辑性（TS + Go）

**`editor-go/engine/engine.go`**

* 重构 `Resize`：不再 `app.New` / 不再 `focusEditor`；仅重建 `gg.Context`+`render.Canvas`、更新 `view.Options/VM.Options` 的字号行高（按 scale）。app 与 root 只在 `New` 创建一次。保留 `view`/`view.VM`（滚动、选区跨 resize 不丢）。

* `focusEditor` 仅保留在 `New`（初始聚焦）；新增导出方法 `Focus()`：在不移动光标的前提下对 `EditorView` `RequestFocus`（通过事件或给 `EditorView` 加一个 `Focus(ctx)` 方法）。

**`editor-go/engine/protocol.go`** **+** **`editor-go/cmd/gode-engine/main.go`**

* 协议新增命令 `focus`（`cmd:"focus"`）。`handleCommand` 增加 `case "focus": eng.Focus(); sendFrame(eng)`。

**`editor-go/view.go`**

* 给 `EditorView` 增加可被引擎调用的聚焦途径（如 `RequestFocus` 通过一次合成事件，但不清选区），供 `Focus()` 使用。

**`src/vs/workbench/contrib/gode/browser/godeEngineClient.ts`**

* 新增 `focus()`：发 `{cmd:'focus'}`。

* 新增 `setText(text)`：复用 `openDocument`（全量同步用）。

* `setViewport` 增加「与上次发送相同则跳过」，避免每帧刷屏触发 `Resize`。

**`src/vs/workbench/contrib/gode/browser/godeView.ts`**

* `focus()`：在 `this._canvas.focus()` 之后调用 `this._client.focus()`，让引擎 `EditorView` 真正聚焦（解决 1b）。

* 加重入保护 `_applyingEngineEdit`：`onDidEdit`→`pushEditOperations` 前置位、并在该标志位期间忽略 `onDidChangeContent` 回写，避免环路。

* 加 VS Code 模型 → 引擎内容回写（1c）：监听 `model.onDidChangeContent`；若非本端 `pushEditOperations` 触发，则发 `setText(全量)` 并随后恢复引擎选区（`setSelection`）。全量回写优先保证正确性（增量留作后续优化）。

* `render()` 中 `setViewport` 仅在尺寸真正变化时发（依赖 client 侧去重）。

### Fix 2：滚动（TS + Go）

**`src/vs/workbench/contrib/gode/browser/godeView.ts`**（`sendWheel` 改由 `GodeView` 计算，或给 `GodeEngineClient.sendWheel` 传入归一化后的值）

* 读 `EditorOption.mouseWheelScrollSensitivity`（默认 1）与 `EditorOption.fastScrollSensitivity`（默认 5，`e.altKey` 时启用）；倍率 `mult = sensitivity * (alt ? fast : 1)`。

* 按 `deltaMode` 归一化为像素：`DOM_DELTA_LINE` → `deltaY * lineHeight * mult`（`lineHeight` 取 `EditorOption.lineHeight`，横向用 `fontSize` 近似）；`DOM_DELTA_PAGE` → `deltaY * viewportHeight * mult`；`DOM_DELTA_PIXEL` → `deltaY * mult`。

* 发归一化后的像素 `dx/dy`。

**`editor-go/view.go`** **`handleWheel`**

* 删除 `|delta|<=3` 的行/像素猜测（`view.go:352-357`）；主机现在恒发像素，直接 `v.VM.ScrollBy(dy, dx)`。

### Fix 3：语法高亮（TS + Go）

**协议** **`godeProtocol.ts`** **+** **`editor-go/engine/protocol.go`**

* 新增命令 `set_tokens`，载荷：

  ```
  { cmd:"set_tokens", tokens: [{ line: number, spans: [{ start: number, end: number, color: string }] }] }
  ```

  （列 1-based，`color` 为 `#rrggbb` 或 `rgba(...)`）

**`src/vs/workbench/contrib/gode/browser/godeView.ts`**

* 新增 `_syncTokens()`：对当前可见行范围（`_viewModel` 的可见行，或全量）逐行 `model.tokenization.getLineTokens(line)`，用 `LineTokens.getCount()/getForeground(i)/getEndOffset(i)`，`getForeground` 得 `ColorId` → `TokenizationRegistry.getColorMap()[colorId]` 得 `Color` → 转 css 字符串；组装 `set_tokens` 发送。

* 触发时机：`render()` 时按可见行增量发送（去重）+ 监听 `model.onDidChangeTokens` 重发 + `themeService.onDidColorThemeChange` 重发（颜色随主题变）。

* 引擎未覆盖的行回退到 `foregroundColor`。

**`editor-go/engine/engine.go`** **/** **`editor-go/view.go`**

* 引擎保存 `map[int][]TokenSpan`（行号 → span 列表，含 start/end 列与解析后的 RGBA）。

* `view.go` 的 `Draw` 文本绘制由「整行单次 `DrawText`」改为「按 token span 分段绘制」：对每段，起始 x = `ColumnToX(line, startCol)`（tab 感知），在 `textLeft - scrollLeft + startx` 处用 span 颜色绘制该段子串。为正确处理 tab，按 rune 迭代推进 x（tab → `advanceToTabStop`），同色连续 rune 合并一次 `DrawText`。无 token 的行用 `foregroundColor`。

***

## 涉及文件清单

Go 侧：

* `editor-go/engine/engine.go`（Resize 重构、Focus、set\_tokens 入口）

* `editor-go/engine/protocol.go`（focus、set\_tokens 协议）

* `editor-go/cmd/gode-engine/main.go`（handleCommand 增加 focus、set\_tokens）

* `editor-go/view.go`（handleWheel 像素化、Draw 分段着色、EditorView 聚焦辅助）

TS 侧：

* `src/vs/workbench/contrib/gode/browser/godeView.ts`（focus 同步、选区重入保护、模型→引擎回写、wheel 归一化、token 同步）

* `src/vs/workbench/contrib/gode/browser/godeEngineClient.ts`（focus/setText、setViewport 去重）

* `src/vs/workbench/contrib/gode/common/godeProtocol.ts`（focus、set\_tokens 命令类型）

***

## 验证方法

1. **编译/构建**

   * Go：`cd editor-go && go build ./... && go test ./...`

   * TS：在仓库根用项目既有编译/启动流程（参考 `.github/copilot-instructions.md`），如 `npm run compile` 或启动 Code OSS。
2. **运行时端到端**（需 `gode.enabled=true` 且 `GODE_ENGINE_PATH` 指向编译后的 `gode-engine --port 47810`）

   * 只读：打开文件 → 直接键盘输入应在光标处出现；切走再切回编辑器（程序化聚焦）后仍可输入；撤销/重做后继续输入位置正确。

   * 滚动：触控板与鼠标轮滚动速度与原生 VS Code 一致；按 Alt 加速生效；不出现跳变。

   * 高亮：打开 `.ts/.go/.json` 等文件，关键字/字符串/注释等按当前主题着色；切换主题颜色随之变化。
3. **回归**：选区点击、双击选词、拖拽选择、行号区整行选择仍正常；滚动条 thumb 位置正确。
4. 必要时用 `launch` 技能启动 Code OSS 验证 UI 行为。

