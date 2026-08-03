# Gode

基于 VS Code（Code-OSS）的编辑器项目，其核心编辑器渲染层由 [Go](https://go.dev/) 重新实现：用 [gogpu/ui](https://github.com/gogpu/ui)（纯 Go、零 CGO、WebGPU 渲染）驱动的**离屏渲染引擎**替换了 VS Code 原生 DOM 编辑器视图、**标签页栏**、**文件操作后端**和**Git集成后端**。

## 项目结构

```
src/                  VS Code 主体（工作台、扩展系统、语言服务等）
  vs/workbench/contrib/gode/
      common/          协议定义
        godeProtocol.ts          编辑器+标签页协议
        godeServicesProtocol.ts  文件服务+Git服务协议
      browser/         浏览器端实现
        godeView.ts              编辑器Canvas渲染
        godeTabsControl.ts       标签页Canvas渲染
        goFileServiceClient.ts   文件服务WebSocket客户端
        goGitServiceClient.ts    Git服务WebSocket客户端
        godeServicesManager.ts   服务管理器（统一入口）
        gode.contribution.ts     注册入口
editor-go/            Go 渲染引擎（核心）
  engine/             engine.go 无头渲染引擎 + protocol.go JSON-line 协议
  view.go             EditorView 编辑器渲染widget
  tabs.go             TabBar 标签页渲染widget
  cmd/gode-engine/    离屏渲染服务 (端口 47810)
mini-services/       Go 后端服务
  file-service/       文件系统操作服务 (端口 47811)
    index.ts           WebSocket服务器 + 文件操作实现
  git-service/        Git 版本控制服务 (端口 47812)
    index.ts           WebSocket服务器 + Git操作实现
```

## 架构总览

```
┌─────────────────────────────────────────────────────────────────────┐
│                        VS Code Workbench                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─ 编辑器渲染 (gode.enabled=true) ─────────────────────────────┐   │
│  │  GodeView (Canvas) ←→ GodeEngineClient (WS:47810)          │   │
│  │                     ↓                                        │   │
│  │              gode-engine 进程                               │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌─ 标签页渲染 (gode.enabled=true) ─────────────────────────────┐   │
│  │  GodeTabsControl (Canvas) ←→ GodeEngineClient (WS:47810)   │   │
│  │                            ↓                                 │   │
│  │                   gode-engine TabBar                        │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌─ 文件/Git 后端 (gode.services.enabled=true) ─────────────────┐   │
│  │                                                             │   │
│  │  GoFileServiceClient (WS:47811)                             │   │
│  │   ├─ readFile, writeFile, delete, move, copy               │   │
│  │   ├─ listDir, stat, exists                                 │   │
│  │   ├─ watch (文件变更监听)                                   │   │
│  │   └─ search (文件内容搜索)                                  │   │
│  │                    ↓                                       │   │
│  │         file-service 进程 (Go)                              │   │
│  │                                                             │   │
│  │  GoGitServiceClient (WS:47812)                              │   │
│  │   ├─ status, diff, commit, stage, unstage                  │   │
│  │   ├─ branch (list/create/delete/rename)                    │   │
│  │   ├─ checkout, merge, rebase                               │   │
│  │   ├─ push, pull, fetch                                     │   │
│  │   ├─ log, blame                                            │   │
│  │   ├─ stash (push/pop/drop/list)                            │   │
│  │   ├─ tag (create/delete/list)                              │   │
│  │   ├─ clone, remote                                         │   │
│  │   ├─ reset, revert, cherry-pick                             │   │
│  │                    ↓                                       │   │
│  │         git-service 进程 (Go + go-git/v5)                  │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## 配置选项

```jsonc
{
  // 启用 Go 渲染引擎（编辑器 + 标签页）
  "gode.enabled": true,

  // 启用 Go 后端服务（文件操作 + Git 集成）
  "gode.services.enabled": true,

  // 文件服务端口（默认 47811）
  "gode.services.filePort": 47811,

  // Git 服务端口（默认 47812）
  "gode.services.gitPort": 47812
}
```

## 运行

### 1. 编译 Go 服务

```bash
# 编辑器渲染引擎 + 标签页
cd editor-go && go build -o bin/gode-engine ./cmd/gode-engine

# 文件系统服务
cd ../mini-services/file-service && go build -o bin/file-service .

# Git 版本控制服务
cd ../mini-services/git-service && go build -o bin/git-service .
```

### 2. 启动服务

```bash
# 终端1: 编辑器渲染引擎（端口 47810）
./bin/gode-engine --port 47810

# 终端2: 文件系统服务（端口 47811）
./bin/file-service

# 终端3: Git 服务（端口 47812）
./bin/git-service
```

### 3. 启动 VS Code

设置环境变量并启动：
```bash
export GODE_ENGINE_PATH=/path/to/bin/gode-engine
# 然后启动 Code OSS
```

## 已实现功能

### 🎨 编辑器区域 (EditorView) - 端口 47810
- 行文本渲染、行号列、当前行高亮
- 选区、光标导航、滚动条
- 语法高亮（TextMate tokens 同步）

### 📑 标签页栏 (TabBar) - 端口 47810
- 完整 Go 渲染的标签页
- VS Code Dark+ 配色
- 脏标记、关闭按钮、固定标签
- 滚动支持

### 📁 文件系统服务 (file-service) - 端口 47811

| 操作 | 命令 | 说明 |
|------|------|------|
| **读取文件** | `file.read` | 支持多种编码 |
| **写入文件** | `file.write` | 自动创建目录 |
| **追加内容** | `file.append` | 追加到文件末尾 |
| **删除** | `file.delete` | 支持递归删除目录 |
| **移动/重命名** | `file.move` | 跨设备自动回退为复制 |
| **复制** | `file.copy` | 递归复制目录 |
| **创建目录** | `file.mkdir` | 支持递归创建 |
| **获取信息** | `file.stat` | 文件元数据 |
| **列出目录** | `file.list` | 递归列表、过滤扩展名 |
| **检查存在** | `file.exists` | 检查路径是否存在 |
| **监听变化** | `file.watch` | 基于 fsnotify 实时监听 |
| **搜索内容** | `file.search` | 正则/文本搜索文件内容 |
| **路径工具** | `file.resolve/basename/dirname/extname` | 路径处理 |

**技术特点：**
- 使用 `fsnotify` 实现高效的文件监听
- 支持大文件读写（最大 10MB）
- 自动忽略 `.git`, `node_modules` 等目录的搜索
- 多连接共享 watcher 实例

### 🔀 Git 集成服务 (git-service) - 端口 47812

#### 仓库操作
| 操作 | 命令 | 说明 |
|------|------|------|
| **状态** | `git.status` | 分支、暂存、未暂存、冲突 |
| **克隆** | `git.clone` | 支持浅克隆、单分支 |

#### 差异与提交
| 操作 | 命令 | 说明 |
|------|------|------|
| **差异** | `git.diff` | 支持 staged/unstaged/指定文件 |
| **提交** | `git.commit` | 支持 amend |
| **暂存** | `git.stage` | git add |
| **取消暂存** | `git.unstage` | git reset HEAD |

#### 分支操作
| 操作 | 命令 | 说明 |
|------|------|------|
| **列分支** | `git.branch.list` | 本地/远程/全部 |
| **创建分支** | `git.branch.create` | 新建分支 |
| **删除分支** | `git.branch.delete` | 强制删除 |
| **重命名分支** | `git.branch.rename` | 重命名 |
| **切换** | `git.checkout` | 切换分支/commit/tag |

#### 远程操作
| 操作 | 命令 | 说明 |
|------|------|------|
| **列远程** | `git.remote.list` | 列出远程仓库 |
| **添加远程** | `git.remote.add` | 添加远程 |
| **删除远程** | `git.remote.remove` | 删除远程 |
| **设置URL** | `git.remote.set-url` | 修改远程地址 |
| **推送** | `git.push` | 推送到远程 |
| **拉取** | `git.pull` | 拉取并合并/变基 |
| **获取** | `git.fetch` | 获取远程更新 |

#### 历史与注解
| 操作 | 命令 | 说明 |
|------|------|------|
| **日志** | `git.log` | 提交历史（分页/筛选） |
| ** blame** | `git.blame` | 逐行注解 |

#### 其他操作
| 操作 | 命令 | 说明 |
|------|------|------|
| **合并** | `git.merge` | 合并分支 |
| **变基** | `git.rebase` | 变基操作 |
| **储藏** | `git.stash` | push/pop/drop/list/show |
| **标签** | `git.tag` | create/delete/list |
| **重置** | `git.reset` | mixed/soft/hard |
| **还原** | `git.revert` | 还原提交 |
| **摘取** | `git.cherry-pick` | 摘取提交 |

**技术特点：**
- 使用 [go-git/v5](https://github.com/go-git/go-git) 纯 Go 实现
- 不依赖系统 git 命令（部分高级操作除外）
- 完整的类型定义和错误处理
- 支持所有常用 Git 工作流

## 协议定义

### 文件服务协议 (godeServicesProtocol.ts)

```typescript
// 请求格式
interface IFileServiceRequest {
  id: string;       // 请求ID（用于响应匹配）
  cmd: string;      // 命令名
  params?: any;     // 命令参数
}

// 响应格式
interface IFileServiceResponse<T> {
  id: string;
  success: boolean;
  data?: T;         // 返回数据
  error?: string;   // 错误信息
}
```

### Git 服务协议 (godeServicesProtocol.ts)

```typescript
// 请求格式
interface IGitServiceRequest {
  id: string;
  cmd: string;      // 如 'git.status', 'git.commit' 等
  params?: any;
}

// 响应格式
interface IGitServiceResponse<T> {
  id: string;
  success: boolean;
  data?: T;
  error?: string;
}
```

## 删除/替代的原生实现

以下原生功能已被 Go 后端替代：

| 原生实现 | 替代方案 | 条件 |
|----------|----------|------|
| Node.js `fs` 操作 | Go file-service | `gode.services.enabled=true` |
| Electron `dialog.showOpenDialog` | Go file-service | `gode.services.enabled=true` |
| chokidar 文件监听 | Go fsnotify | `gode.services.enabled=true` |
| ripgrep 搜索 | Go regexp | `gode.services.enabled=true` |
| vscode.git 扩展 | Go git-service (go-git) | `gode.services.enabled=true` |
| MultiEditorTabsControl | Go TabBar | `gode.enabled=true` |
| SingleEditorTabsControl | Go TabBar | `gode.enabled=true` |

## 后续方向

### 短期
- [ ] 增量文本同步（当前为全量回写）
- [ ] 标签页拖拽排序
- [ ] 实际文件类型图标渲染
- [ ] Git 图形化日志（类似 gitk）

### 中期
- [ ] 对接 VS Code 后端的 ITextModel 桥（IPC）
- [ ] LSP（语言服务器协议）Go 实现
- [ ] 扩展系统 Go 沙箱运行时
- [ ] 远程开发 Go 代理

### 长期
- [ ] 完全脱离 Electron 的独立运行时
- [ ] WebAssembly 编译目标支持
- [ ] 移动端适配
