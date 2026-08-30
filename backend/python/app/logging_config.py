# -*- coding: utf-8 -*-
"""日志配置：支持 LOG_FORMAT=json 预设与自定义格式串。

Go 网关/桌面端以 LOG_FORMAT=json 拉起引擎时，旧代码把 "json" 直接传给
logging.basicConfig(format=...) ，logging.Formatter 对无 % 占位符的格式串抛
ValueError 导致启动崩溃。本模块归一化配置入口：

- format_spec == "json"（大小写不敏感）→ JsonLogFormatter，每行一条 JSON；
- format_spec 含 "%(" → 视为自定义 logging 格式串原样使用；
- 其他任何值（含 None/空/未知预设）→ DEFAULT_TEXT_FORMAT 文本回退，绝不抛异常。
"""
import json
import logging
import sys

DEFAULT_TEXT_FORMAT = "%(asctime)s [%(levelname)s] %(name)s - %(message)s"


class JsonLogFormatter(logging.Formatter):
    def format(self, record: logging.LogRecord) -> str:
        return json.dumps(
            {
                "timestamp": self.formatTime(record, self.datefmt),
                "level": record.levelname,
                "logger": record.name,
                "message": record.getMessage(),
            },
            ensure_ascii=False,
        )


def configure_logging(level_name: str, format_spec: str, stream=None) -> None:
    handler = logging.StreamHandler(stream or sys.stdout)
    normalized = (format_spec or "text").strip()
    if normalized.lower() == "json":
        handler.setFormatter(JsonLogFormatter())
    else:
        selected = (
            normalized
            if "%(" in normalized
            else DEFAULT_TEXT_FORMAT
        )
        handler.setFormatter(logging.Formatter(selected))
    logging.basicConfig(
        level=getattr(logging, level_name.upper(), logging.INFO),
        handlers=[handler],
        force=True,
    )
