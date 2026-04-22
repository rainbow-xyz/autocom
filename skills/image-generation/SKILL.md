---
name: image-generation
description: Generate images for content production, marketing assets, covers, posters, thumbnails, first frames, or visual references. Current provider is ZLHub Seedream via the bundled Python CLI.
---

# Image Generation

Use this skill when an agent needs to generate images, covers, posters, thumbnails, first-frame assets, or visual references for downstream content/video workflows.

Current provider:

```text
ZLHub Seedream
```

ZLHub is a provider under this business skill. Future image providers should be added under `providers/<provider-name>/`.

## Files

```text
providers/zlhub/scripts/zlhub_cli.py
providers/zlhub/examples/image_request.json
providers/zlhub/config/autocom.yaml
```

Run commands from the installed `image-generation` skill directory, or pass absolute paths.

For Paperclip/OpenClaw jobs, the workspace convention wins:

```text
runs/YYYY-MM-DD/<job_id>/creation/images/
```

Examples using `outputs/` are standalone CLI smoke tests only. Do not use `outputs/` as the formal job directory.

## Environment

Set the API key on the host:

```bash
export ZLHUB_API_KEY="your-zlhub-api-key"
```

Never write real keys into config, examples, requests, logs, output summaries, or final answers.

## Generate Image

Default provider/model:

```text
provider: ZLHub
model: doubao-seedream-5.0-lite
size: 1440x2560
response_format: url
watermark: false
```

Command:

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

For formal jobs, copy or generate selected outputs into `creation/images/`, and register reusable images in:

```text
runs/YYYY-MM-DD/<job_id>/assets/asset_manifest.json
```

When a generated image will be used by a video API as `first_frame`, `last_frame`, or `reference_image`, make sure the manifest entry has a reachable `public_url`, `usable_for_api=true`, and a matching `allowed_uses` value.

Use JSON for complex requests:

```bash
python3 providers/zlhub/scripts/zlhub_cli.py image \
  --config providers/zlhub/config/autocom.yaml \
  --file providers/zlhub/examples/image_request.json \
  --out-dir outputs/physics-poster-001
```

Disable download when only the returned URL is needed:

```bash
python3 providers/zlhub/scripts/zlhub_cli.py image \
  --config providers/zlhub/config/autocom.yaml \
  --prompt "生成一张课程海报" \
  --download=false \
  --out-dir outputs/image-test
```

## Constraints And Cost

- `doubao-seedream-5.0-lite` is the default low-cost image model.
- Observed ZLHub price: `¥0.22 / image`.
- Use `1440x2560` for 9:16 because `1024x1792` was rejected for not meeting the minimum `3686400` pixels.
- `X-Trace-ID` must be exactly 32 characters; the CLI auto-generates a valid value.

## Safety

- Do not commit `outputs/`; it may contain signed URLs and generated media.
- Do not include API keys or signed media URLs in final answers.
