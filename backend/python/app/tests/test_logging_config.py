# -*- coding: utf-8 -*-
"""LOG_FORMAT=json 启动崩溃回归测试。

引擎由 Go 网关/桌面端以 LOG_FORMAT=json 拉起时，旧代码把 "json" 直接传给
logging.basicConfig(format=...) ，logging.Formatter 对无 % 占位符的格式串抛
ValueError，导致启动即崩。本模块验证 JSON 预设与未知预设回退。
"""
import io
import json
import logging

from app.logging_config import configure_logging


def test_json_log_format_is_valid_json_and_does_not_raise():
    output = io.StringIO()
    configure_logging("INFO", "json", stream=output)
    logging.getLogger("nuwa-test").info("engine ready")

    record = json.loads(output.getvalue().strip())
    assert record["level"] == "INFO"
    assert record["logger"] == "nuwa-test"
    assert record["message"] == "engine ready"


def test_unknown_log_preset_falls_back_without_raising():
    output = io.StringIO()
    configure_logging("INFO", "unknown-preset", stream=output)
    logging.getLogger("nuwa-test").warning("still logging")

    text = output.getvalue()
    assert "still logging" in text
    assert text.startswith("%(asctime)s") is False  # 不把 "unknown-preset" 当格式串
