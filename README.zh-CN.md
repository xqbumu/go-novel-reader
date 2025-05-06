# 📚 go-novel-reader: 你的私人命令行小说朗读者！

[English](./README.md) | [中文](./README.zh-CN.md)

**厌倦了盯着屏幕看小说？让 `go-novel-reader` 为你朗读吧！**

这是一个基于命令行的工具，使用 Go 语言编写，可以为你朗读本地存储的小说文件（TXT 或 Markdown 格式）。它能自动识别并分割章节，利用 macOS 内建的 TTS（文本转语音）功能为你“讲故事”，并且能记住你每本小说的阅读进度，精确到段落！

## ✨ 主要特性

*   **多书库管理**: 轻松添加、列出、移除和切换你的小说收藏 (`add`, `list`, `remove`, `switch`)。
*   **智能章节分割**: 自动检测常见的章节标题格式（中文数字、英文 "Chapter X"、Markdown 标题）并进行分割。
*   **流畅 TTS 朗读**: 调用 macOS 的 `say` 命令，逐段朗读选定的章节 (`read`, `next`, `prev`)。
*   **精准进度保存**: 为每本小说单独保存最后阅读的章节和段落索引，下次打开接着听！
*   **自动连播**: 可选配置，读完当前段落/章节后自动开始下一段/章节 (`config auto_next`)。
*   **便捷导航**: 快速查看当前阅读位置和章节列表 (`where`, `chapters`)。
*   **跨平台？(仅限 macOS)**: 由于依赖 macOS 的 `say` 命令，目前仅支持 macOS 系统。

## 🖥️ 平台要求

*   **macOS**: 必须，因为依赖 `say` 命令进行 TTS。
*   **Go**: 需要 Go 编译环境 (例如 Go 1.24 或更高版本) 来构建。

## 🚀 安装

**选项 1: 使用 `go install` (推荐)**

如果你的 Go 环境配置正确 (GOPATH, GOBIN in PATH)，可以直接运行：
```bash
go install github.com/xqbumu/go-novel-reader@latest
```
这将会下载、编译并将 `go-novel-reader` 可执行文件安装到你的 `$GOPATH/bin` (或 `$GOBIN`) 目录下。

**选项 2: 手动构建**

1.  克隆仓库:
    ```bash
    git clone https://github.com/xqbumu/go-novel-reader.git
    cd go-novel-reader
    ```
2.  构建:
    ```bash
    go build
    ```
    这会在当前目录下生成一个名为 `go-novel-reader` 的可执行文件。你可以将其移动到你的 PATH 路径下的任意位置，方便全局调用。

## 💡 使用方法

```bash
# 添加一本新小说到书库，并设为当前活动小说
./go-novel-reader add /path/to/your/novel.txt

# 列出书库中的所有小说及其阅读进度
./go-novel-reader list

# 切换到书库中的第二本小说
./go-novel-reader switch 2

# 列出当前活动小说的所有章节
./go-novel-reader chapters

# 从上次停止的地方继续朗读当前活动小说
./go-novel-reader read

# 从当前活动小说的第 5 章开始朗读
./go-novel-reader read 5

# 朗读当前活动小说的下一章
./go-novel-reader next

# 朗读当前活动小说的上一章
./go-novel-reader prev

# 查看当前活动小说及阅读进度
./go-novel-reader where

# 查看/切换配置项 (例如：自动连播)
./go-novel-reader config          # 查看当前配置
./go-novel-reader config auto_next # 切换 auto_next 的状态 (true/false)

# 获取帮助信息
./go-novel-reader --help
```

## ⚙️ 配置文件

`go-novel-reader` 会在你的用户配置目录下创建文件来存储信息：

*   `~/.config/go-novel-reader/config.json`: 存储书库列表、活动小说路径和应用设置（如 `auto_next`）。
*   `~/.config/go-novel-reader/progress.json`: 存储每本小说的阅读进度（最后阅读的章节和段落索引）。

通常你不需要手动编辑这些文件。

## 🔮 未来可能

*   支持更多 TTS 引擎？
*   跨平台支持？(需要寻找替代 `say` 的方案)
*   更丰富的配置选项？

欢迎提出建议和贡献！

<!-- ## 📜 许可证

本项目采用 [MIT 许可证](LICENSE)。 -->
