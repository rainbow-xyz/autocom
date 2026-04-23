# Paperclip Workspace Convention

这是给 Paperclip / OpenClaw 使用的最小落盘约定。目标是让任务可恢复、产物可找到、敏感信息不进仓库；不要为了满足目录而创建无用文件。

## 0. 平台优先

如果 Paperclip / OpenClaw 已经提供协作、交接、运行状态、资源管理或发布记录能力，优先使用平台能力。

autocom 目录只负责三件事：

- 保存 job 的业务输入和当前进度。
- 保存外部 API 的可恢复状态，例如 `task_id`、输出目录、错误信息。
- 保存最终产物路径和必要的本地审计记录。

不要用本规范替代 Paperclip / OpenClaw 的编排流程。平台负责“谁来做、什么时候做、怎么分派、如何交接”；autocom 只负责“落盘到哪里、断了怎么继续、什么不能提交”。

## 1. 命令执行

Agent 执行命令时保持单步清晰：

- 能设置 `workdir` 时用 `workdir`，不要写 `cd xxx && command`。
- 不能设置 `workdir` 时用绝对路径。
- 不要依赖 `export`、`&&`、`;`、管道或命令替换。
- 一个命令只做一个主要动作，例如 create、query、download、ffmpeg 分开执行。
- 如果 preflight 拒绝命令，先拆成更小的单步命令。
- 真实密钥必须由宿主机环境提供，不要写进命令行、文件或日志。

## 2. Job 目录

每个任务一个目录：

```text
runs/YYYY-MM-DD/<job_id>/
```

示例：

```text
runs/2026-04-22/20260422-physics-contest-campaign/
```

`YYYY-MM-DD` 使用完整日期。跨天继续同一个任务时，继续使用原 job 目录。

## 3. 最小必需

每个 job 只强制维护两个文件：

```text
runs/YYYY-MM-DD/<job_id>/
  brief.md
  state.json
```

`brief.md` 写任务目标、限制、验收标准。

`state.json` 写当前进度，最小格式：

```json
{
  "job_id": "20260422-physics-contest-campaign",
  "status": "running",
  "current_step": "video",
  "updated_at": "2026-04-22T20:00:00+08:00"
}
```

其它目录按需创建，不要预先铺满。

## 4. 按需目录

需要哪个阶段才创建哪个目录：

```text
research/   资料和来源
creation/   文案、图片、视频、生成请求
publish/    平台 payload 和发布结果
assets/     平台资源管理不可用、资源要跨 agent 复用、或要发布时再创建
```

单视频、单图片、单次脚本任务可以只使用 `creation/`，不必创建 `research/`、`publish/`、`assets/`。

## 5. 恢复规则

外部异步任务、付费生成、发布动作必须能恢复。最少记录：

```json
{
  "id": "main_video",
  "status": "running",
  "action": "zlhub.video-create",
  "task_id": "cgt-xxx",
  "out_dir": "creation/videos/main",
  "retry_count": 0,
  "max_retries": 2,
  "error": null,
  "updated_at": "2026-04-22T20:00:00+08:00"
}
```

规则：

- 开始前先读 `state.json`。
- 已经有 `task_id` 时继续 query，不要重复 create。
- 同一个付费 create/generate step 失败最多重试 2 次。
- 失败 2 次后停止自动尝试，记录错误，交给用户或平台决策。

## 6. 资源规则

优先使用 Paperclip / OpenClaw 自带的资源管理能力。只有平台资源管理不可用，或者资源需要在本仓库长期落盘索引时，才创建：

```text
assets/asset_manifest.json
```

最小记录：

```json
[
  {
    "asset_id": "video-001",
    "type": "video",
    "status": "final",
    "local_path": "creation/videos/final/complete.mp4",
    "public_url": "",
    "usable_for_api": false,
    "rights": "owned_or_generated",
    "notes": "最终视频"
  }
]
```

传给外部 API 的资源必须是公网可访问 URL，不能是本地路径、localhost、内网地址、登录态 URL 或 base64。

## 7. 敏感信息

不要提交：

- API Key、Cookie、登录态、平台账号凭证。
- 临时签名 URL。
- 生成媒体文件、下载文件。
- 含真实密钥或签名 URL 的 `runs/`、`outputs/`、`task.json`。

`runs/` 和 `outputs/` 默认应被 `.gitignore` 忽略。
