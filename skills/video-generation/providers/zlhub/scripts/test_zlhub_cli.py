import argparse
import json
import tempfile
import unittest
from pathlib import Path

import zlhub_cli


def ns(**kwargs):
    values = {
        "file": None,
        "prompt": None,
        "image": [],
        "video": [],
        "audio": [],
        "callback_url": None,
        "model": None,
        "resolution": None,
        "ratio": None,
        "duration": None,
        "generate_audio": None,
        "watermark": None,
        "size": None,
        "response_format": None,
        "download": None,
    }
    values.update(kwargs)
    return argparse.Namespace(**values)


class ZLHubCLITest(unittest.TestCase):
    def test_build_video_payload_from_flags(self):
        cfg = dict(zlhub_cli.DEFAULTS)
        args = ns(
            prompt="参考图片1生成视频",
            image=["reference_image=https://example.com/a.jpg"],
            model="doubao-seedance-2.0-fast",
            resolution="480p",
            ratio="9:16",
            duration=4,
            generate_audio=False,
            watermark=False,
        )

        payload = zlhub_cli.build_video_payload(args, cfg)

        self.assertEqual(payload["model"], "doubao-seedance-2.0-fast")
        self.assertEqual(payload["resolution"], "480p")
        self.assertEqual(payload["duration"], 4)
        self.assertIs(payload["generate_audio"], False)
        self.assertEqual(payload["content"][1]["image_url"]["url"], "https://example.com/a.jpg")

    def test_build_image_payload_from_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            request = Path(tmp) / "request.json"
            request.write_text(json.dumps({"prompt": "hello"}), encoding="utf-8")
            cfg = dict(zlhub_cli.DEFAULTS)
            args = ns(file=str(request), model="doubao-seedream-5.0-lite", size="1440x2560", response_format="url", watermark=False)

            payload = zlhub_cli.build_image_payload(args, cfg)

        self.assertEqual(payload["prompt"], "hello")
        self.assertEqual(payload["model"], "doubao-seedream-5.0-lite")
        self.assertEqual(payload["size"], "1440x2560")

    def test_load_config_reads_fast_model_and_resolution(self):
        with tempfile.TemporaryDirectory() as tmp:
            config = Path(tmp) / "autocom.yaml"
            config.write_text(
                """
zlhub:
  model: "doubao-seedance-2.0-fast"
  default_resolution: "480p"
  default_duration: 4
  default_generate_audio: false
""",
                encoding="utf-8",
            )

            cfg = zlhub_cli.load_config(str(config))

        self.assertEqual(cfg["model"], "doubao-seedance-2.0-fast")
        self.assertEqual(cfg["default_resolution"], "480p")
        self.assertEqual(cfg["default_duration"], 4)
        self.assertIs(cfg["default_generate_audio"], False)

    def test_trace_id_is_32_chars(self):
        self.assertEqual(len(zlhub_cli.trace_id(None)), 32)
        self.assertEqual(zlhub_cli.trace_id("a" * 32), "a" * 32)
        with self.assertRaises(zlhub_cli.CLIError):
            zlhub_cli.trace_id("short")

    def test_task_summary_reads_wrapped_video_url(self):
        response = {
            "code": "success",
            "data": {
                "id": "cgt-test",
                "status": "succeeded",
                "content": {"video_url": "https://example.com/final.mp4"},
            },
        }
        data = zlhub_cli.response_payload(response)

        summary = zlhub_cli.task_summary("cgt-test", data, response, "unknown")

        self.assertEqual(summary["status"], "succeeded")
        self.assertEqual(summary["video_url"], "https://example.com/final.mp4")

    def test_image_summary_reads_usage(self):
        summary = zlhub_cli.build_image_summary(
            {
                "model": "doubao-seedream-5-0-260128",
                "data": [{"url": "https://example.com/a.jpeg", "size": "1440x2560"}],
                "usage": {"generated_images": 1, "total_tokens": 14400},
            }
        )

        self.assertEqual(summary["generated_images"], 1)
        self.assertEqual(summary["total_tokens"], 14400)
        self.assertEqual(summary["images"][0]["url"], "https://example.com/a.jpeg")


if __name__ == "__main__":
    unittest.main()
