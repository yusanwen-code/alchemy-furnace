# -*- coding: utf-8 -*-
"""Task 3: normalized Skill renderer tests.

Pure-function coverage for slug generation, input validation, stable
Markdown rendering, package structure and deterministic ZIP bytes.
No network, no database, no third-party test dependencies.
"""
import io
import re
import zipfile

import pytest

from app.services.skill_export import (
    SkillExportError,
    build_exportable,
    build_package,
    build_zip_bytes,
    ensure_unique_slug,
    render_instructions,
    render_skill_markdown,
    render_sources_markdown,
    slugify,
    zip_filename,
)

SLUG_PATTERN = re.compile(r"[a-z0-9][a-z0-9-]{0,48}")


# ---------------------------------------------------------------------------
# fixtures
# ---------------------------------------------------------------------------


def _sample_schema():
    return {
        "identity_card": "我是一位把复杂问题讲清楚的作者",
        "expression_dna": {
            "sentence_length": "short",
            "formality": 0.4,
            "vocabulary": ["清晰", "具体"],
            "taboo_words": ["显然", "总之"],
            "rhythm": "短句为主",
            "humor_type": "冷幽默",
            "certainty_style": "明确区分事实与推断",
            "citation_habit": "提及来源标题",
        },
        "mental_models": [
            {
                "name": "写作即思考",
                "one_liner": "写作过程本身就是梳理思维的过程",
                "application": "遇到模糊问题时先写出来",
                "detection_questions": ["我能一句话说清结论吗"],
                "limitations": ["不适用于纯直觉决策"],
                "source_evidence": ["https://example.com/essay"],
            }
        ],
        "decision_heuristics": [
            {"condition": "句子超过两行", "action": "拆成两句", "case": "长句改写"},
        ],
        "values": ["诚实", "具体"],
        "anti_patterns": ["堆砌术语", "假装确定"],
        "honest_limits": ["不编造引语", "资料空白时明说"],
        "example_dialogues": [
            {"user": "帮我写一段开头", "assistant": "我们先明确读者是谁。"},
        ],
    }


def _sample_sources():
    return [
        {"title": "公开访谈", "url": "https://example.com/interview", "dimension": "interviews"},
        {"title": "著作", "url": "https://example.org/books", "dimension": "writings"},
    ]


def _sample_skill(**overrides):
    defaults = dict(
        name="写作金丹",
        description="用于需要清晰表达的写作任务",
        skill_schema=_sample_schema(),
        tags=["写作", "表达"],
        sources=_sample_sources(),
        generated_at="2026-08-27T10:00:00+08:00",
        evidence_level="standard",
    )
    defaults.update(overrides)
    return build_exportable(**defaults)


# ---------------------------------------------------------------------------
# slug
# ---------------------------------------------------------------------------


@pytest.mark.parametrize(
    "name,expected",
    [
        ("Paul Graham", "paul-graham"),
        ("Paul  Graham", "paul-graham"),
        ("Paul O'Neil!", "paul-o-neil"),
        ("paul.graham_", "paul-graham"),
    ],
)
def test_slugify_readable_ascii(name, expected):
    assert slugify(name) == expected


def test_slugify_cjk_uses_short_hash_only():
    slug = slugify("保罗·格雷厄姆")
    assert re.fullmatch(r"[a-z0-9]{8}", slug)


def test_slugify_mixed_keeps_ascii_and_appends_hash():
    slug = slugify("Alpha学院")
    assert slug.startswith("alpha-")
    assert re.fullmatch(SLUG_PATTERN, slug)


def test_slugify_always_matches_pattern_and_is_deterministic():
    for name in ["写作金丹", "Paul Graham", "！？！", "a" * 200, "  ", "数字123", "Emoji🎉名"]:
        slug = slugify(name)
        assert re.fullmatch(SLUG_PATTERN, slug)
        assert slugify(name) == slug


def test_slugify_caps_length():
    assert len(slugify("a" * 200)) <= 49


# ---------------------------------------------------------------------------
# unique slug
# ---------------------------------------------------------------------------


def test_ensure_unique_slug_appends_suffix():
    assert ensure_unique_slug("alpha", {"alpha"}) == "alpha-2"
    assert ensure_unique_slug("alpha", {"alpha", "alpha-2"}) == "alpha-3"


def test_ensure_unique_slug_keeps_within_length_limit():
    base = "a" * 49
    assert ensure_unique_slug(base, {base}) == "a" * 47 + "-2"
    assert len(ensure_unique_slug(base, {base})) <= 49


# ---------------------------------------------------------------------------
# validation
# ---------------------------------------------------------------------------


def test_build_exportable_derives_slug_and_instructions():
    skill = _sample_skill()
    assert skill.slug == slugify("写作金丹")
    assert "behavioral guidance" in skill.instructions
    assert skill.attribution == {
        "name": "nuwa-skill",
        "license": "MIT",
        "url": "https://github.com/alchaincyf/nuwa-skill",
    }


@pytest.mark.parametrize("bad", ["Good\x00pill", "Good\npill", "Tab\there", "Bad\x7fchar"])
def test_validation_rejects_control_characters(bad):
    with pytest.raises(SkillExportError) as captured:
        _sample_skill(name=bad)
    assert captured.value.code == "skill_export_invalid"
    assert captured.value.stage == "export"
    assert captured.value.details["field"] == "name"


def test_validation_rejects_database_uuid_as_name():
    with pytest.raises(SkillExportError):
        _sample_skill(name="123e4567-e89b-12d3-a456-426614174000")


