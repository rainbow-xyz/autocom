# autocom

本仓库用于沉淀公司自动化能力。仓库按业务能力组织，`skills/` 是面向智能体的能力入口。

当前已有业务 skills：

```text
skills/image-generation    生图、封面、海报、缩略图、视频首帧/参考图
skills/video-generation    生视频、异步视频任务创建/查询、多模态素材转视频
```

ZLHub 不是仓库或 skill 的业务名称，只是当前生图/生视频的 provider：

```text
skills/image-generation/providers/zlhub
skills/video-generation/providers/zlhub
```

未来可以继续增加：

```text
docs/paperclip/workspace-convention.md
docs/paperclip/agent-instructions.md
skills/content-research
skills/social-publishing
skills/image-generation/providers/<other-provider>
skills/video-generation/providers/<other-provider>
```

## 给智能体安装

给智能体一个 Git 地址后，按需要安装对应业务 skill：

```bash
install-skill-from-github.py --repo <owner>/<repo> --path skills/image-generation
install-skill-from-github.py --repo <owner>/<repo> --path skills/video-generation
```

手动安装：

```bash
cp -R skills/image-generation "$CODEX_HOME/skills/image-generation"
cp -R skills/video-generation "$CODEX_HOME/skills/video-generation"
```

安装后重启 Codex/agent，让 skill 元数据生效。

## 目录结构

```text
skills/
  image-generation/
    SKILL.md
    providers/zlhub/
      scripts/zlhub_cli.py
      scripts/test_zlhub_cli.py
      examples/image_request.json
      config/autocom.yaml

  video-generation/
    SKILL.md
    providers/zlhub/
      scripts/zlhub_cli.py
      scripts/test_zlhub_cli.py
      examples/video_request.json
      config/autocom.yaml

archive/
  go-fallback/              历史 Go 版归档，不作为当前 skill 能力
  skill-drafts/             历史 skill 草稿归档；不得保留 SKILL.md

scripts/
  build_paperclip_package.sh
```

## Paperclip/OpenClaw 落盘约定

autocom 只保留单 agent 最小落盘约定：job 放哪里、如何恢复、哪些内容不能泄露。它不定义角色、不定义交接、不替代 Paperclip/OpenClaw 的编排能力。当前规范见：

```text
docs/paperclip/workspace-convention.md
```

这份规范不作为 skill 安装。它应放进 instructions 或 onboarding 文档中，只约束落盘、恢复和敏感信息。

可直接放进 Paperclip/OpenClaw agent instructions 的模板见：

```text
docs/paperclip/agent-instructions.md
```

建议在相关 instructions 中引用 `docs/paperclip/workspace-convention.md`。如果 Paperclip/OpenClaw 要拆分子 agent，让平台自己处理；autocom 不额外定义协作流程。

## 使用方式

宿主机需要预先设置 `ZLHUB_API_KEY`。Agent 执行命令时不要把真实 key 写进命令行，也不要使用 `export ... && command` 链式命令；如果执行环境支持 `workdir`，优先用 `workdir`，否则使用绝对路径。

进入已安装的 `skills/image-generation` 目录后，逐条执行：

```bash
python3 providers/zlhub/scripts/zlhub_cli.py image \
  --config providers/zlhub/config/autocom.yaml \
  --prompt "生成一张物理竞赛课程海报" \
  --out-dir outputs/poster-001
```

进入已安装的 `skills/video-generation` 目录后，逐条执行：

```bash
python3 providers/zlhub/scripts/zlhub_cli.py video-create \
  --config providers/zlhub/config/autocom.yaml \
  --prompt "生成一段 4 秒物理竞赛课程营销短视频" \
  --ratio 9:16 \
  --resolution 480p \
  --duration 4 \
  --generate-audio=false \
  --watermark=false \
  --out-dir outputs/video-001

python3 providers/zlhub/scripts/zlhub_cli.py video-get \
  --config providers/zlhub/config/autocom.yaml \
  --id cgt-xxx \
  --out-dir outputs/video-001
```

复杂生成流程不提供固定编排脚本。按 `skills/video-generation/SKILL.md` 调用 `zlhub_cli.py`、`curl`、`ffmpeg` 等基础工具。需要续接时从 `state.json` 和各 step 的 `task.json` 继续。

## 敏感信息规则

- `ZLHUB_API_KEY` 只放宿主机环境变量或本地 `.env`，不要写入配置文件、示例、请求 JSON 或文档。
- `outputs/` 可能包含签名 URL 和生成素材，默认忽略，不要提交。
- `dist/` 是打包产物，默认忽略，需要时用脚本重新生成。
- 真实 API Key、临时 TOS 签名 URL、生成媒体文件都不要提交。

## 验证与打包

逐条执行：

```bash
python3 -m unittest discover -s skills/image-generation/providers/zlhub/scripts -p 'test_*.py'
python3 -m unittest discover -s skills/video-generation/providers/zlhub/scripts -p 'test_*.py'
scripts/build_paperclip_package.sh
```
