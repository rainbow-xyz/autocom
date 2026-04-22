---
name: video-generation
description: Create and query asynchronous video generation tasks for content production, marketing videos, social-media clips, or multimodal image/video/audio-to-video workflows. Current provider is ZLHub Seedance via the bundled Python CLI.
---

# Video Generation

Use this skill when an agent needs to create videos, query asynchronous video jobs, use first/last frames, use reference images/videos/audio, or prepare generated video assets for downstream publishing.

Current provider:

```text
ZLHub Seedance
```

ZLHub is a provider under this business skill. Future video providers should be added under `providers/<provider-name>/`.

## Files

```text
providers/zlhub/scripts/zlhub_cli.py
providers/zlhub/examples/video_request.json
providers/zlhub/config/autocom.yaml
```

Run commands from the installed `video-generation` skill directory, or pass absolute paths.

For Paperclip/OpenClaw jobs, the workspace convention wins:

```text
runs/YYYY-MM-DD/<job_id>/creation/videos/
```

Examples using `outputs/` are standalone CLI smoke tests only. Do not use `outputs/` as the formal job directory.

## Environment

Set the API key on the host:

```bash
export ZLHUB_API_KEY="your-zlhub-api-key"
```

Never write real keys into config, examples, requests, logs, output summaries, or final answers.

## Create Video Task

Default provider/model:

```text
provider: ZLHub
model: doubao-seedance-2.0-fast
ratio: 9:16
watermark: false
```

The fast model is the default because `doubao-seedance-2.0` costs more. Use the standard model only for final output.

Command:

```bash
python3 providers/zlhub/scripts/zlhub_cli.py video-create \
  --config providers/zlhub/config/autocom.yaml \
  --prompt "生成一段 9:16 竖屏物理竞赛课程营销短视频，时长 5 秒。画面为黑板公式、竞赛课堂和学生思考，真实高级教育广告质感，底部留白用于后期添加字幕。" \
  --ratio 9:16 \
  --resolution 480p \
  --duration 5 \
  --generate-audio=true \
  --watermark=false \
  --out-dir outputs/physics-video-001
```

This calls:

```text
POST /v1/task/create
```

The returned `id` is the video task ID, not the final video file ID.

Outputs:

```text
outputs/physics-video-001/zlhub/
  request.json
  create_response.json
  task.json
```

Use standard model for final output:

```bash
python3 providers/zlhub/scripts/zlhub_cli.py video-create \
  --config providers/zlhub/config/autocom.yaml \
  --model doubao-seedance-2.0 \
  --prompt "生成最终成片版本" \
  --ratio 9:16 \
  --duration 8 \
  --out-dir outputs/video-final
```

If a reachable HTTPS callback exists:

```bash
python3 providers/zlhub/scripts/zlhub_cli.py video-create \
  --config providers/zlhub/config/autocom.yaml \
  --prompt "生成一段课程营销短视频" \
  --callback-url "https://example.com/api/zlhub/callback" \
  --out-dir outputs/video-with-callback
```

If no callback is available, omit it and poll later.

## Create Video From JSON Or Materials

For complex multimodal requests:

```bash
python3 providers/zlhub/scripts/zlhub_cli.py video-create \
  --config providers/zlhub/config/autocom.yaml \
  --file providers/zlhub/examples/video_request.json \
  --out-dir outputs/video-json-001
```

For command-line materials:

```bash
python3 providers/zlhub/scripts/zlhub_cli.py video-create \
  --config providers/zlhub/config/autocom.yaml \
  --prompt "以图片1作为首帧，参考图片2的课程海报风格，生成 5 秒竖屏视频。" \
  --image first_frame=https://example.com/first.jpg \
  --image reference_image=https://example.com/poster.jpg \
  --ratio 9:16 \
  --duration 5 \
  --out-dir outputs/video-assets-001
```

Material flags are repeatable:

```text
--image role=url
--video role=url
--audio role=url
```

Only pass public URLs that ZLHub can fetch. Do not pass local paths, localhost URLs, private bucket URLs without signatures, or base64 content.

For shared job assets, first check the job-level manifest:

```text
runs/YYYY-MM-DD/<job_id>/assets/asset_manifest.json
```

Only use assets whose `status` allows use, `allowed_uses` matches the intended role, and `public_url` is usable for API calls. If a local file has no `public_url`, upload it to object storage first; this CLI does not upload local files.

In prompts, refer to materials by type and index:

```text
图片1、图片2、视频1、音频1
```

## Query Video Task

```bash
python3 providers/zlhub/scripts/zlhub_cli.py video-get \
  --config providers/zlhub/config/autocom.yaml \
  --id cgt-20260421155200-nb62z \
  --out-dir outputs/physics-video-001
```

This calls:

```text
GET /v1/task/get/{id}
```

Read `task.json` first:

```json
{
  "task_id": "cgt-xxx",
  "status": "succeeded",
  "video_url": "https://...",
  "error": null,
  "updated_at": "2026-04-21T15:56:27+08:00"
}
```

When `status` is `succeeded`, download `video_url` promptly because ZLHub returns temporary signed URLs.

Do not poll too frequently. For real video generation, wait about 10 minutes after creation if no callback arrives, then query at most once per task per minute.

## Agent-Managed Workflow

这里有意不提供 workflow runner 脚本。多步骤任务由 agent 自己读取 brief、判断下一步、调用基础工具，并在 job 目录维护状态文件。

状态文件固定使用：

```text
runs/YYYY-MM-DD/<job_id>/creation/videos/workflow_state.json
```

状态结构：

