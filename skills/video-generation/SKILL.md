---
name: video-generation
description: Create and query asynchronous video generation tasks for content production, marketing videos, social-media clips, or multimodal image/video/audio-to-video workflows. Current provider is ZLHub Seedance via the bundled Python CLI.
---

# Video Generation

用于创建和查询异步视频生成任务。当前 provider 是 ZLHub Seedance。

## 准备

宿主机必须预先提供环境变量 `ZLHUB_API_KEY`。不要把真实 key 写进命令行、配置、请求、日志或最终回答。

相关文件：

```text
providers/zlhub/scripts/zlhub_cli.py
providers/zlhub/config/autocom.yaml
providers/zlhub/examples/video_request.json
```

执行命令时：

- 从已安装的 `video-generation` skill 目录运行，或使用绝对路径。
- 能设置 `workdir` 时，用 `workdir`，不要写 `cd ... && command`。
- 如果 preflight 拒绝链式命令，拆成单步执行。

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

素材只能传公网 URL，不能传本地路径、localhost、内网地址、登录态 URL 或 base64。提示词中用 `图片1`、`视频1`、`音频1` 这类序号引用素材。

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

`task.json` 会保存 `task_id`、`status`、`video_url`、`error`、`updated_at`。

任务成功后及时下载 `video_url`，因为通常是临时签名 URL。不要频繁轮询；真实任务无 callback 时，建议创建约 10 分钟后查询，同一任务最多每分钟查一次。

## 简单流程

单视频：

```text
create -> query -> download video_url -> final
```

续写或多段视频：

```text
query 上一段拿到 video_url -> 用 --video reference_video=<video_url> 创建下一段 -> 下载 -> ffmpeg 拼接
```

需要恢复时只记住两件事：

- 已有 `task_id` 就继续 query，不要重复 create。
- 同一个付费 create/generate 失败最多重试 2 次。

正式 job 建议输出到：

```text
runs/YYYY-MM-DD/<job_id>/outputs/videos/
```

简单测试可以继续用：

```text
outputs/<name>/
```

## 成本和限制

- 默认低成本测试：`doubao-seedance-2.0-fast`、`480p`、`generate_audio=false`。
- 已观察价格：`doubao-seedance-2.0-fast` 约 `¥37 / 1M tokens`，`doubao-seedance-2.0` 约 `¥46 / 1M tokens`。
- 已观察限制：`doubao-seedance-2.0-fast` 文生视频 `duration=1/2/3` 会返回 `InvalidParameter`，`duration=4` 成功。
- 正常视频时长建议使用 `4-15` 秒。
- `doubao-seedance-2.0-mock` 对当前测试账号返回过 `model_not_found`。

## 安全

不要提交 `outputs/`、`runs/`、真实 API Key、签名 URL 或生成媒体文件。
