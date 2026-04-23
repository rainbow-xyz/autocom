# Paperclip Workspace Convention

这是 autocom 给单个 Paperclip / OpenClaw agent 使用的最小落盘约定。它不是编排流程，不定义角色，也不要求 agent 拆分任务。

## 原则

- 默认一个 agent 自己完成任务；平台如果要拆分子 agent，由平台自己决定。
- autocom 只关心：产物放哪里、断了怎么继续、密钥不要泄露。
- 不为了目录规范创建空目录。

## 命令

- 优先用工具的 `workdir`，不要写 `cd xxx && command`。
- 命令尽量单步执行；create、query、download、ffmpeg 分开跑。
- 不要把真实密钥写进命令行。密钥只从宿主机环境变量读取。
- 如果 preflight 拒绝链式命令，拆成更小的单步命令或改用绝对路径。

## 目录

正式任务使用一个目录：

```text
runs/YYYY-MM-DD/<job_id>/
```

最少只需要：

```text
runs/YYYY-MM-DD/<job_id>/
  brief.md
  state.json
```

简单测试可以继续用 `outputs/<name>/`，不用创建 `runs/`。

建议把产物放在：

```text
runs/YYYY-MM-DD/<job_id>/outputs/
```

需要分类时再创建：

```text
outputs/images/
outputs/videos/
outputs/final/
```

## 状态

`state.json` 只记录必要状态，不追求完整 schema。最小示例：

```json
{
  "status": "running",
  "task_id": "cgt-xxx",
  "output": "outputs/videos/main.mp4",
  "error": null,
  "updated_at": "2026-04-22T20:00:00+08:00"
}
```

规则：

- 已经拿到外部任务 `task_id` 后，继续 query，不要重复 create。
- 同一个付费 create/generate 失败最多重试 2 次。
- 失败 2 次后停止，记录错误，交给用户或平台决定。

## 素材

传给外部 API 的图片、视频、音频必须是公网可访问 URL。

不能传：

- 本地路径
- localhost / 127.0.0.1
- 内网地址
- 需要登录态的 URL
- base64

## 安全

不要提交：

- API Key、Cookie、登录态、平台账号凭证
- 临时签名 URL
- 生成媒体文件、下载文件
- 含真实密钥或签名 URL 的 `runs/`、`outputs/`、`task.json`

`runs/` 和 `outputs/` 默认应被 `.gitignore` 忽略。
