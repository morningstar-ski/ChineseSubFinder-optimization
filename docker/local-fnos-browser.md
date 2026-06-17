# Local FnOS Browser Regression

用于本地源码构建 Docker，同时挂载飞牛桥接卷，验证 `subhd + SVG + ddddocr + browser` 整条字幕下载链。

## Compose Files

- `compose.source.yaml`: 本地源码构建
- `compose.browser.yaml`: 开启 browser 运行时
- `compose.fnos.yaml`: 挂载飞牛桥接卷，并默认 `PERMS=false`

## Start

```bash
docker compose -f compose.source.yaml -f compose.browser.yaml -f compose.fnos.yaml up -d --build
```

## Optional Volume Overrides

默认外部卷名：

- `fnos_movies_bind`
- `fnos_series_bind`

如需覆盖：

```bash
CSF_MOVIES_VOLUME=your_movies_volume
CSF_SERIES_VOLUME=your_series_volume
docker compose -f compose.source.yaml -f compose.browser.yaml -f compose.fnos.yaml up -d --build
```

## Notes

- `PERMS=false` 用来避免对挂载影视库做递归权限重置。
- 测试完成后，及时删除新下载字幕和 `config/Logs/Once-*.log`。
- 默认本地 browser 镜像名为 `chinesesubfinder:local-browser`。