```json
{
  "job_id": "20260422-physics-video-test",
  "status": "running",
  "current_step": "base_video",
  "steps": [
    {
      "id": "base_video",
      "status": "completed",
      "action": "zlhub.video-create",
      "out_dir": "runs/2026-04-22/20260422-physics-video-test/creation/videos/base",
      "retry_count": 0,
      "max_retries": 2,
      "task_id": "cgt-xxx",
      "video_url": "https://...",
      "updated_at": "2026-04-22T20:00:00+08:00"
    }
  ],
  "updated_at": "2026-04-22T20:00:00+08:00"
}
```

Agent 执行规则：

```text
1. Before each step, read workflow_state.json if it exists.
2. If a step is completed, reuse its outputs and continue with the next step.
3. Before starting a step, write status=running and current_step=<step_id>.
4. After finishing a step, write status=completed and record durable outputs such as task_id, video_url, local file paths, and command notes.
5. If interrupted, resume from workflow_state.json and the per-step zlhub/task.json files.
6. Do not create a duplicate ZLHub task if a previous task_id already exists and can still be queried.
7. For paid create calls, retry at most 2 failed attempts for the same step. After 2 failures, set status=failed, record the error, and stop for user or Coordinator decision.
```

视频子流程目录建议：

```text
creation/videos/
  workflow_state.json
  main/                 single video or primary task
    zlhub/
      request.json
      create_response.json
      query_response.json
      task.json
    video.mp4
  segments/             optional multi-part videos
    segment-001/
    segment-002/
  extensions/           optional continuation videos
    extend-001/
  downloads/
  final/
    complete.mp4
```

单视频只需要 `main/` 和 `final/`。续写或多段拼接时再使用 `segments/` 或 `extensions/`。

示例：先生成基础视频，再用返回的视频 URL 延展，最后拼接。

第 1 步：创建基础视频任务。

```bash
python3 providers/zlhub/scripts/zlhub_cli.py video-create \
  --config providers/zlhub/config/autocom.yaml \
  --prompt "生成一段 4 秒 9:16 竖屏物理竞赛课程营销视频，480p。画面为黑板公式、竞赛课堂和学生抬头思考，真实教育广告质感。" \
  --ratio 9:16 \
  --resolution 480p \
  --duration 4 \
  --generate-audio=false \
  --watermark=false \
  --out-dir runs/2026-04-22/20260422-physics-video-test/creation/videos/base
```

第 2 步：查询直到 `task.json` 里的 `status=succeeded` 且 `video_url` 非空。

```bash
python3 providers/zlhub/scripts/zlhub_cli.py video-get \
  --config providers/zlhub/config/autocom.yaml \
  --id cgt-xxx \
  --out-dir runs/2026-04-22/20260422-physics-video-test/creation/videos/base
```

第 3 步：使用基础视频的 `video_url` 作为 `视频1` 创建延展任务。

```bash
python3 providers/zlhub/scripts/zlhub_cli.py video-create \
  --config providers/zlhub/config/autocom.yaml \
  --prompt "向后延长视频1，保持同一课堂、同一画风和同一镜头运动，继续展示老师在黑板写出关键公式，画面自然衔接。" \
  --video reference_video="<base_video_url>" \
  --ratio 9:16 \
  --resolution 480p \
  --duration 4 \
  --generate-audio=false \
  --watermark=false \
  --out-dir runs/2026-04-22/20260422-physics-video-test/creation/videos/extend
```

第 4 步：及时下载两个临时签名视频 URL。

```bash
mkdir -p runs/2026-04-22/20260422-physics-video-test/creation/videos/downloads
curl -L "<base_video_url>" -o runs/2026-04-22/20260422-physics-video-test/creation/videos/downloads/base.mp4
curl -L "<extend_video_url>" -o runs/2026-04-22/20260422-physics-video-test/creation/videos/downloads/extend.mp4
```

第 5 步：用 ffmpeg 拼接本地文件。

```bash
mkdir -p runs/2026-04-22/20260422-physics-video-test/creation/videos/final
ffmpeg -y \
  -i runs/2026-04-22/20260422-physics-video-test/creation/videos/downloads/base.mp4 \
  -i runs/2026-04-22/20260422-physics-video-test/creation/videos/downloads/extend.mp4 \
  -filter_complex '[0:v:0][1:v:0]concat=n=2:v=1:a=0[v]' \
  -map '[v]' \
  -r 24 \
  -pix_fmt yuv420p \
  runs/2026-04-22/20260422-physics-video-test/creation/videos/final/complete.mp4
```

这只是一个示例流程，不是固定工作流。Agent 应根据 job brief 自己增加、删除、重复或重排步骤，但必须保持状态可落盘、可恢复。

## Constraints And Cost

- Observed ZLHub price: `doubao-seedance-2.0-fast` is `¥37 / 1M tokens`.
- Observed ZLHub price: `doubao-seedance-2.0` is `¥46 / 1M tokens`.
- Normal Seedance video duration should use `4-15` seconds.
- Observed on 2026-04-22: `doubao-seedance-2.0-fast` rejected `duration=1`, `duration=2`, and `duration=3` for text-to-video with HTTP 400 `InvalidParameter`; `duration=4` succeeded.
- Use `--resolution 480p` for low-cost quality checks when supported.
- For cost control, use `doubao-seedance-2.0-fast`, `--resolution 480p`, `--generate-audio=false`, and `--watermark=false` for smoke tests.
- Do not automatically retry paid `video-create` calls more than 2 times for the same step. After 2 failed attempts, stop and ask for direction instead of trying more durations, models, or prompts.
- Query existing `task_id` instead of creating a new task when a previous create call already succeeded.
- `doubao-seedance-2.0-mock` returned `model_not_found` for the tested API key/group.

## Safety

- Do not commit `outputs/`; it may contain signed URLs and generated media.
- Do not include API keys or signed media URLs in final answers.
