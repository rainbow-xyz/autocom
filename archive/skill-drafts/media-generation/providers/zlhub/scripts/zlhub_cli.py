#!/usr/bin/env python3
"""ZLHub image/video CLI for agent environments.

Only Python standard library is used so the script can run in most agent hosts.
"""

from __future__ import annotations

import argparse
import json
import os
import secrets
import shutil
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any


DEFAULTS: dict[str, Any] = {
    "api_base": "https://api.zlhub.cn",
    "model": "doubao-seedance-2.0-fast",
    "image_model": "doubao-seedream-5.0-lite",
    "callback_url": "",
    "default_resolution": "",
    "default_image_size": "1440x2560",
    "default_image_format": "url",
    "default_download_image": True,
    "default_ratio": "9:16",
    "default_duration": 8,
    "default_generate_audio": True,
    "default_watermark": False,
}


class CLIError(Exception):
    pass


class HTTPResult:
    def __init__(self, status_code: int, body: bytes, headers: Any = None):
        self.status_code = status_code
        self.body = body
        self.headers = headers or {}


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    try:
        args.func(args)
        return 0
    except CLIError as exc:
        print(str(exc), file=sys.stderr)
        return 1


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="zlhub_cli.py", description="ZLHub 图片/视频生成 CLI")
    subparsers = parser.add_subparsers(dest="command", required=True)

    image = subparsers.add_parser("image", help="生成图片")
    add_common_flags(image)
    image.add_argument("--file", help="完整图片请求 JSON 文件")
    image.add_argument("--prompt", help="图片提示词")
    image.add_argument("--model", help="图片生成模型")
    image.add_argument("--size", help="图片尺寸，例如 1440x2560")
    image.add_argument("--response-format", help="响应格式，默认 url")
    image.add_argument("--watermark", type=parse_bool, help="是否添加水印")
    image.add_argument("--download", type=parse_bool, help="是否自动下载返回图片")
    image.add_argument("--out-dir", required=True, help="输出目录")
    image.set_defaults(func=run_image)

    create = subparsers.add_parser("video-create", help="创建视频任务")
    add_common_flags(create)
    create.add_argument("--file", help="完整视频请求 JSON 文件")
    create.add_argument("--prompt", help="视频提示词")
    create.add_argument("--image", action="append", default=[], help="图片素材，格式 role=url，可重复")
    create.add_argument("--video", action="append", default=[], help="视频素材，格式 role=url，可重复")
    create.add_argument("--audio", action="append", default=[], help="音频素材，格式 role=url，可重复")
    create.add_argument("--callback-url", help="任务状态回调地址")
    create.add_argument("--model", help="视频生成模型")
    create.add_argument("--resolution", help="视频分辨率，例如 480p 或 720p")
    create.add_argument("--ratio", help="视频比例，例如 9:16 或 16:9")
    create.add_argument("--duration", type=int, help="视频时长，单位秒")
    create.add_argument("--generate-audio", type=parse_bool, help="是否生成音频")
    create.add_argument("--watermark", type=parse_bool, help="是否添加水印")
    create.add_argument("--out-dir", required=True, help="输出目录")
    create.set_defaults(func=run_video_create)

    get = subparsers.add_parser("video-get", help="查询视频任务")
    add_common_flags(get)
    get.add_argument("--id", required=True, help="ZLHub 任务 ID")
    get.add_argument("--out-dir", required=True, help="输出目录")
    get.set_defaults(func=run_video_get)
    return parser


