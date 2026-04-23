# Paperclip Instructions

这份内容用于粘贴到 Paperclip / OpenClaw instructions。它只说明 autocom 的落盘和恢复约定，不定义角色、交接或编排流程。

## 公共指令

```text
你在 autocom 自动化工作区内工作。开始 job 前阅读：

- docs/paperclip/workspace-convention.md

Paperclip/OpenClaw 的协作、交接、运行状态、资源管理能力优先。autocom 目录只负责业务输入、恢复状态、最终产物路径和敏感信息隔离。

正式 job 放在：

runs/YYYY-MM-DD/<job_id>/

每个 job 最少维护：

- brief.md
- state.json

其它目录按需创建，不要为了目录规范预先铺满空目录。平台已有 handoff、run state、resource 管理时，优先用平台能力。

中断恢复时先读 state.json。外部异步任务如果已经有 task_id，只能继续查询，不要重复创建。

同一个付费 create/generate step 失败最多重试 2 次；失败 2 次后停止，记录错误，交给用户或平台决策。

传给外部 API 的资源必须是公网可访问 URL，不能是本地路径、localhost、内网地址、登录态地址或 base64。

执行命令时保持单步清晰。不要依赖 cd、export、&&、;、管道或命令替换。如果 preflight 拒绝链式命令，改用 workdir、绝对路径，或拆成多次单命令执行。

不要把 API Key、Cookie、登录态、临时签名 URL 或生成媒体文件提交到仓库。真实密钥只从宿主机环境变量读取。
```

## 优先级

```text
用户指令 > job brief > Paperclip/OpenClaw 平台流程 > docs/paperclip/workspace-convention.md > skill instructions
```
