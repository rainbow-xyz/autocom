---
name: image-generation
description: Generate images for content production, marketing assets, covers, posters, thumbnails, first frames, or visual references. Current provider is ZLHub Seedream via the bundled Python CLI.
---

# Image Generation

用于生成海报、封面、缩略图、视频首帧或参考图。当前 provider 是 ZLHub Seedream。

## 准备

宿主机必须预先提供环境变量 `ZLHUB_API_KEY`。在普通 shell 中可以用 `export ZLHUB_API_KEY="your-zlhub-api-key"` 设置，但 agent 执行任务时不要把真实 key 写进命令行，也不要使用 `export ... && command` 链式命令。

如果运行时发现 `ZLHUB_API_KEY` 缺失，停止并提示用户在宿主机环境设置。不要把 key 写入配置、请求、日志或最终回答。

相关文件：

```text
providers/zlhub/scripts/zlhub_cli.py
providers/zlhub/config/autocom.yaml
providers/zlhub/examples/image_request.json
```

执行命令时：

- 从已安装的 `image-generation` skill 目录运行，或使用绝对路径。
- 能设置 `workdir` 时，用 `workdir`，不要写 `cd ... && command`。
- 如果 preflight 拒绝链式命令，拆成单步命令执行。

## 生成图片

```bash
python3 providers/zlhub/scripts/zlhub_cli.py image \
  --config providers/zlhub/config/autocom.yaml \
  --prompt "生成一张竖屏物理竞赛课程营销海报，真实高级教育广告质感。" \
  --out-dir outputs/physics-poster-001
```

复杂请求用 JSON：

```bash
python3 providers/zlhub/scripts/zlhub_cli.py image \
  --config providers/zlhub/config/autocom.yaml \
  --file providers/zlhub/examples/image_request.json \
  --out-dir outputs/physics-poster-001
```

输出位置：

```text
<out-dir>/zlhub/image/
  request.json
  response.json
  summary.json
  image_1.jpeg
```

## 默认参数

```text
model: doubao-seedream-5.0-lite
size: 1440x2560
response_format: url
watermark: false
```

已观察价格：`doubao-seedream-5.0-lite` 约 `¥0.22 / image`。

## 给视频使用

如果图片要传给视频接口作为 `first_frame`、`last_frame` 或 `reference_image`：

- 必须有公网可访问 `public_url`。
- 不能传本地路径、localhost、内网地址、登录态 URL 或 base64。
- 如果图片会被其它流程复用或发布，再登记到 `runs/YYYY-MM-DD/<job_id>/assets/asset_manifest.json`。

`outputs/` 只用于 CLI 测试；正式 job 使用 `runs/YYYY-MM-DD/<job_id>/creation/images/`。

## 安全

不要提交 `outputs/`、`runs/`、真实 API Key、签名 URL 或生成媒体文件。
