# VS Code 核心部分用 Go 重写计划

## 1. 背景与目标

当前 VS Code 代码库（Code-OSS）基于 TypeScript/Node.js/Electron 构建。我们计划将其核心部分（除扩展宿主外）重写为 Go 语言，从而实现更高效、更现代化的编辑器架构。

### 保留的组件
- **扩展宿主**：完整的 Node.js 扩展宿主进程（`src/vs/workbench/api/node/extensionHostProcess.js`），保留现有 VS Code 扩展(.vsix)兼容性
- **VS Code API 模块**：所有扩展点、RPC 协议、扩展生命周期管理

### 重写目标
- 应用生命周期 (Electron 主进程) → 纯 Go 主进程
- 渲染层 (Electron + WebView) → Go 原生 GUI (gogpu/ui)
- 所有核心服务模块 (配置、生命周期、状态、文件、日志、环境) → Go 原生实现
- 编辑器核心 (文本模型、缓冲区、渲染) → Go 实现
- UI 组件 (命令面板、工作台、状态栏、标题栏) → Go 原生 UI

## 2. 高层架构

```
┌─────────────────────────────────────────────────────┐
│              Gode 应用核心 (Go)                      │
├─────────────────────────────────────────────────────┤
│  应用层 / UI层                                       │
│  ├─ 主窗口 & GUI 事件循环                            │
│  ├─ 命令面板 & 菜单系统                               │
│  ├─ 编辑器视图 & 布局管理                            │
│  ├─ 状态栏 & 标题栏                                 │
│  ├─ 文件浏览器 & 搜索界面                             │
│  └─ 对话框与通知                                   │
├─────────────────────────────────────────────────────┤
│  编辑器核心层                                       │
│  ├─ 文本文档模型 & 缓冲区管理                         │
│  ├─ 光标位置与选择范围                             │
│  ├─ 编辑器实例与布局                                 │
│  ├─ 语法高亮 (基础)                                 │
│  └─ 剪贴板操作                                     │
├─────────────────────────────────────────────────────┤
│  核心服务层                                         │
│  ├─ 配置服务 (settings)                             │
│  ├─ 生命周期服务 (init/exit)                       │
│  ├─ 状态服务 (全局/工作区状态)                   │
│  ├─ 文件系统服务 (读写、目录遍历)               │
│  ├─ 日志服务                                       │
│  ├─ 环境服务 (路径、平台、进程信息)             │
│  ├─ 产品服务 (应用元数据)                         │
│  ├─ 隧道服务 (远程连接)                             │
│  └─ 窗口服务 (多窗口管理)                         │
├─────────────────────────────────────────────────────┤
│  IPC 层 (PersistentProtocol)                        │
│  ├─ 帧协议 (13字节头 + 数据体)                     │
│  ├─ 握手 (init data/Ready/Initialized)             │
│  ├─ RPC 通道 (MainThread*/ExtHost* 服务)           │
│  └─ 心跳与保活                                   │
├─────────────────────────────────────────────────────┤
│  Node.js 扩展宿主 (保留)                            │
│  ├─ src/vs/workbench/api/node/extensionHostProcess.js │
│  ├─ VS Code API 模块 (extHost.api.impl.ts 等)       │
│  └─ 所有扩展兼容性保持不变                        │
└─────────────────────────────────────────────────────┘
```

## 3. 模块划分与 Go 实现

### 3.1 应用层 (cmd/gode/main.go)
- 替代 `src/main.ts` 和 `src/vs/code/electron-main/main.ts`
- 负责 CLI 解析、单例锁、应用初始化
- 创建扩展宿主子进程并建立 IPC 连接

### 3.2 核心服务 (internal/services/)
- **EnvironmentService**：替代 `EnvironmentMainService`
- **ConfigurationService**：替代 `ConfigurationService`
- **LifecycleService**：替代 `LifecycleMainService`
- **StateService**：替代 `StateService`
- **LogService**：替代日志系统
- **ProductService**：应用元数据 (从 product.json 加载)

