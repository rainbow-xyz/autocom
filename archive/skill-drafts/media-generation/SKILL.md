---
name: media-generation
description: Generate marketing or content-production images and videos for agent workflows. Use bundled provider tools, currently ZLHub Seedream for images and Seedance for videos, to create assets, create/query async video tasks, manage outputs, and avoid leaking API keys or signed media URLs.
---

# Media Generation

Use this skill when an agent needs to generate images or videos for content production, marketing material, social-media drafts, course promotion, or automated publishing workflows.

Provider selection:

```text
Current image provider: ZLHub Seedream
Current video provider: ZLHub Seedance
```

ZLHub is an implementation detail inside this skill, not the business capability name. If another provider is added later, keep the same business workflow and add it under `providers/<provider-name>/`.

## Files

```text
providers/zlhub/scripts/zlhub_cli.py          # Python standard-library CLI
providers/zlhub/examples/image_request.json   # Seedream image request example
providers/zlhub/examples/video_request.json   # Seedance video request example
providers/zlhub/config/autocom.yaml           # Safe default ZLHub config
```

Run commands from the installed skill directory, or pass absolute paths to these files.

## Environment

For ZLHub, set the API key on the host:

```bash
export ZLHUB_API_KEY="your-zlhub-api-key"
```

Never write real keys into config, examples, requests, logs, output summaries, or final answers.

## ZLHub Defaults

Safe defaults live in `providers/zlhub/config/autocom.yaml`.

Important defaults:

```text
image_model: doubao-seedream-5.0-lite
video model: doubao-seedance-2.0-fast
default_image_size: 1440x2560
default_ratio: 9:16
```

Cost guidance:

```text
Use doubao-seedance-2.0-fast for drafts and batch tests.
Use doubao-seedance-2.0 only for final video output because it costs more.
Use --resolution 480p for low-cost video checks when supported.
```

Parameter precedence:

```text
command line flags > provider config > script defaults
```

## Generate Image With ZLHub

```bash
python3 providers/zlhub/scripts/zlhub_cli.py image \
  --config providers/zlhub/config/autocom.yaml \
  --prompt "生成一张竖屏物理竞赛课程营销海报，真实高级教育广告质感。画面中心是一名专注思考的高中生，背景是黑板上的力学和电磁学公式。底部留白用于后期添加 logo 和二维码。" \
  --out-dir outputs/physics-poster-001
```

This calls:

```text
POST /v1/images/generations
```

Outputs:

```text
outputs/physics-poster-001/zlhub/image/
  request.json
  response.json
  summary.json
  image_1.jpeg
```

Use JSON for complex image requests:

```bash
python3 providers/zlhub/scripts/zlhub_cli.py image \
  --config providers/zlhub/config/autocom.yaml \
  --file providers/zlhub/examples/image_request.json \
  --out-dir outputs/physics-poster-001
```

Disable image download when only the returned URL is needed:

```bash
python3 providers/zlhub/scripts/zlhub_cli.py image \
  --config providers/zlhub/config/autocom.yaml \
  --prompt "生成一张课程海报" \
  --download=false \
  --out-dir outputs/image-test
```

Image constraints learned from ZLHub:

- `X-Trace-ID` must be exactly 32 characters; the CLI auto-generates a valid value.
- `doubao-seedream-5.0-lite` rejected `1024x1792`; use `1440x2560` for 9:16 because it meets the minimum `3686400` pixels.

## Create Video Task With ZLHub

Seedance video generation is asynchronous:

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

Use the standard model only for final output:

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

## Failure Handling

- `缺少环境变量 ZLHUB_API_KEY`: set the host environment variable.
- `X-Trace-ID ... must be exactly 32 characters`: the CLI auto-generates a valid value; check only if passing `--trace-id`.
- `token quota is not enough`: reduce duration/model cost or ask the user to top up.
- `model_not_found` or `无可用渠道`: the model is not configured for the current ZLHub group/key.
- Image `size` invalid: use `1440x2560` for 9:16 Seedream Lite tests.

Keep API keys and signed media URLs out of committed files and final answers.
