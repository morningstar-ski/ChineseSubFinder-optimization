export const PROJECT_REPO_URL = 'https://github.com/morningstar-ski/ChineseSubFinder-optimization';
export const PROJECT_HELP_URL = PROJECT_REPO_URL;
export const PROJECT_DOCKER_DOC_URL = `${PROJECT_REPO_URL}/blob/main/docker/readme.md`;
export const PROJECT_ISSUES_URL = `${PROJECT_REPO_URL}/issues`;
export const PROJECT_BUG_TEMPLATE_URL = `${PROJECT_REPO_URL}/issues/new?template=----bug----.md`;

export const PROJECT_UPDATE_MARKDOWN = `
## 当前仓库

- 本界面和当前运行时以 [ChineseSubFinder-optimization](${PROJECT_REPO_URL}) 为准。
- 这里展示的是本仓库自己的接线与更新摘要，不再复用上游官方发布说明。

## 当前版本已对齐的内容

- 新增并接入 \`OpenSubtitles\`、\`TVsubtitles\`、\`Moviesubtitles\`、\`SubHD\`。
- 修复预任务初始化完成信号，避免前端长时间停在初始化状态。
- 设置页已对齐当前可用字幕源，并隐藏未启用的前端展示入口。
- 帮助文档和问题反馈统一跳转到本仓库。
`;
