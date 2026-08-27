# -*- coding: utf-8 -*-
"""Skill 导出接口测试: POST /api/v1/distillation/skill-export(ZIP 二进制响应)。"""
import io
import zipfile

from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)

VALID_SKILL = {
    "name": "结构化金丹",
    "description": "一份结构化的语言风格技能包",
    "skill_schema": {"identity_card": "我是金丹"},
    "tags": ["语言", "风格"],
    "sources": [
        {
            "title": "公开资料",
            "url": "https://example.com/intro",
            "dimension": "decision_heuristics",
        }
    ],
    "generated_at": "2026-08-27T10:00:00Z",
}


def test_skill_export_codex_returns_zip_with_skill_md():
    resp = client.post("/api/v1/distillation/skill-export", json={**VALID_SKILL, "format": "codex"})

    assert resp.status_code == 200
    assert resp.headers["content-type"] == "application/zip"
    disposition = resp.headers["content-disposition"]
    assert disposition.startswith('attachment; filename="alchemy-skill-')
    assert disposition.endswith("-codex.zip\"")

    with zipfile.ZipFile(io.BytesIO(resp.content)) as archive:
        names = archive.namelist()
        assert any(name.endswith("SKILL.md") for name in names)
        assert any(name.endswith("references/sources.md") for name in names)
        assert any(name.endswith("README.md") for name in names)
        assert not any(name.endswith("platform/claude.json") for name in names)


def test_skill_export_claude_adds_platform_json():
    resp = client.post("/api/v1/distillation/skill-export", json={**VALID_SKILL, "format": "claude"})

    assert resp.status_code == 200
    disposition = resp.headers["content-disposition"]
    assert disposition.endswith("-claude.zip\"")

    with zipfile.ZipFile(io.BytesIO(resp.content)) as archive:
        names = archive.namelist()
        assert any(name.endswith("platform/claude.json") for name in names)


def test_skill_export_deterministic_bytes():
    first = client.post("/api/v1/distillation/skill-export", json={**VALID_SKILL, "format": "codex"})
    second = client.post("/api/v1/distillation/skill-export", json={**VALID_SKILL, "format": "codex"})

    assert first.content == second.content


def test_skill_export_invalid_content_returns_422_with_code():
    invalid = {**VALID_SKILL, "name": "", "format": "codex"}
    resp = client.post("/api/v1/distillation/skill-export", json=invalid)

    assert resp.status_code == 422
    detail = resp.json()["detail"]
    assert detail["code"] == "skill_export_invalid"
    assert detail["stage"] == "export"
    assert detail["retryable"] is False


def test_skill_export_invalid_format_rejected():
    resp = client.post("/api/v1/distillation/skill-export", json={**VALID_SKILL, "format": "yaml"})

    assert resp.status_code == 422
