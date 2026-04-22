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

日期目录使用完整日期 `YYYY-MM-DD`，不要只用月份或不完整日期。这样便于按天归档，也能避免跨日期任务混在同一层。

示例：

```text
runs/2026-04-22/20260422-physics-contest-campaign/
```

## 标准目录结构

```text
runs/YYYY-MM-DD/<job_id>/
  README.md
  brief.md
  state.json

  assets/
    asset_manifest.json
    images/
    videos/
    audio/
    documents/

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
  "run_date": "2026-04-22",
  "run_dir": "runs/2026-04-22/20260422-physics-contest-campaign",
  "status": "researching",
  "owner": "claw-ops-coordinator",
  "current_stage": "research",
  "updated_at": "2026-04-22T20:00:00+08:00"
}
```

`state.json` 只记录整个 job 的阶段状态。某个子流程的细节状态放在子目录，例如 `creation/videos/workflow_state.json`。下游 agent 恢复工作时先读顶层 `state.json`，再读自己负责目录里的状态文件。

## 状态与重试

涉及外部 API、付费生成、发布平台写入或长耗时任务时，子流程状态文件必须记录每个 step 的重试信息：

```json
{
  "id": "main_video",
  "status": "failed",
  "action": "zlhub.video-create",
  "retry_count": 2,
  "max_retries": 2,
  "task_id": "",
  "error": "InvalidParameter: duration is not valid",
  "updated_at": "2026-04-22T22:00:00+08:00"
}
```

重试规则：

```text
同一个付费 create/generate step 最多失败重试 2 次
如果已经拿到 task_id，只能继续 query，不能重复 create
连续 2 次失败后，必须把 step 标记为 failed，并停止自动尝试
停止后由用户或 Coordinator 决定是否换参数、换模型、降级需求或终止 job
每次失败都要记录 request_path、response_path、error 和 updated_at
```

这个限制是成本保护，不是技术限制。Agent 不应为了“自己完成任务”而无限尝试更多时长、模型或素材组合。

## 跨日期查找

如果需要跨日期、跨 job 查找资源，Coordinator 维护 `runs/index.jsonl`。每个 job 一行，记录可搜索摘要和关键产物位置：

```json
{"job_id":"20260422-physics-contest-campaign","run_date":"2026-04-22","run_dir":"runs/2026-04-22/20260422-physics-contest-campaign","topic":"物理竞赛课程营销","status":"published","assets":["creation/images/poster.jpeg","creation/videos/final/complete.mp4"],"updated_at":"2026-04-22T22:00:00+08:00"}
```

Agent 查找历史素材时，先读 `runs/index.jsonl`，再进入匹配的 `run_dir` 查看 `state.json`、`creation/` 和 `publish/`。不要为了找资源全量扫描每个 job 的长文本正文。

## 资源规范

跨 agent 可复用资源统一登记到 job 级资源池：

```text
runs/YYYY-MM-DD/<job_id>/assets/
  asset_manifest.json
  images/
  videos/
  audio/
  documents/
```

`asset_manifest.json` 是资源索引。任何 agent 产出可复用图片、视频、音频、PDF、长文、截图或外部 URL 时，都应追加一条记录：

```json
[
  {
    "asset_id": "img-001",
    "type": "image",
    "status": "approved",
    "local_path": "assets/images/first-frame.png",
    "public_url": "https://example.com/first-frame.png",
    "usable_for_api": true,
    "provider": "zlhub",
    "source_agent": "creator",
    "source": "generated",
    "allowed_uses": ["first_frame", "reference_image", "cover"],
    "rights": "owned_or_generated",
    "credit_required": false,
    "expires_at": "2026-04-23T20:00:00+08:00",
    "derived_from": [],
    "notes": "视频首帧候选图"
  }
]
```

资源状态只能使用：

```text
candidate    候选素材，未确认可用
approved     已确认可用于创作
generated    生成产物，待筛选或待处理
final        最终交付资源
rejected     禁用，不再使用
expired      URL 过期，需要重新上传或刷新
```

传给外部 API 的资源必须满足：

```text
public_url 非空
usable_for_api=true
expires_at 为空或尚未过期
URL 是公网可访问地址
不能是 localhost、127.0.0.1、内网地址、需要登录态的地址或 base64
临时签名 URL 只能保存在本地 runs/，提交仓库前必须清理
```

跨 agent 使用资源时必须通过 `asset_id` 引用，不要只写模糊文件名。下游 agent 使用资源前必须检查 `status`、`allowed_uses`、`rights`、`public_url`、`usable_for_api` 和 `expires_at`。如果对资源做了裁剪、转码、重新生成或二次创作，应生成新的 `asset_id`，并在 `derived_from` 里记录来源。

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

研究阶段资源如果会交给 Creator 使用，必须同步登记到 job 级 `assets/asset_manifest.json`，并在 `handoff/research-to-creator.md` 里引用对应 `asset_id`。

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

创作阶段最终可交付资源必须同步登记到 `assets/asset_manifest.json`，并把最终文件放到明确的 `final/` 或 `publish/` 位置，避免 Publisher 从中间文件里猜测应该发布哪个版本。

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

Coordinator 负责创建 `runs/YYYY-MM-DD/<job_id>/`、维护 `brief.md`、`state.json` 和 `runs/index.jsonl`，并检查交接文件与 `assets/asset_manifest.json` 是否完整。

Research 主要写入 `research/` 和 `handoff/research-to-creator.md`。如果产出可复用资源，可以写入 `assets/` 并追加或更新自己产出的 `asset_manifest.json` 记录。除非 Coordinator 明确要求，不要修改 brief。

Creator 主要读取 `brief.md`、`research/`、`assets/asset_manifest.json` 和 `handoff/research-to-creator.md`，写入 `creation/`、`handoff/creator-to-publisher.md`，并登记自己产出的可复用资源。

Publisher 主要读取 `brief.md`、`creation/`、`assets/asset_manifest.json` 和 `handoff/creator-to-publisher.md`，写入 `publish/` 和 `handoff/publisher-report.md`。发布成功后可以把已发布最终资源的状态更新为 `final`。

## 目录优先级

目录规范以本文为准。各 skill 只能在本文定义的目录下补充本能力的子目录和文件命名，不能另起一套正式 job 目录。Skill 文档里的 `outputs/` 示例只用于单独 CLI 连通性测试，不用于 Paperclip/OpenClaw 正式 job。

## 敏感信息

不要把 API Key、Cookie、登录态、临时签名 URL 或平台账号凭证写进仓库。生成服务返回的临时视频链接可以进入本地 `task.json`，但如果准备提交仓库，必须先清理。

`runs/` 默认应被 `.gitignore` 忽略。需要保留交付物时，只提交脱敏后的摘要、脚本和模板，不提交临时下载文件。
