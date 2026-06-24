## 鸣谢

当前版本基于 [ChineseSubFinder](https://github.com/ChineseSubFinder/ChineseSubFinder) 持续增强而来，保留对原作者和所有贡献者的感谢。

## 当前最终版相对原版的增强

- 扩展字幕源：新增 `OpenSubtitles`、`TVsubtitles`、`Moviesubtitles`、`SubHD`、`SubtitleCat`
- 英文字幕回退链默认保留 `SubtitleCat`
- 中文字幕链增加翻译回退能力，可接 `SubtitleCat` 远端翻译和 LLM 翻译链
- `SubHD` 下载链补上本地 `ddddocr` 和 `SVG` 直读
- 默认时间轴校正引擎切到 `ffsubsync`
- Docker 镜像内置 `Subflow`、`ddddocr`、`ffsubsync` 和翻译运行时
- 配置中心、帮助入口和部署文档统一收口到当前仓库

## 常用入口

- 仓库主页：[ChineseSubFinder Optimization](https://github.com/morningstar-ski/ChineseSubFinder-optimization)
- Docker 文档：[docker/readme.md](https://github.com/morningstar-ski/ChineseSubFinder-optimization/blob/main/docker/readme.md)
- 问题反馈：[Issues](https://github.com/morningstar-ski/ChineseSubFinder-optimization/issues)
