# Claw Ops Workspace Convention

本文定义 Claw Ops 多 agent 协作时的共享目录规范。它不是一个 skill，而是所有相关 agent 都应遵守的工作区契约。

## 设计原则

目录规范应放在三层里共同约束：

```text
repo docs                 长期稳定的目录契约，给人和 agent 查阅
shared/base instructions  所有 agent 都必须遵守的共同规则
role agent instructions   每个角色只写自己的输入、输出和交接责任
```

不要只依赖某个工具型 skill 来约束目录。skill 适合封装能力和命令，例如生图、生视频、发布平台 API；目录规范属于跨角色工作协议，应放进共同 agent 定义或 workspace onboarding 文档里。角色 agent 的定义只补充本角色职责，避免每个 agent 复制整套目录规则导致漂移。

## 根目录

所有业务自动化任务统一放在工作区的 `runs/` 目录。每个任务一个稳定 `job_id`，建议格式：

```text
YYYYMMDD-topic-slug
```

示例：

```text
runs/20260422-physics-contest-campaign/
```

## 标准目录结构

```text
runs/<job_id>/
  README.md
  brief.md
  state.json

  research/
    sources.json
    notes.md
    fulltext/
    assets/

  creation/
    outline.md
    copy.md
    images/
    videos/
    requests/

  review/
    checklist.md
    comments.md

  publish/
    plan.md
    platform_payloads/
    published.json

  handoff/
    research-to-creator.md
    creator-to-publisher.md
    publisher-report.md
```

## 顶层文件

`README.md` 用于说明任务目标、当前负责人、关键链接和最终交付位置。

`brief.md` 保存用户需求、受众、主题、限制条件、发布时间、目标平台和验收标准。任何 agent 在开始执行前都应先读这个文件。

`state.json` 保存机器可读状态，建议字段：

```json
{
  "job_id": "20260422-physics-contest-campaign",
  "status": "researching",
  "owner": "claw-ops-coordinator",
  "current_stage": "research",
  "updated_at": "2026-04-22T20:00:00+08:00"
}
```

## Research 目录

`research/sources.json` 保存结构化来源清单。每条记录至少包含：

```json
{
  "id": "src-001",
  "title": "页面标题",
  "url": "https://example.com",
  "source_type": "web",
  "retrieved_at": "2026-04-22T20:00:00+08:00",
  "summary": "一句话摘要",
  "usefulness": "可用于课程卖点或事实依据"
}
```

`research/notes.md` 保存研究摘要、关键事实、可引用数据、风险点和待确认事项。

`research/fulltext/` 保存长文本材料，命名为 `src-001.md`、`src-002.md`。不要把长文塞进 `notes.md`。

`research/assets/` 保存研究阶段下载或整理的参考素材，例如截图、公开图片、PDF、音频、视频链接说明等。

## Creation 目录

`creation/outline.md` 保存内容结构、镜头脚本、图文大纲或发布系列大纲。

`creation/copy.md` 保存最终文案，包括标题、正文、字幕、口播稿、封面文案和平台变体。

`creation/requests/` 保存发给生图、生视频、LLM 或其他生成服务的请求体，例如：

```text
creation/requests/zlhub-image-poster.json
creation/requests/zlhub-video-main.json
```

`creation/images/` 保存生成图、封面、首帧、缩略图和中间图像结果。

`creation/videos/` 保存生成视频、视频任务响应、最终视频链接摘要和本地下载文件。若使用 ZLHub，保留原始 `request.json`、`create_response.json`、`query_response.json` 和 `task.json`。

## Publish 目录

`publish/plan.md` 保存平台、账号、发布时间、素材选择、标题和发布策略。

`publish/platform_payloads/` 保存每个平台的待发布 payload，例如：

```text
publish/platform_payloads/douyin.json
publish/platform_payloads/xiaohongshu.md
publish/platform_payloads/bilibili.json
```

`publish/published.json` 保存发布结果，包括平台、URL、发布时间、失败原因和重试状态。

## Handoff 目录

每个角色交接都写入 `handoff/`，不要只依赖聊天上下文。

`research-to-creator.md` 应包含研究结论、可用素材、重点事实、风险和建议创作角度。

`creator-to-publisher.md` 应包含最终素材路径、标题/正文建议、平台适配说明、不可修改项和待确认项。

`publisher-report.md` 应包含发布结果、链接、失败项、后续观察指标和复盘建议。

## 角色责任

Coordinator 负责创建 `runs/<job_id>/`、维护 `brief.md` 和 `state.json`，并检查交接文件是否完整。

Research 只能写入 `research/` 和 `handoff/research-to-creator.md`，除非 Coordinator 明确要求修改 brief。

Creator 主要读取 `brief.md`、`research/` 和 `handoff/research-to-creator.md`，写入 `creation/` 和 `handoff/creator-to-publisher.md`。

Publisher 主要读取 `brief.md`、`creation/` 和 `handoff/creator-to-publisher.md`，写入 `publish/` 和 `handoff/publisher-report.md`。

## 敏感信息

不要把 API Key、Cookie、登录态、临时签名 URL 或平台账号凭证写进仓库。生成服务返回的临时视频链接可以进入本地 `task.json`，但如果准备提交仓库，必须先清理。

`runs/` 默认应被 `.gitignore` 忽略。需要保留交付物时，只提交脱敏后的摘要、脚本和模板，不提交临时下载文件。
