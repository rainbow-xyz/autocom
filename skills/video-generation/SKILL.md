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

## Constraints And Cost

- Observed ZLHub price: `doubao-seedance-2.0-fast` is `¥37 / 1M tokens`.
- Observed ZLHub price: `doubao-seedance-2.0` is `¥46 / 1M tokens`.
- Normal Seedance video duration should use `4-15` seconds.
- Use `--resolution 480p` for low-cost quality checks when supported.
- `doubao-seedance-2.0-mock` returned `model_not_found` for the tested API key/group.

## Safety

- Do not commit `outputs/`; it may contain signed URLs and generated media.
- Do not include API keys or signed media URLs in final answers.
