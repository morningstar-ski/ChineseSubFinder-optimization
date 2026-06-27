# ChineseSubFinder Optimization

基于上游 [ChineseSubFinder](https://github.com/ChineseSubFinder/ChineseSubFinder) 持续整理和增强的交付版，目标是把实际可用的字幕下载、回退、翻译、时间轴校正和 Docker 部署链路收口成一版更适合直接落地使用的版本。

## 鸣谢原作者

当前仓库建立在上游 `ChineseSubFinder` 的长期工作基础上。媒体库扫描、字幕匹配、字幕整理、时间轴处理和 WebUI 能力都继承自原项目的长期积累。

- 上游仓库：[ChineseSubFinder](https://github.com/ChineseSubFinder/ChineseSubFinder)
- 当前仓库：[ChineseSubFinder Optimization](https://github.com/morningstar-ski/ChineseSubFinder-optimization)

## ⚠️ 免责声明 (Disclaimer)

**在使用本项目（ChineseSubFinder Optimization）之前，请您务必仔细阅读并透彻理解本声明。当您拉取代码、部署或使用本项目，即视为您已阅读、理解并完全同意接受以下全部条款：**

1. **项目定位与非商业用途**：
   本项目建立在对上游优秀开源项目的学习与优化之上，仅供个人技术研究、学习交流以及家庭媒体库的自动化管理使用。**严禁将本项目用于任何形式的商业用途、非法盈利或构建公共服务。**

2. **版权与内容声明**：
   本项目作为纯技术工具，**本身不提供、不发布、不存储**任何形式的影视资源及字幕文件。程序运行过程中获取的所有字幕及元数据，均来源于互联网公开的第三方网站、接口或社区贡献。用户应自行承担使用本项目下载内容的版权风险。请遵守所在国家/地区的版权法律法规，尊重版权方权益，建议仅为已获得合法授权的媒体内容匹配字幕。

3. **第三方服务合规**：
   本项目中集成的字幕源站点检索、翻译 API 等功能均依赖于第三方服务提供商。这些服务的所有权及最终解释权归原厂商所有。请使用者合理设置请求频率，务必遵守各第三方平台的《用户服务协议》及 API 使用规范。因用户滥用工具（如高频并发请求）导致的账号封禁、IP 屏蔽或引发的任何法律纠纷，均由使用者本人自行承担，与本项目及开发者无关。

4. **无担保与责任限制（AS-IS）**：
   本项目代码（包括上游继承代码及修改后的代码）按“原样”（AS-IS）提供，没有任何形式的明示或暗示保证（包括但不限于对特定用途的适用性、程序稳定性、数据无损性等）。由于安装、配置、修改或使用本软件而造成的任何直接或间接损失（包括但不限于数据丢失、系统损坏、业务中断或法律纠纷），**当前项目的开发者、贡献者以及上游项目（ChineseSubFinder）的原作者均不承担任何法律及连带责任。**

5. **权益保护**：
   如果您是某个第三方字幕站点或服务接口的所有权人，且认为本项目的检索规则侵犯了您的合法权益或给您的服务器带来了困扰，请通过提交 Issue 告知，我们会在核实后第一时间进行评估并移除相关功能代码。

> 💡 **总结**：请合理、合法、合规地使用本工具，技术分享不易，请共同维护良好的开源环境。

## NEW FEATURES

相比较原版，当前最终版重点补的是“可交付”和“全链路回退”这两件事，不是只加几个零散 provider。

### 1. 更完整的字幕源与回退链

- 新增并接入更多字幕源：`OpenSubtitles`、`TVsubtitles`、`Moviesubtitles`、`SubHD`、`SubtitleCat`
- 英文字幕回退链默认保留 `SubtitleCat`，在主源拿不到英文字幕时自动补位
- 中文字幕下载链支持更清晰的分层回退，避免单一源波动直接把整条链打死

### 2. 中文字幕翻译回退能力

- 支持 `SubtitleCat` 远端翻译能力接入中文链路
- 支持英文字幕下载后再走 LLM 翻译回退链
- LLM 翻译链已经适配 Docker 一键部署场景，不再要求用户自己额外挂 Python 路径和 Subflow 路径

### 3. SubHD 下载链强化

- `SubHD` 验证码下载链新增本地 `ddddocr`
- 支持 `SVG` 直读
- 外部 OCR 改成显式开关，不再作为默认兜底，避免无感把请求全部打到外部服务

### 4. 时间轴校正链路增强

- 默认时间轴修正引擎切到 `ffsubsync`
- 自动下载字幕后的时间轴校正链已接入容器运行时
- 发布镜像已内置 `ffsubsync`，不需要再额外手装

### 5. Docker 交付链收口

- 发布镜像单独维护，避免“源码能跑、发布镜像缺运行时”的情况
- 容器镜像已内置：
  - `Subflow`
  - `ddddocr`
  - `ffsubsync`
  - LLM 翻译所需 Python 运行时
- 适合直接一键部署，不需要用户再补运行依赖

### 6. 前端与配置体验重整

- 配置中心按能力分组重新整理，减少原先杂乱堆叠
- 回退、翻译、浏览器和实验项的入口重新归类
- 项目说明、鸣谢、帮助入口、Docker 文档入口统一对齐到当前仓库

### 7. 发布流程与验收流程补强

- 补了本地交付审计入口
- 发布镜像改为独立 GitHub Actions 工作流构建
- 发布前会串联前端构建、后端测试和镜像构建校验，降低“发出去才发现缺东西”的概率

## Docker 部署

普通用户只需要认一个部署入口：

- `docker-compose.yaml`

`compose.yaml` 仅作为兼容别名保留；`compose.source.yaml`、`compose.browser.yaml`、`compose.fnos.yaml` 都是开发或验证辅助文件，不是普通用户部署入口。

### 标准部署步骤

1. 复制 `.env.example` 为 `.env`
2. 按宿主机情况填写 `.env` 里的路径和端口
3. 先渲染检查标准部署文件
4. 启动标准部署文件

默认镜像：

`ghcr.io/morningstar-ski/chinesesubfinder-optimization:latest`

执行：

```bash
cp .env.example .env
docker compose -f docker-compose.yaml config
docker compose -f docker-compose.yaml pull
docker compose -f docker-compose.yaml up -d
```

健康检查：

```bash
curl http://127.0.0.1:19035/system-status
```

### `.env` 关键项

- `CSF_CONFIG_DIR`：宿主机配置目录，映射到容器 `/config`
- `CSF_MEDIA_DIR`：宿主机影视库根目录，映射到容器 `/media`
- `CSF_BROWSER_DIR`：宿主机浏览器缓存目录
- `CSF_CONTAINER_NAME` / `CSF_HOSTNAME`：容器名与主机名，默认都是 `chinesesubfinder`
- `CSF_WEB_PORT`：WebUI 端口，默认 `19035`
- `CSF_STATIC_PORT`：静态文件端口，默认 `19037`
- `PUID` / `PGID` / `PERMS` / `TZ` / `UMASK`：容器运行参数

Windows 默认可直接使用：

```dotenv
CSF_CONFIG_DIR=./config
CSF_MEDIA_DIR=./media
CSF_BROWSER_DIR=./browser
CSF_CONTAINER_NAME=chinesesubfinder
CSF_HOSTNAME=chinesesubfinder
```

FnOS / Linux 可改成：

```dotenv
CSF_CONFIG_DIR=./config
CSF_MEDIA_DIR=/vol2/1000/video/link
CSF_BROWSER_DIR=./browser
CSF_CONTAINER_NAME=chinesesubfinder
CSF_HOSTNAME=chinesesubfinder
PUID=999
PGID=901
PERMS=false
```

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

- [Docker 说明](./docker/readme.md)
- [当前仓库 Issues](https://github.com/morningstar-ski/ChineseSubFinder-optimization/issues)
- [上游使用文档](https://github.com/ChineseSubFinder/ChineseSubFinder/tree/docs/DesignFile)

## 开发验证

```bash
go test ./...
cd frontend
npm run build
```

Windows 本地运行 `go test ./...` 前，请确保：

- `CGO_ENABLED=1`
- `PATH` 中可用 `gcc`（例如 MinGW）

## 本地自动化入口

- 交付前统一本地审计入口：
  `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/local_delivery_audit.ps1`
- 该入口会串联 `frontend build`、目标 `go test`、后端 / 前端探活与残留审计
- 下列脚本仅用于专项回归，不作为默认交付入口：
  - `scripts/local_full_acceptance.ps1`
  - `scripts/local_llm_acceptance.ps1`
  - `scripts/local_expanded_acceptance.ps1`

## 说明

当前仓库不是上游官方发布版，而是基于上游持续增强后的独立交付版本。若你需要回溯原始设计或基础能力，请以原作者仓库为准。

## License

[Apache License 2.0](LICENSE)