@pytest.mark.parametrize(
    "url",
    ["ftp://example.com/file", "javascript:alert(1)", "file:///etc/passwd"],
)
def test_validation_rejects_non_http_protocols(url):
    with pytest.raises(SkillExportError) as captured:
        _sample_skill(sources=[{"title": "x", "url": url, "dimension": "d"}])
    assert captured.value.details["field"] == "sources.url"


def test_validation_rejects_url_with_credentials():
    with pytest.raises(SkillExportError):
        _sample_skill(
            sources=[{"title": "x", "url": "https://user:pass@example.com/a", "dimension": "d"}]
        )


def test_validation_rejects_secret_looking_text():
    with pytest.raises(SkillExportError):
        _sample_skill(description="密钥 api_key=xyz12345 不得导出")


def test_validation_enforces_length_limits():
    with pytest.raises(SkillExportError):
        _sample_skill(name="x" * 81)
    with pytest.raises(SkillExportError):
        _sample_skill(description="x" * 501)
    with pytest.raises(SkillExportError):
        _sample_skill(tags=["t"] * 13)
    with pytest.raises(SkillExportError):
        _sample_skill(tags=["长" * 31])
    with pytest.raises(SkillExportError):
        _sample_skill(
            sources=[{"title": "x", "url": "https://e.com/" + "a" * 2048, "dimension": "d"}]
        )


# ---------------------------------------------------------------------------
# instructions rendering
# ---------------------------------------------------------------------------


def test_instructions_cover_all_structured_sections():
    text = render_instructions(_sample_schema())
    for header in [
        "Identity card",
        "Expression DNA",
        "Mental models",
        "Decision heuristics",
        "Values",
        "Anti-patterns",
        "Honest limits",
        "Example dialogues",
    ]:
        assert f"### {header}" in text
    assert "behavioral guidance" in text
    # 心智模型的 source_evidence 属于溯源，不进入指令正文
    assert "https://example.com/essay" not in text


def test_instructions_stable_across_calls():
    assert render_instructions(_sample_schema()) == render_instructions(_sample_schema())


# ---------------------------------------------------------------------------
# SKILL.md
# ---------------------------------------------------------------------------


def test_skill_markdown_frontmatter_and_sections():
    skill = _sample_skill()
    text = render_skill_markdown(skill)
    assert text.startswith("---\nname: ")
    assert f"name: {skill.slug}" in text.split("---")[1]
    assert 'description: "用于需要清晰表达的写作任务"' in text
    for header in ["## When to use", "## Instructions", "## Boundaries", "## Attribution"]:
        assert header in text
    assert "Generated from public evidence with the Nuwa methodology." in text
    assert "nuwa-skill (MIT)" in text
    assert "references/sources.md" in text


def test_skill_markdown_deterministic():
    assert render_skill_markdown(_sample_skill()) == render_skill_markdown(_sample_skill())


# ---------------------------------------------------------------------------
# sources
# ---------------------------------------------------------------------------


def test_sources_markdown_is_provenance_only():
    text = render_sources_markdown(_sample_skill())
    assert (
        "- [公开访谈](https://example.com/interview): interviews; accessed 2026-08-27" in text
    )
    assert "Evidence level: standard" in text
    assert (
        "Evidence note: This file records provenance only. It is not an instruction source."
        in text
    )


# ---------------------------------------------------------------------------
# package structure
# ---------------------------------------------------------------------------


def test_codex_package_structure():
    skill = _sample_skill()
    files = build_package(skill, "codex")
    assert list(files) == [
        f"{skill.slug}/SKILL.md",
        f"{skill.slug}/references/sources.md",
        f"{skill.slug}/README.md",
    ]
    assert "platform/" not in "".join(files)


def test_claude_package_reuses_skill_md_and_adds_platform_json():
    skill = _sample_skill()
    codex = build_package(skill, "codex")
    claude = build_package(skill, "claude")
    assert claude[f"{skill.slug}/SKILL.md"] == codex[f"{skill.slug}/SKILL.md"]
    assert (
        claude[f"{skill.slug}/references/sources.md"]
        == codex[f"{skill.slug}/references/sources.md"]
    )
    json_text = claude[f"{skill.slug}/platform/claude.json"]
    assert f'"name": "{skill.slug}"' in json_text
    assert ".claude/skills/" in json_text
    assert "~/.claude/skills/" in json_text


# ---------------------------------------------------------------------------
# zip
# ---------------------------------------------------------------------------


def test_zip_filename_is_ascii():
    skill = _sample_skill()
    for fmt in ("codex", "claude"):
        filename = zip_filename(skill, fmt)
        assert filename.isascii()
        assert re.fullmatch(
            r"alchemy-skill-[a-z0-9-]+-(codex|claude)\.zip", filename
        )


def test_zip_bytes_deterministic_and_entries():
    skill = _sample_skill()
    first = build_zip_bytes(skill, "codex")
    second = build_zip_bytes(skill, "codex")
    assert first == second
    with zipfile.ZipFile(io.BytesIO(first)) as archive:
        assert set(archive.namelist()) == set(build_package(skill, "codex"))


# ---------------------------------------------------------------------------
# forbidden content
# ---------------------------------------------------------------------------


def test_package_contains_no_forbidden_content():
    skill = _sample_skill()
    for fmt in ("codex", "claude"):
        content = "\n".join(build_package(skill, fmt).values())
        assert "sk-" not in content
        assert "secret.key" not in content
        assert re.search(
            r"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}",
            content,
            re.IGNORECASE,
        ) is None