### 3.3 IPC 协议 (internal/ipc/)
- **PersistentProtocol 实现**：帧编解码、握手逻辑
- **ExtensionHostInitData**：初始化数据结构 (匹配 TypeScript 类型)
- **Connection**：套接字管理、消息循环

### 3.4 扩展宿主管理 (internal/extproc/)
- **Manager**：Node.js 子进程管理
- **Launcher**：设置环境变量、创建 IPC 管道、握手协调
- **Command**：实际执行 `node out/vs/workbench/api/node/extensionHostProcess.js`

### 3.5 RPC 通道 (internal/protocol/)
- **Dispatcher**：路由消息到对应服务
- **MainThread* 服务实现**：命令、窗口、编辑器、文档、工作台、扩展等
- **ExtHost 通道处理**：转发消息到 Node.js 扩展宿主

### 3.6 编辑器核心 (internal/editor/)
- **TextDocument**：文本模型，支持增量更新
- **TextEditorManager**：管理打开的文档
- **ActiveEditor**：当前活动编辑器
- **Clipboard**：剪贴板操作

### 3.7 UI 层 (internal/ui/)
- **App**：基于 gogpu/ui 的窗口管理
- **EditorView**：文本编辑控件
- **CommandPalette**：命令面板
- **StatusBar**：状态栏组件
- **WindowManager**：多窗口支持

## 4. 阶段式实现计划

### 阶段 0 - 脚手架 (已完成)
- Go 模块创建 (`go.mod`)
- 基本目录结构 (`cmd/gode/`, `internal/*/`) 
- 构建系统 (`Makefile`)，支持 `make build`/`make run`
- 扩展宿主检查脚本 (验证 TS 构建产物是否存在)

### 阶段 1 - MVP (核心端到端)
**目标**：实现最小可用系统

#### 1.1 应用核心
- Go 主进程启动，解析命令行参数
- 创建用户数据目录，设置日志
- 检查并启动 Node.js 扩展宿主子进程

#### 1.2 扩展宿主 handshake
- 设置环境变量 `VSCODE_EXTHOST_IPC_HOOK=<pipe>`
- 扩展宿主连接到 Go 创建的 Unix socket
- 发送 `IExtensionHostInitData` (JSON)
- 扩展宿主回复 `Ready`
- Go 发送 `Initialized`
- RPC 通道建立，扩展可注册命令

#### 1.3 基本编辑功能
- 支持打开/编辑/保存本地文本文件
- 多文档标签页管理
- 基本的撤销/重做
- 搜索与替换

#### 1.4 扩展支持
- 扩展可注册命令并执行
- 命令面板显示并执行命令
- 扩展点支持: `onCommand`, `onStartupFinished`

#### 1.5 UI 展示
- 创建主窗口
- 显示菜单栏
- 显示工具栏
- 显示编辑器区域
- 显示状态栏

### 阶段 2 - 编辑器保真
- 高级文本编辑功能 (自动完成、代码折叠、缩进)
- 多语言支持 (语法定义、关键字)
- 打印与查找全部
- 列编辑与块选择
- 查找/替换历史记录

### 阶段 3 - 核心服务完善
- 完善配置服务 (settings.json, workspace settings)
- 文件系统服务支持网络文件系统
- 状态服务持久化
- 日志服务支持旋转
- 环境服务支持便携模式

### 阶段 4 - UI 丰富化
- 完整的命令面板 (历史记录、搜索)
- 工作台布局 (侧边栏、面板、活动项)
- 文件浏览器支持树状结构、多列视图
- 主题支持 (深色/浅色、高对比度)
- 国际化支持 (根据 locale 加载翻译)

### 阶段 5 - 扩展生态完善
- 完整扩展 API 对接 (MainThreadWindow、MainThreadEditors 等)
- 扩展生命周期管理 (激活、停用、更新)
- 扩展通信 (postMessage、registerMessageChannel)
- 扩展权限管理 (键盘快捷键、菜单项)
- 扩展测试支持

## 5. 技术实现要点

### 5.1 IPC 协议对齐
- 使用 VS Code 开源的 `PersistentProtocol` 实现 (src/vs/base/parts/ipc/common/ipc.net.ts)
- 确保二进制协议完全兼容，扩展宿主可以无缝连接
- 握手流程严格遵循现有实现 (init data -> Ready -> Initialized)

