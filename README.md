# ChineseSubFinder

基于开源项目 [ChineseSubFinder](https://github.com/ChineseSubFinder/ChineseSubFinder) 的学习与交流分支，用于中文字幕的自动扫描、下载、整理与基础管理。

## 致敬原项目

本仓库基于原项目 `ChineseSubFinder` 修改而来。
感谢原作者及所有贡献者在字幕检索、媒体库扫描和自动化处理方面的长期投入。

- 原项目仓库：<https://github.com/ChineseSubFinder/ChineseSubFinder>
- 本仓库中的后续修改，建立在原项目已有工作的基础上

## 用途说明

本仓库仅供技术交流、学习研究和个人实验使用。
请勿将本项目用于任何侵犯版权、绕过授权或其他不合规用途。
如涉及影片、字幕和媒体资源，请自行确认来源合法性，并支持正版内容。

## 项目简介

ChineseSubFinder 是一个面向电影与剧集媒体库的中文字幕自动检索与管理工具。
当前仓库保留的核心能力包括：

- 扫描电影与剧集目录
- 自动检索和整理中文字幕
- 通过 Web 页面统一管理配置
- 支持代理、TMDB、字幕源等参数设置

## 使用介绍

### Docker 一键部署

当前仓库已经提供源码直出镜像的默认链路，直接在仓库根目录执行：

```bash
docker compose up -d --build
```

默认会：

- 使用仓库当前源码构建前端和后端
- 生成你自己的 `chinesesubfinder` 镜像，而不是下载上游官方 release
- 启动容器 `chinesesubfinder`

可选构建参数：

```bash
APP_VERSION=v0.55.4-local GOPROXY=https://goproxy.cn,direct docker compose up -d --build
```

运行配置见根目录 [compose.yaml](compose.yaml)。

### 1. 准备媒体目录

先准备电影和剧集目录，并确保程序可以访问这些路径。

- [电影目录结构示例](https://github.com/ChineseSubFinder/ChineseSubFinder/blob/docs/DesignFile/%E7%94%B5%E5%BD%B1%E5%92%8C%E8%BF%9E%E7%BB%AD%E5%89%A7%E7%9B%AE%E5%BD%95%E7%BB%93%E6%9E%84%E7%A4%BA%E4%BE%8B.md)
- [连续剧目录结构要求](https://github.com/ChineseSubFinder/ChineseSubFinder/blob/docs/DesignFile/%E8%BF%9E%E7%BB%AD%E5%89%A7%E7%9B%AE%E5%BD%95%E7%BB%93%E6%9E%84%E8%A6%81%E6%B1%82.md)

### 2. 完成基础配置

在设置页面中完成以下内容：

- 电影目录
- 剧集目录
- 字幕源配置
- TMDB 配置
- 代理配置（如需要）

### 3. 进入 Web 管理界面

启动程序或容器后，通过 Web 页面完成系统初始化、参数调整、任务查看和日志排查。

### 4. 执行扫描与字幕处理

配置完成后执行媒体库扫描，系统会根据任务流程处理字幕匹配、下载和整理。

### 5. 常用文档

- Docker 部署：见 [docker/readme.md](docker/readme.md)
- Windows 使用说明：见 [官方文档](https://github.com/ChineseSubFinder/ChineseSubFinder/blob/docs/DesignFile/v0.20%E6%95%99%E7%A8%8B/01.%E5%A6%82%E4%BD%95%E5%9C%A8Windows%E4%B8%8A%E4%BD%BF%E7%94%A8.md)
- 使用教程：见 [官方文档目录](https://github.com/ChineseSubFinder/ChineseSubFinder/tree/docs/DesignFile/%E4%BD%BF%E7%94%A8%E6%95%99%E7%A8%8B)

## 开发验证

Windows 本地执行 `go test ./...` 前，请确保 `CGO_ENABLED=1`，并且 `PATH` 中包含可用的 MinGW `gcc`。

```bash
go test ./...
cd frontend
npm run build
```

## Docker 说明

- 根目录 `Dockerfile` 是当前仓库推荐的可部署构建链路
- 根目录 `compose.yaml` 是默认启动入口
- `docker/full-release.Dockerfile` 仍保留作历史文件，不适合本仓库源码直出部署

## 说明

本仓库不是官方发布版本，也不应被描述为“原项目官方更新”。
如果你需要长期稳定使用，请优先关注原项目及其正式维护版本。

## License

请保留原项目许可证、原作者署名和相关版权说明。
