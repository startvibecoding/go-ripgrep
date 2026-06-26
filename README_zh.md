# go-ripgrep

用 Go 语言编写的高性能面向行的搜索工具 —— [ripgrep](https://github.com/BurntSushi/ripgrep) 的纯 Go 移植版本。它既提供了与 ripgrep 接口兼容的 CLI 工具 (`rg`)，也提供了供编程使用的 Go SDK。

[![Go Version](https://img.shields.io/badge/Go%201.26+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## 特性

- **快速递归搜索** — 使用 goroutine 实现多线程文件遍历和搜索
- **正则和固定字符串匹配** — 完整的 Go `regexp` 支持，加上字面量字符串搜索 (`-F`)
- **忽略文件支持** — 支持 `.gitignore`、`.ignore` 和 `.rgignore`，包括嵌套目录
- **Glob 过滤** — 使用 `-g`/`--glob` 模式包含/排除文件，支持取反
- **上下文行** — 显示匹配行前后的上下文 (`-A`、`-B`、`-C`)
- **彩色输出** — 匹配项、文件名和行号的 ANSI 彩色高亮
- **JSON 输出** — 换行分隔的 JSON (NDJSON) 格式 (`--json`)，兼容 ripgrep 的 JSON 规范
- **标准输入支持** — 可从其他命令通过管道输入
- **跨平台** — 支持 Linux (amd64, arm64, loong64)、macOS (amd64, arm64) 和 Windows (amd64, arm64)
- **NPM 包** — 可通过 npm 分发，用于 Node.js 集成
- **纯 Go SDK** — 在你的 Go 应用中嵌入搜索功能

## 安装

### 从源码构建

```bash
git clone https://github.com/startvibecoding/go-ripgrep.git
cd go-ripgrep
make build
# 二进制文件位于 ./bin/rg
```

### 通过 `go install`

```bash
go install github.com/startvibecoding/go-ripgrep/cmd/rg@latest
```

### 通过 npm

```bash
npm install go-ripgrep
# 二进制文件位于 node_modules/.bin/rg
```

### 预编译二进制

从 [GitHub Releases](https://github.com/startvibecoding/go-ripgrep/releases) 下载对应平台的二进制文件。

## 使用方法

### 命令行 (CLI)

```
rg [选项] 模式 [路径...]
rg [选项] -F 模式 [路径...]
cat 文件 | rg [选项] 模式
```

#### 示例

```bash
# 递归搜索模式
rg "hello" ./src

# 不区分大小写搜索
rg -i "error" /var/log

# 固定字符串搜索（不使用正则）
rg -F "function(" ./src

# 显示上下文行
rg -C 3 "TODO" .

# 仅搜索特定文件类型
rg -g "*.go" "func main" .

# 排除文件模式
rg -g "!*.min.js" "function" ./dist

# JSON 输出
rg --json "pattern" .

# 从标准输入读取
cat README.md | rg "install"

# 全词匹配
rg -w "test" .

# 限制每个文件的匹配数
rg -m 5 "error" /var/log

# 显示列号
rg --column "TODO" .

# 跟随符号链接
rg -L "pattern" ./links

# 搜索隐藏文件
rg --hidden "secret" .

# 忽略 .gitignore 规则
rg --no-ignore "node_modules" .
```

### Go SDK

```go
package main

import (
    "context"
    "fmt"
    goriggrep "go-ripgrep"
    "go-ripgrep/pkg/printer"
)

func main() {
    opts := goriggrep.Options{
        Pattern:         "TODO",
        CaseInsensitive: true,
        // MaxDepth:       3,
        // Globs:          []string{"*.go", "!vendor/"},
        // Threads:        8,
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    results, err := goriggrep.Search(ctx, []string{"./src"}, opts)
    if err != nil {
        panic(err)
    }

    for res := range results {
        fmt.Printf("文件: %s (%d 个匹配)\n", res.Path, res.Stats.Matches)
        for _, m := range res.Matches {
            if !m.IsContext {
                fmt.Printf("  第 %d 行: %s\n", m.LineNum, m.Line)
            }
        }
    }
}
```

## CLI 选项参考

| 选项 | 简写 | 说明 |
|------|------|------|
| `--ignore-case` | `-i` | 不区分大小写搜索 |
| `--case-sensitive` | `-s` | 强制区分大小写（覆盖 `-i`） |
| `--word-regexp` | `-w` | 全词匹配 |
| `--fixed-strings` | `-F` | 将模式视为字面量字符串 |
| `--invert-match` | `-v` | 选择不匹配的行 |
| `--glob GLOB` | `-g` | 通过 glob 模式包含/排除文件 |
| `--after-context NUM` | `-A` | 显示每个匹配后的 NUM 行 |
| `--before-context NUM` | `-B` | 显示每个匹配前的 NUM 行 |
| `--context NUM` | `-C` | 显示每个匹配前后的 NUM 行 |
| `--max-count NUM` | `-m` | 限制每个文件的匹配数 |
| `--threads NUM` | `-j` | 工作线程数 |
| `--hidden` | | 搜索隐藏文件和目录 |
| `--no-ignore` | | 不遵循忽略文件 |
| `--follow` | `-L` | 跟随符号链接 |
| `--max-depth NUM` | | 最大目录深度 |
| `--json` | | 输出换行分隔的 JSON |
| `--color WHEN` | | 彩色输出：`always`、`never`、`auto` |
| `--heading` | | 在文件标题下分组显示匹配 |
| `--no-heading` | | 不打印文件标题 |
| `--line-number` | `-n` | 显示行号（默认开启） |
| `--no-line-number` | `-N` | 不显示行号 |
| `--with-filename` | `-H` | 打印每个匹配的文件路径 |
| `--no-filename` | `-I` | 不打印文件路径 |
| `--column` | | 显示首个匹配的列号 |
| `--help` | `-h` | 打印帮助信息 |
| `--version` | `-V` | 打印版本信息 |

## 架构

```
go-ripgrep/
├── cmd/rg/           # CLI 入口
│   └── main.go       # 参数解析、搜索编排、输出
├── pkg/
│   ├── matcher/      # 模式匹配引擎
│   │   ├── matcher.go    # RegexMatcher、FixedMatcher、BuildMatcher
│   │   └── matcher_test.go
│   ├── searcher/     # 文件读取与逐行搜索
│   │   ├── searcher.go   # 支持上下文的 Searcher
│   │   └── searcher_test.go
│   ├── printer/      # 输出格式化
│   │   ├── printer.go    # CLI 文本和 NDJSON 输出
│   │   └── printer_test.go
│   ├── globset/      # Glob 模式编译与匹配
│   │   ├── globset.go    # GlobToRegex、GlobSet、MatchGlobFilter
│   │   └── globset_test.go
│   └── ignore/       # 忽略文件解析与栈管理
│       ├── ignore.go     # .gitignore、.ignore、.rgignore 支持
│       └── ignore_test.go
├── sdk.go            # 公共 Go SDK（Search、Options）
├── sdk_test.go       # SDK 单元测试
├── tests/
│   └── integration_test.go  # 端到端 CLI 测试
├── npm/              # NPM 包分发
├── scripts/          # 构建和打包脚本
└── Makefile          # 构建系统
```

### 数据流

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  CLI / SDK   │────▶│   遍历器     │────▶│   搜索器     │
│  （选项）    │     │ （目录/文件） │     │ （按文件）   │
└──────────────┘     └──────────────┘     └──────────────┘
                           │                     │
                     ┌─────▼─────┐         ┌─────▼─────┐
                     │  忽略栈   │         │  匹配器   │
                     │ (.gitignore│         │  （正则/  │
                     │  .ignore) │         │   固定）   │
                     └───────────┘         └───────────┘
```

1. **CLI** 解析参数并创建 `Options`
2. **遍历器** 递归遍历目录，通过**忽略栈**遵循 `.gitignore` 规则
3. 文件通过 channel 发送给工作 goroutine
4. **搜索器** 逐行读取每个文件，使用**匹配器**查找匹配项
5. 结果通过 channel 流回**打印机**进行格式化输出

## 构建

```bash
# 当前平台
make build

# 所有平台
make build-all

# 特定平台
make build-linux
make build-darwin
make build-windows

# 静态二进制（musl）
make build-linux-musl

# 运行测试
make test

# 格式化代码
make fmt

# 清理构建产物
make clean
```

## NPM 分发

```bash
# 同步 npm 包版本
make npm-version

# 构建平台特定的 npm 包
make npm-packages

# 打包所有 npm 包
make npm-pack

# 发布所有 npm 包
make npm-publish-all
```

## 退出码

| 代码 | 含义 |
|------|------|
| `0` | 找到匹配 |
| `1` | 未找到匹配 |
| `2` | 错误（无效参数、模式错误等） |

## 与 ripgrep 的比较

| 特性 | ripgrep (Rust) | go-ripgrep (Go) |
|------|----------------|-----------------|
| 语言 | Rust | Go |
| 正则引擎 | Rust `regex` crate | Go `regexp` 标准库 |
| SIMD 优化 | 是 | 是 (amd64 使用 AVX2，arm64 使用 NEON) |
| PCRE2 支持 | 是 | 否 |
| .gitignore 支持 | 是 | 是 |
| JSON 输出 | 是 | 是 |
| Go SDK | 否 | 是 |
| npm 分发 | 通过社区 | 内置 |
| 交叉编译 | Rust 工具链 | `GOOS`/`GOARCH` |

> **注意：** 本项目旨在与 ripgrep 实现 CLI 兼容，但在性能和边界情况行为上可能存在差异。对于大型代码库的最大性能需求，建议使用原版 [ripgrep](https://github.com/BurntSushi/ripgrep)。

## 贡献

1. Fork 本仓库
2. 创建你的特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交你的更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 许可证

本项目基于 MIT 许可证 — 详见 [LICENSE](LICENSE) 文件。

## 致谢

- [ripgrep](https://github.com/BurntSushi/ripgrep) by Andrew Gallant — Rust 原版实现
- [Go 标准库](https://pkg.go.dev/) — `regexp`、`filepath`、`os` 包