### 5.2 RPC 协议映射
- 将 `extHost.protocol.ts` 中的 `MainThread*` 服务接口映射到 Go
- 支持请求/响应模式和事件订阅模式
- 保持相同的错误码和事件类型

### 5.3 文件系统集成
- Go 原生 `os`/`filepath` 包实现基本文件操作
- 支待 UNC 路径 (Windows)
- 实现 VS Code 特殊路径处理 (如 `file://`、`vscode-userdata://`)
- 保持扩展宿主的文件系统权限模型

### 5.4 内存管理
- Go 的垃圾回收适合长运行的编辑器应用
- 优化大文本文档的内存使用 (使用字符串或文本编辑器模式)
- 实现文档缓存与换页

### 5.5 并发模型
- Go 的协程适合事件驱动的 UI 应用
- 使用通道进行线程间通信
- 支持扩展宿主的阻塞操作 (通过通道 + 协程实现)

## 6. 迁移策略与兼容性

### 6.1 渐进迁移
- 先重写核心服务和应用层
- 扩展宿主保持不变，扩展无需修改
- MVP 完成后逐步迁移其他功能
- 保持单源代码仓库，避免分支混乱

### 6.2 接口隔离与桩实现
- 为每个未实现的服务提供桩实现 (Null Service)
- 扩展调用这些接口时不会崩溃，保持稳定性
- 标记为“搁置”的功能在未来逐步实现

### 6.3 验证与测试
- 端到端测试：启动 Go 应用，连接扩展宿主，执行简单扩展
- 功能测试：与原始 VS Code 的行为对比
- 性能测试：评估 Go 实现的性能特点

## 7. 风险与对策

### 7.1 IPC 协议兼容性
- **风险**：二进制协议变化可能导致扩展宿主无法连接
- **对策**：严格复刻现有协议，使用相同的序列化格式

### 7.2 扩展 API 对接
- **风险**：部分扩展可能依赖特定实现细节
- **对策**：为每个 API 提供统一的抽象层，后续实现具体的逻辑

### 7.3 UI 复杂性
- **风险**：Go 原生 UI 可能无法完全模拟 Electron 的功能
- **对策**：MVP 时使用简单的窗口和控件，逐步完善

### 7.4 依赖管理
- **风险**：Go 生态可能缺少某些特定库
- **对策**：使用 Go 原生包，或重新实现核心功能

## 8. 里程碑与交付

### 8.1 短期 (1-2 周)
- MVP 构建完成 (阶段 1)
- 基本文件编辑 + 扩展命令执行
- 扩展宿主完全兼容

### 8.2 中期 (1个月)
- 编辑器功能完善 (阶段 2)
- 核心服务完善 (阶段 3)
- UI 基本可用

### 8.3 长期 (2-3 个月)
- 完整功能对等 (阶段 4 + 5)
- 所有扩展均可用，不需要修改
- 性能与原始版本相当或优于

## 9. 下一步行动

1. **验证 Go 构建**：运行 `make build`，确保可执行
2. **测试扩展宿主连接**：手动编译 TS 代码，运行扩展宿主，验证 Go 侧握手
3. **实现基本 UI**：使用 gogpu/ui 搭建窗口和编辑器区域
4. **实现基本文件服务**：支持打开、编辑、保存
5. **实现扩展命令处理**：支持 `MainThreadCommands` 通道

当前实现状态：
- [x] Go 模块创建及基本构建系统
- [x] IPC 协议实现 (PersistentProtocol)
- [x] 扩展宿主管理器 (extproc)
- [x] RPC 分发器框架
- [x] 基本应用层 (app/main.go)
- [ ] UI 层实现 (gogpu/ui 集成功能)
- [ ] 文件服务实现
- [ ] 编辑器核心实现
- [ ] 扩展通道实现

> **提示**：下一步是验证 Go 构建，运行 `make build` 并测试扩展宿主握手流程。需要先编译 TS 代码 (运行 `make build-ts`)，或使用现有的 `out/` 目录 (如果存在)。
