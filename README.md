# ChineseSubFinder

基于开源项目 [ChineseSubFinder](https://github.com/ChineseSubFinder/ChineseSubFinder) 持续修改的分支，用于中文字幕的扫描、下载、整理与基础管理。

## 鸣谢原作者

当前仓库建立在上游 `ChineseSubFinder` 的长期工作基础上。
感谢原作者和所有贡献者在媒体库扫描、字幕检索、自动整理和 WebUI 等方面的持续投入。

- 上游仓库：<https://github.com/ChineseSubFinder/ChineseSubFinder>
- 当前仓库：<https://github.com/morningstar-ski/ChineseSubFinder-optimization>

## 相比原仓库的当前增强

- 扩展并接入更多字幕源：`OpenSubtitles`、`TVsubtitles`、`Moviesubtitles`、`SubHD`、`SubtitleCat`。
- 英文字幕回退链默认保留 `SubtitleCat`，在缺少直链字幕时补足英文源。
- 增加中文字幕翻译回退能力，可按配置使用 `SubtitleCat` 远端翻译或 LLM 翻译链。
- `SubHD` 下载链补了本地 `ddddocr` 与 SVG 直读能力，并保留外部 OCR 显式开关。
- WebUI、Docker 文档和问题反馈入口已统一对齐到当前仓库。

## 用途说明

本仓库仅用于技术交流、学习研究和个人实验。
请勿将其用于侵犯版权、绕过授权或其他不合规用途。涉及影视、字幕和媒体资源时，请自行确认来源合法性并支持正版内容。

## Docker 部署

### 直接使用当前仓库镜像

根目录 `compose.yaml` 默认拉取：

`ghcr.io/morningstar-ski/chinesesubfinder-optimization:latest`

执行：

```bash
docker compose pull
docker compose up -d
```

默认端口：

- `19035` WebUI
- `19037` 视频列表图片读取

默认挂载：

- `./config:/config`
- `./media:/media`
- `./browser:/root/.cache/rod/browser`

### 从当前源码本地构建

根目录 `compose.source.yaml` 用于本地源码构建：

```bash
docker compose -f compose.source.yaml up -d --build
```

可选构建参数示例：

```bash
APP_VERSION=dev GOPROXY=https://goproxy.cn,direct docker compose -f compose.source.yaml up -d --build
```

`compose.source.yaml` 支持把现有影视库直接挂进容器：

- `${CSF_MOVIES_SOURCE:-./media/movies}:/media/movies`
- `${CSF_SERIES_SOURCE:-./media/series}:/media/series`

## 使用前准备

建议先保证媒体库目录结构规范，尤其是连续剧目录。相关说明可参考上游文档：

- [电影目录结构示例](https://github.com/ChineseSubFinder/ChineseSubFinder/blob/docs/DesignFile/%E7%94%B5%E5%BD%B1%E5%92%8C%E8%BF%9E%E7%BB%AD%E5%89%A7%E7%9B%AE%E5%BD%95%E7%BB%93%E6%9E%84%E7%A4%BA%E4%BE%8B.md)
- [连续剧目录要求](https://github.com/ChineseSubFinder/ChineseSubFinder/blob/docs/DesignFile/%E8%BF%9E%E7%BB%AD%E5%89%A7%E7%9B%AE%E5%BD%95%E7%BB%93%E6%9E%84%E8%A6%81%E6%B1%82.md)

完成部署后访问：

`http://<host>:19035`

## 相关文档

- Docker 说明：[docker/readme.md](docker/readme.md)
- 当前仓库 Issues：[Issues](https://github.com/morningstar-ski/ChineseSubFinder-optimization/issues)
- 上游使用文档：[ChineseSubFinder docs](https://github.com/ChineseSubFinder/ChineseSubFinder/tree/docs/DesignFile)

## 开发验证

```bash
go test ./...
cd frontend
npm run build
```

Windows 本地运行 `go test ./...` 前，请确保：

- `CGO_ENABLED=1`
- `PATH` 中可用 `gcc`（例如 MinGW）

## 说明

当前仓库不是上游官方发布版本，也不应描述为“上游官方更新”。
如果你需要长期稳定使用，请优先关注上游项目及其官方维护版本。

## License

[Apache License 2.0](LICENSE)
