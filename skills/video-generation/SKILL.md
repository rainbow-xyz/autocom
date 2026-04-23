---
name: video-generation
description: Create and query asynchronous video generation tasks for content production, marketing videos, social-media clips, or multimodal image/video/audio-to-video workflows. Current provider is ZLHub Seedance via the bundled Python CLI.
---

# Video Generation

用于创建和查询异步视频生成任务，支持文生视频、首尾帧、参考图片、参考视频、参考音频。当前 provider 是 ZLHub Seedance。

## 准备

宿主机必须预先提供环境变量 `ZLHUB_API_KEY`。在普通 shell 中可以用 `export ZLHUB_API_KEY="your-zlhub-api-key"` 设置，但 agent 执行任务时不要把真实 key 写进命令行，也不要使用 `export ... && command` 链式命令。

如果运行时发现 `ZLHUB_API_KEY` 缺失，停止并提示用户在宿主机环境设置。不要把 key 写入配置、请求、日志或最终回答。

相关文件：

```text
providers/zlhub/scripts/zlhub_cli.py
providers/zlhub/config/autocom.yaml
providers/zlhub/examples/video_request.json
```

执行命令时：

- 从已安装的 `video-generation` skill 目录运行，或使用绝对路径。
- 能设置 `workdir` 时，用 `workdir`，不要写 `cd ... && command`。
- 如果 preflight 拒绝链式命令，拆成单步命令执行。
- 下载、查询、拼接视频分开执行，不要把 `curl`、变量解析和 `ffmpeg` 串成一条命令。

## 创建任务

低成本测试默认用 fast + 480p：

```bash
python3 providers/zlhub/scripts/zlhub_cli.py video-create \
  --config providers/zlhub/config/autocom.yaml \
  --prompt "生成一段 4 秒 9:16 竖屏物理竞赛课程营销视频，黑板公式、竞赛课堂、学生思考，真实教育广告质感。" \
  --model doubao-seedance-2.0-fast \
  --ratio 9:16 \
  --resolution 480p \
  --duration 4 \
  --generate-audio=false \
  --watermark=false \
  --out-dir outputs/physics-video-001
```

复杂请求用 JSON：

```bash
python3 providers/zlhub/scripts/zlhub_cli.py video-create \
  --config providers/zlhub/config/autocom.yaml \
  --file providers/zlhub/examples/video_request.json \
  --out-dir outputs/video-json-001
```

命令行素材参数可重复：

```text
--image role=url
--video role=url
--audio role=url
```

示例：

```bash
python3 providers/zlhub/scripts/zlhub_cli.py video-create \
  --config providers/zlhub/config/autocom.yaml \
  --prompt "以图片1作为首帧，参考视频1的镜头运动，生成 4 秒竖屏视频。" \
  --image first_frame=https://example.com/first.jpg \
  --video reference_video=https://example.com/ref.mp4 \
  --ratio 9:16 \
  --resolution 480p \
  --duration 4 \
  --out-dir outputs/video-assets-001
```

请求会调用：

```text
POST /v1/task/create
```

返回的 `id` 是视频任务 ID。

## 查询任务

```bash
python3 providers/zlhub/scripts/zlhub_cli.py video-get \
  --config providers/zlhub/config/autocom.yaml \
  --id cgt-xxx \
  --out-dir outputs/physics-video-001
```

请求会调用：

```text
GET /v1/task/get/{id}
```

输出位置：

```text
<out-dir>/zlhub/
  request.json
  create_response.json
  query_response.json
  task.json
```

`task.json` 会保存 `task_id`、`status`、`video_url`、`error`、`updated_at`。`status=succeeded` 后及时下载 `video_url`，因为通常是临时签名 URL。

不要频繁轮询。真实任务无 callback 时，建议创建约 10 分钟后查询；同一任务最多每分钟查一次。

## 素材和提示词

传给 ZLHub 的图片、视频、音频必须是公网可访问 URL；不能是本地路径、localhost、内网地址、登录态 URL 或 base64。

提示词引用素材时用同类素材序号：

```text
图片1、图片2、视频1、音频1
```

正式 Paperclip job 中，如果素材会被其它流程复用、传给外部 API、或用于发布，再登记到：

```text
runs/YYYY-MM-DD/<job_id>/assets/asset_manifest.json
```

使用前至少检查 `status`、`rights`、`public_url`、`usable_for_api`。

## 恢复与多步任务

这里不定义固定编排流程。按 `brief.md` 和平台流程决定步骤，只要保证可恢复：

- 正式 job 使用 `runs/YYYY-MM-DD/<job_id>/creation/videos/`，不要用 `outputs/`。
- 长流程维护 `creation/videos/workflow_state.json`。
- 已经拿到 `task_id` 时继续 query，不要重复 create。
- 同一个付费 create/generate step 失败最多重试 2 次；失败 2 次后停止并记录错误。
- 单视频只需要 `main/` 和 `final/`；多段或续写时再创建 `segments/`、`extensions/`。

续写视频的核心做法：

1. 查询上一个任务，拿到 `video_url`。
2. 用 `--video reference_video=<video_url>` 创建下一段。
3. 下载所有段落。
4. 用 `ffmpeg` 拼接本地文件。

## 成本和限制

- 默认低成本测试：`doubao-seedance-2.0-fast`、`480p`、`generate_audio=false`。
- 已观察价格：`doubao-seedance-2.0-fast` 约 `¥37 / 1M tokens`，`doubao-seedance-2.0` 约 `¥46 / 1M tokens`。
- 已观察限制：`doubao-seedance-2.0-fast` 文生视频 `duration=1/2/3` 会返回 `InvalidParameter`，`duration=4` 成功。
- 正常视频时长建议使用 `4-15` 秒。
- `doubao-seedance-2.0-mock` 对当前测试账号返回过 `model_not_found`。

## 安全

不要提交 `outputs/`、`runs/`、真实 API Key、签名 URL 或生成媒体文件。
