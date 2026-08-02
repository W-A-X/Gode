# Gode Editor (Go)

VS Code 风格代码编辑器的**核心渲染层** Go 实现，基于 [gogpu/ui](https://github.com/gogpu/ui)（纯 Go、零 CGO、WebGPU 渲染）。

对应 VS Code 中 `src/vs/editor/browser/view/` + `src/vs/editor/common/viewModel/` 的职责。

## 架构

```
model.go      ITextModel 接口 + 内存实现 TextModel（对应 VS Code ITextModel）
viewmodel.go  ViewModel：布局、视口、滚动、坐标转换（纯逻辑，无渲染依赖）
view.go       EditorView：gogpu/ui widget——绘制 + 键盘/鼠标交互
colors.go     VS Code dark+ 配色
cmd/demo/     可运行演示程序
```

依赖关系：`view → viewmodel → model`，渲染层只依赖 `ITextModel` 接口。

## 可替换设计

`EditorView` 不持有任何文本状态，全部数据来自 `ITextModel` 接口：

```go
type ITextModel interface {
    LineCount() int
    LineContent(line int) string
    LineMaxColumn(line int) int
    ValueInRange(r Range) string
    OffsetAt(pos Position) int
    PositionAt(offset int) Position
    EOL() string
}
```

要让渲染器对接 VS Code 现有后端（如扩展宿主/编辑器模型），只需实现该接口
（例如通过 IPC/RPC 桥接 `mainThreadDocuments`），无需改动任何渲染代码。

坐标模型与 VS Code 一致：Position 行/列均为 1-based；布局用"典型字符宽度"
（`CharWidth`，每帧从真实字体测量），字形用真实字体渲染——与 VS Code 的做法相同。

## 运行

```bash
go run ./cmd/demo                # 内置示例
go run ./cmd/demo path/to/file   # 打开真实文件
```

macOS 使用 Metal 后端（`GOGPU_GRAPHICS_API=metal`，默认）；Linux 可用
`GOGPU_GRAPHICS_API=vulkan|gles|software`。

## 已实现

- 行文本渲染（制表符展开）、行号列、当前行高亮
- 选区（鼠标拖拽 / Shift+方向键 / 双击选词 / 行号列点击选整行）
- 光标 + 垂直/水平滚动条、滚轮滚动
- 光标导航：方向键（列保持）、Home/End、PageUp/Down、Ctrl/Cmd+Home/End、Ctrl/Cmd+左右（按词）
- 光标自动滚动到可见范围（reveal）

## 测试

```bash
go test ./...
```

`model_test.go` / `viewmodel_test.go` / `view_test.go` 覆盖行模型、偏移换算、
视口/滚动/坐标转换、光标移动与选区逻辑。

## 后续方向

- 文本编辑操作（输入、退格、剪切粘贴）→ 扩展 `ITextModel` 为可变接口
- 语法高亮（token 化 → 按 token 分段绘制）
- 字体注册等宽字体（当前为内嵌 Inter，光标按等宽网格近似）
- 对接 VS Code 后端的 ITextModel 桥（IPC）
