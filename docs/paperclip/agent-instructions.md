# Paperclip Instructions

可粘贴到 Paperclip / OpenClaw instructions：

```text
你在 autocom 自动化工作区内工作。开始正式 job 前阅读：

- docs/paperclip/workspace-convention.md

默认由你这个 agent 自己完成任务；如果平台拆分子 agent，由平台自行处理，不要在 autocom 里定义额外编排流程。

正式 job 放在 runs/YYYY-MM-DD/<job_id>/。最少维护 brief.md 和 state.json。简单连通性测试可以用 outputs/<name>/。

中断恢复时先读 state.json。外部异步任务如果已经有 task_id，只能继续查询，不要重复创建。

同一个付费 create/generate step 失败最多重试 2 次；失败 2 次后停止，记录错误，交给用户或平台决策。

传给外部 API 的素材必须是公网可访问 URL，不能是本地路径、localhost、内网地址、登录态地址或 base64。

执行命令保持单步清晰。不要依赖 cd、export、&&、;、管道或命令替换。如果 preflight 拒绝链式命令，改用 workdir、绝对路径，或拆成多次单命令执行。

不要把 API Key、Cookie、登录态、临时签名 URL 或生成媒体文件提交到仓库。真实密钥只从宿主机环境变量读取。
```
