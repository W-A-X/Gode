# Gode

基于 VS Code（Code-OSS）的编辑器项目，其核心编辑器渲染层由 [Go](https://go.dev/) 重新实现：用 [gogpu/ui](https://github.com/gogpu/ui)（纯 Go、零 CGO、WebGPU 渲染）驱动的**离屏渲染引擎**替换了 VS Code 原生 DOM 编辑器视图。

## 项目结构

```
src/                  VS Code 主体（工作台、扩展系统、语言服务等）
  vs/workbench/contrib/gode/
                      Gode 集成层：GodeView → GodeEngineClient(WebSocket) → gode-engine
editor-go/            Go 渲染引擎（核心）
  engine/             engine.go 无头渲染引擎 + protocol.go JSON-line 协议
  model.go            ITextModel 接口 + 内存实现
  viewmodel.go        ViewModel：布局、视口、滚动、坐标转换（纯逻辑）
  view.go             EditorView：gogpu/ui widget——绘制 + 键盘/鼠标交互
  colors.go           VS Code dark+ 配色
  cmd/gode-engine/    离屏渲染服务（stdin/stdout 或 --port WebSocket）
  cmd/demo/           可运行演示程序
```

## 架构

Go 引擎与 VS Code 主体通过 WebSocket（JSON-line 协议）通信，职责边界如下：

```
workbench.desktop.main.ts
  └─ gode.contribution.ts（gode.enabled=true 时注册 setGodeViewFactory）
       └─ GodeView（继承 View）
            └─ GodeEngineClient（WebSocket）
                 └─ gode-engine 进程（editor-go/engine + view.go）
```

- **VS Code 侧**：保留模型、视图模型与扩展生态；编辑器视图层（`src/vs/editor/browser/view*`）替换为 canvas 渲染，由 Go 引擎接管绘制与交互。
- **Go 侧**：`EditorView` 不持有任何文本状态，全部数据来自 `ITextModel` 接口，视图层只依赖该接口，便于与 VS Code 后端桥接。
- 坐标模型与 VS Code 一致：Position 行/列均为 1-based。

## 运行

### Go 渲染引擎

```bash
cd editor-go
go run ./cmd/demo                # 内置示例
go run ./cmd/demo path/to/file   # 打开真实文件
go test ./...                    # 运行测试
```

macOS 使用 Metal 后端（`GOGPU_GRAPHICS_API=metal`，默认）；Linux 可用 `GOGPU_GRAPHICS_API=vulkan|gles|software`。

### 集成到 VS Code

1. 编译离屏渲染服务：

   ```bash
   cd editor-go && go build -o bin/gode-engine ./cmd/gode-engine
   ```

2. 设置环境变量 `GODE_ENGINE_PATH` 指向 `bin/gode-engine`（默认端口 47810）。

3. 配置 `gode.enabled=true`（默认开启），启动 Code OSS 后编辑器即由 Go 引擎渲染。

详见 [editor-go/README.md](editor-go/README.md) 与 [.trae/documents/gode-editor-fixes.md](.trae/documents/gode-editor-fixes.md)。

## 已实现

- 行文本渲染（制表符展开）、行号列、当前行高亮
- 选区（鼠标拖拽 / Shift+方向键 / 双击选词 / 行号列点击选整行）
- 光标 + 垂直/水平滚动条、滚轮滚动
- 光标导航：方向键（列保持）、Home/End、PageUp/Down、Ctrl/Cmd+Home/End、Ctrl/Cmd+左右（按词）
- 光标自动滚动到可见范围（reveal）
- 编辑回写 VS Code 模型（onDidEdit → pushEditOperations）与模型→引擎全量回写
- 语法高亮：VS Code TextMate 分词结果经 `set_tokens` 同步到引擎，按 token 分段着色
- 滚动归一化：遵循 `mouseWheelScrollSensitivity` 与 `fastScrollSensitivity`（Alt 加速）

## 后续方向

- 增量文本同步（当前为全量回写，正确性优先）
- 文本编辑操作（输入、退格、剪切粘贴）进一步扩展
- 对接 VS Code 后端的 ITextModel 桥（IPC）
