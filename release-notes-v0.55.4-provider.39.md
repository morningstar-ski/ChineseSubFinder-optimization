# v0.55.4-provider.39

本版重点是把近期反复暴露的交付问题再收一遍，尤其补齐本地验收、时间轴链路和 LLM 英文字幕回退的稳定性。

## 修复的关键问题

- 修复了时间轴校准链路里的回归问题，补齐了 `ffsubsync` 参数和失败路径测试，避免出现校准流程跑完却没有有效产物的情况。
- 修复了手动触发时间轴校准的失败结果记录问题，保证任务失败时能正确落库和返回状态。
- 修复了 Docker 发布镜像与本地代码快照之间的遗漏风险，这一版经过了本地前端构建、后端定向测试、Docker 构建和容器启动冒烟验证。
- 修复了 LLM OpenAI-compatible 回退链在端到端测试中的 prompt 兼容回归，恢复 downloader 侧完整链路通过。

## LLM 翻译链提升

- 主 prompt 做了收敛，明确限制跨 cue 串义、提前吃掉下一句、借邻近 cue 漂移语义。
- 翻译分块从纯 chunk 改成了 `target + context_only` 模式：模型可以参考邻近上下文，但只允许输出目标 cue。
- repair 阶段增加了邻近 cue 上下文，只修碎片、英文残留、混合中英名称和轻度排版问题，不做整段重写。
- 增加了轻量确定性后处理，能清理诸如 ` / 。` 这类明显排版伪影，并把可疑 ASCII 残留继续送回 repair 判定。

## 本地发布前验收

本版发布前已完成以下本地验证：

- `npm run lint`
- `npm run build`
- `go test ./pkg/logic/sub_timeline_fixer ./pkg/manual_upload_sub_2_local ./pkg/downloader`
- `python -m unittest subflow.test_translate_job subflow.test_openai_compatible_client`
- `docker build -f Dockerfile.release -t csf-provider-pack:preflight .`
- 本地容器启动冒烟，`http://127.0.0.1:19155/` 返回 `200`
- `scripts/local_delivery_audit.ps1` 全绿通过

这一版的目标不是继续堆更多 fallback，而是把现有下载、翻译、时间轴和 Docker 交付链路收紧到更稳定、可复现、可发布的状态。