def add_common_flags(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("--config", default="autocom.yaml", help="项目配置文件路径")
    parser.add_argument("--api-base", help="ZLHub API Base")
    parser.add_argument("--trace-id", help="请求追踪 ID，必须 32 字符；不传则自动生成")


def run_image(args: argparse.Namespace) -> None:
    cfg = load_config(args.config)
    api_key = require_api_key()
    out_dir = Path(args.out_dir) / "zlhub" / "image"
    out_dir.mkdir(parents=True, exist_ok=True)

    payload = build_image_payload(args, cfg)
    write_json(out_dir / "request.json", payload)

    result = post_json(
        join_api(value(args.api_base, cfg["api_base"]), "/v1/images/generations"),
        api_key,
        trace_id(args.trace_id),
        payload,
    )
    (out_dir / "response.json").write_bytes(result.body)
    if not 200 <= result.status_code < 300:
        raise CLIError(f"生成图片失败，HTTP 状态码：{result.status_code}")

    response = parse_json_bytes(result.body, "图片响应不是合法 JSON")
    summary = build_image_summary(response)
    write_json(out_dir / "summary.json", summary)

    should_download = args.download if args.download is not None else bool(cfg["default_download_image"])
    if should_download:
        download_images(summary["images"], out_dir)
    print(f"图片生成成功，共 {len(summary['images'])} 张")


def run_video_create(args: argparse.Namespace) -> None:
    cfg = load_config(args.config)
    api_key = require_api_key()
    out_dir = Path(args.out_dir) / "zlhub"
    out_dir.mkdir(parents=True, exist_ok=True)

    payload = build_video_payload(args, cfg)
    write_json(out_dir / "request.json", payload)

    result = post_json(
        join_api(value(args.api_base, cfg["api_base"]), "/v1/task/create"),
        api_key,
        trace_id(args.trace_id),
        payload,
    )
    (out_dir / "create_response.json").write_bytes(result.body)
    if not 200 <= result.status_code < 300:
        raise CLIError(f"创建视频任务失败，HTTP 状态码：{result.status_code}")

    response = parse_json_bytes(result.body, "创建响应不是合法 JSON")
    data = response_payload(response)
    task_id = data.get("id")
    if not task_id:
        raise CLIError("创建响应缺少任务 ID 字段 id")

    summary = task_summary(task_id, data, response, fallback="submitted")
    write_json(out_dir / "task.json", summary)
    print(f"创建成功，任务 ID：{task_id}")


def run_video_get(args: argparse.Namespace) -> None:
    cfg = load_config(args.config)
    api_key = require_api_key()
    out_dir = Path(args.out_dir) / "zlhub"
    out_dir.mkdir(parents=True, exist_ok=True)

    endpoint = join_api(value(args.api_base, cfg["api_base"]), "/v1/task/get/" + urllib.parse.quote(args.id, safe=""))
    result = get_json(endpoint, api_key, trace_id(args.trace_id))
    (out_dir / "query_response.json").write_bytes(result.body)
    if not 200 <= result.status_code < 300:
        raise CLIError(f"查询视频任务失败，HTTP 状态码：{result.status_code}")

    response = parse_json_bytes(result.body, "查询响应不是合法 JSON")
    data = response_payload(response)
    task_id = data.get("id") or args.id
    summary = task_summary(task_id, data, response, fallback="unknown")
    write_json(out_dir / "task.json", summary)
    print(f"查询成功，任务 ID：{task_id}，状态：{summary['status']}")


def build_image_payload(args: argparse.Namespace, cfg: dict[str, Any]) -> dict[str, Any]:
    if args.file:
        payload = read_json_file(args.file, "图片请求 JSON 文件格式错误")
    else:
        if not args.prompt:
            raise CLIError("缺少 --prompt；如果要使用完整图片请求体，请传 --file request.json")
        payload = {"prompt": args.prompt}

    apply_string(payload, "model", args.model, cfg["image_model"], DEFAULTS["image_model"])
    apply_string(payload, "size", args.size, cfg["default_image_size"], DEFAULTS["default_image_size"])
    apply_string(payload, "response_format", args.response_format, cfg["default_image_format"], DEFAULTS["default_image_format"])
    apply_bool(payload, "watermark", args.watermark, cfg["default_watermark"], DEFAULTS["default_watermark"])
    return payload


def build_video_payload(args: argparse.Namespace, cfg: dict[str, Any]) -> dict[str, Any]:
    if args.file:
        payload = read_json_file(args.file, "视频请求 JSON 文件格式错误")
    else:
        if not args.prompt:
            raise CLIError("缺少 --prompt；如果要使用完整视频请求体，请传 --file request.json")
        content: list[dict[str, Any]] = [{"type": "text", "text": args.prompt}]
        content.extend(asset_items("image_url", "image_url", args.image))
        content.extend(asset_items("video_url", "video_url", args.video))
        content.extend(asset_items("audio_url", "audio_url", args.audio))
        payload = {"content": content}

    apply_string(payload, "model", args.model, cfg["model"], DEFAULTS["model"])
    apply_optional_string(payload, "resolution", args.resolution, cfg["default_resolution"])
    apply_string(payload, "ratio", args.ratio, cfg["default_ratio"], DEFAULTS["default_ratio"])
    apply_int(payload, "duration", args.duration, cfg["default_duration"], DEFAULTS["default_duration"])
    apply_bool(payload, "generate_audio", args.generate_audio, cfg["default_generate_audio"], DEFAULTS["default_generate_audio"])
    apply_bool(payload, "watermark", args.watermark, cfg["default_watermark"], DEFAULTS["default_watermark"])

    if args.callback_url is not None:
        if args.callback_url:
            payload["callback_url"] = args.callback_url
        else:
            payload.pop("callback_url", None)
    elif "callback_url" not in payload and cfg["callback_url"]:
        payload["callback_url"] = cfg["callback_url"]
    return payload


def asset_items(item_type: str, field: str, values: list[str]) -> list[dict[str, Any]]:
    items = []
    for raw in values:
        if "=" not in raw:
            raise CLIError(f"素材参数格式错误：{raw}，正确格式为 role=url")
        role, url = raw.split("=", 1)
        role, url = role.strip(), url.strip()
        if not role or not url:
            raise CLIError(f"素材参数格式错误：{raw}，正确格式为 role=url")
        items.append({"type": item_type, field: {"url": url}, "role": role})
    return items


def load_config(path: str) -> dict[str, Any]:
    cfg = dict(DEFAULTS)
    config_path = Path(path)
    if not config_path.exists():
        return cfg
    section = ""
    for line_no, original in enumerate(config_path.read_text(encoding="utf-8").splitlines(), 1):
        line = strip_comment(original).strip()
        if not line:
            continue
        if not original.startswith(" ") and line.endswith(":"):
            section = line[:-1].strip()
            continue
        if section != "zlhub":
            continue
        if ":" not in line:
            raise CLIError(f"配置文件第 {line_no} 行格式错误：{line}")
        key, raw_value = line.split(":", 1)
        key = key.strip()
        raw_value = normalize_yaml_value(raw_value.strip())
        if key in {"default_duration"}:
            cfg[key] = int(raw_value)
        elif key in {"default_generate_audio", "default_watermark", "default_download_image"}:
            cfg[key] = parse_bool(raw_value)
        elif key in cfg:
            cfg[key] = raw_value
    return cfg


def strip_comment(line: str) -> str:
    in_single = False
    in_double = False
    for index, char in enumerate(line):
        if char == "'" and not in_double:
            in_single = not in_single
        elif char == '"' and not in_single:
            in_double = not in_double
        elif char == "#" and not in_single and not in_double:
            return line[:index]
    return line


def normalize_yaml_value(value_: str) -> str:
    if len(value_) >= 2 and value_[0] == value_[-1] and value_[0] in {"'", '"'}:
        return value_[1:-1]
    return value_


def parse_bool(value_: Any) -> bool:
    if isinstance(value_, bool):
        return value_
    text = str(value_).strip().lower()
    if text in {"1", "true", "yes", "y", "on"}:
        return True
    if text in {"0", "false", "no", "n", "off"}:
        return False
    raise argparse.ArgumentTypeError(f"布尔值格式错误：{value_}")


def apply_string(payload: dict[str, Any], key: str, cli_value: str | None, cfg_value: str, default: str) -> None:
    if cli_value is not None:
        if cli_value:
            payload[key] = cli_value
        return
    if key not in payload:
        payload[key] = cfg_value or default


def apply_optional_string(payload: dict[str, Any], key: str, cli_value: str | None, cfg_value: str) -> None:
    if cli_value is not None:
        if cli_value:
            payload[key] = cli_value
        else:
            payload.pop(key, None)
        return
    if key not in payload and cfg_value:
        payload[key] = cfg_value


def apply_int(payload: dict[str, Any], key: str, cli_value: int | None, cfg_value: int, default: int) -> None:
    if cli_value is not None:
        payload[key] = cli_value
    elif key not in payload:
        payload[key] = cfg_value or default


def apply_bool(payload: dict[str, Any], key: str, cli_value: bool | None, cfg_value: bool, default: bool) -> None:
    if cli_value is not None:
        payload[key] = cli_value
    elif key not in payload:
        payload[key] = bool(cfg_value if cfg_value != default else default)


def require_api_key() -> str:
    api_key = os.environ.get("ZLHUB_API_KEY", "").strip()
    if not api_key:
        raise CLIError("缺少环境变量 ZLHUB_API_KEY，请先在宿主机环境变量中配置 API Key")
    return api_key


def trace_id(given: str | None) -> str:
    if given:
        if len(given) != 32:
            raise CLIError("X-Trace-ID 必须正好 32 个字符")
        return given
    return secrets.token_hex(16)


def join_api(api_base: str, path: str) -> str:
    return api_base.rstrip("/") + path


def post_json(endpoint: str, api_key: str, request_trace_id: str, payload: dict[str, Any]) -> HTTPResult:
    body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    request = urllib.request.Request(endpoint, data=body, method="POST")
    add_headers(request, api_key, request_trace_id)
    return do_request(request)


def get_json(endpoint: str, api_key: str, request_trace_id: str) -> HTTPResult:
    request = urllib.request.Request(endpoint, method="GET")
    add_headers(request, api_key, request_trace_id)
    return do_request(request)


def add_headers(request: urllib.request.Request, api_key: str, request_trace_id: str) -> None:
    request.add_header("Authorization", "Bearer " + api_key)
    request.add_header("Content-Type", "application/json")
    request.add_header("X-Trace-ID", request_trace_id)


def do_request(request: urllib.request.Request) -> HTTPResult:
    try:
        with urllib.request.urlopen(request, timeout=60) as response:
            return HTTPResult(response.status, response.read(), response.headers)
    except urllib.error.HTTPError as exc:
        return HTTPResult(exc.code, exc.read(), exc.headers)
    except urllib.error.URLError as exc:
        raise CLIError(f"请求 ZLHub 失败：{exc}") from exc


def download_images(images: list[dict[str, Any]], out_dir: Path) -> None:
    for index, image in enumerate(images, 1):
        url = image.get("url")
        if not url:
            continue
        request = urllib.request.Request(url, method="GET")
        try:
            with urllib.request.urlopen(request, timeout=60) as response:
                ext = image_ext(url, response.headers.get("Content-Type", ""))
                with (out_dir / f"image_{index}{ext}").open("wb") as file:
                    shutil.copyfileobj(response, file)
        except urllib.error.URLError as exc:
            raise CLIError(f"下载图片失败：{exc}") from exc


def image_ext(url: str, content_type: str) -> str:
    content_type = content_type.lower()
    if "png" in content_type:
        return ".png"
    if "webp" in content_type:
        return ".webp"
    if "jpeg" in content_type or "jpg" in content_type:
        return ".jpeg"
    path = urllib.parse.urlparse(url).path.lower()
    suffix = Path(path).suffix
    if suffix in {".jpg", ".jpeg", ".png", ".webp"}:
        return suffix
    return ".jpeg"


def build_image_summary(response: dict[str, Any]) -> dict[str, Any]:
    images = []
    for item in response.get("data", []) or []:
        if isinstance(item, dict):
            images.append({"url": item.get("url", ""), "size": item.get("size", "")})
    usage = response.get("usage") if isinstance(response.get("usage"), dict) else {}
    return {
        "model": response.get("model", ""),
        "generated_images": int(usage.get("generated_images") or len(images)),
        "total_tokens": int(usage.get("total_tokens") or 0),
        "images": images,
        "updated_at": now_iso(),
    }


def response_payload(response: dict[str, Any]) -> dict[str, Any]:
    data = response.get("data")
    return data if isinstance(data, dict) else response


def task_summary(task_id: str, data: dict[str, Any], root: dict[str, Any], fallback: str) -> dict[str, Any]:
    return {
        "task_id": task_id,
        "status": data.get("status") or fallback,
        "video_url": extract_video_url(data),
        "error": data.get("error") or root.get("error"),
        "updated_at": now_iso(),
    }


def extract_video_url(data: dict[str, Any]) -> str:
    for path in (("video_url",), ("content", "video_url"), ("output", "video_url"), ("result", "video_url")):
        current: Any = data
        for key in path:
            if not isinstance(current, dict):
                current = None
                break
            current = current.get(key)
        if isinstance(current, str) and current:
            return current
    return ""


def read_json_file(path: str, error_message: str) -> dict[str, Any]:
    try:
        data = json.loads(Path(path).read_text(encoding="utf-8"))
    except OSError as exc:
        raise CLIError(f"读取请求 JSON 文件失败：{exc}") from exc
    except json.JSONDecodeError as exc:
        raise CLIError(f"{error_message}：{exc}") from exc
    if not isinstance(data, dict):
        raise CLIError(f"{error_message}：顶层必须是 JSON object")
    return data


def parse_json_bytes(body: bytes, message: str) -> dict[str, Any]:
    try:
        data = json.loads(body.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise CLIError(f"{message}：{exc}") from exc
    if not isinstance(data, dict):
        raise CLIError(f"{message}：顶层必须是 JSON object")
    return data


def write_json(path: Path, data: dict[str, Any]) -> None:
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def value(cli_value: str | None, cfg_value: str) -> str:
    return cli_value or cfg_value


def now_iso() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%S%z", time.localtime())


if __name__ == "__main__":
    raise SystemExit(main())
