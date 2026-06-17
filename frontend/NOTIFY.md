## 鸣谢

当前仓库基于 [ChineseSubFinder](https://github.com/ChineseSubFinder/ChineseSubFinder) 持续修改而来。
保留对原作者和所有贡献者的感谢，现有能力建立在上游长期积累的扫描、匹配和字幕处理基础上。

## 相比原仓库的当前增强

- 扩展并接入更多字幕源：`OpenSubtitles`、`TVsubtitles`、`Moviesubtitles`、`SubHD`、`SubtitleCat`。
- 英文字幕回退链默认保留 `SubtitleCat`，在缺少直链字幕时补足英文源。
- 增加中文字幕翻译回退能力，可按配置使用 `SubtitleCat` 远端翻译或 LLM 翻译链。
- `SubHD` 下载链补了本地 `ddddocr` 与 SVG 直读能力，并保留外部 OCR 显式开关。
- WebUI、Docker 文档和问题反馈入口已统一对齐到当前仓库。

## 常用入口

- 帮助文档：[仓库主页](https://github.com/morningstar-ski/ChineseSubFinder-optimization)
- Docker 部署文档：[docker/readme.md](https://github.com/morningstar-ski/ChineseSubFinder-optimization/blob/main/docker/readme.md)
- 问题反馈：[Issues](https://github.com/morningstar-ski/ChineseSubFinder-optimization/issues)
