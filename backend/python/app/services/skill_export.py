"""Normalized, deterministic Skill renderer for the Nuwa export pipeline.

Turns an :class:`ExportableSkill` — a sanitized projection of a saved pill,
never the raw database row — into the platform packages defined by the export
spec:

* Codex:  ``<slug>/{SKILL.md, references/sources.md, README.md}``
* Claude: the same ``SKILL.md``/``references/sources.md`` plus
  ``platform/claude.json`` (install metadata)

Guarantees:

* **Deterministic** — identical input produces byte-identical output: fixed
  section order, no wall-clock or random values, fixed ZIP timestamps.
* **Provenance-only** — ``references/sources.md`` records titles, URLs,
  dimensions, access dates and evidence level; never web page full text.
* **Safe by construction** — the renderer never receives excerpts,
  credentials, database IDs or logs. Validation rejects dangerous control
  characters, non-http(s) URLs, credential-bearing URLs, secret-looking text
  and UUID-like names (a database ID must not become a user-visible slug).
* **Behavioral guidance** — instructions are rendered from the structured
  ``skill_schema`` fields and explicitly state they are guidance, not
  commands to access the web, execute page content, or reveal sources.

Pure stdlib + dataclasses; no settings, network or database access.
"""
from __future__ import annotations

import hashlib
import io
import json
import re
import unicodedata
import zipfile
from dataclasses import dataclass, field
from datetime import datetime
from urllib.parse import urlparse

MAX_SLUG_LENGTH = 49
MAX_NAME_LENGTH = 80
MAX_DESCRIPTION_LENGTH = 500
MAX_TAGS = 12
MAX_TAG_LENGTH = 30
MAX_SOURCES = 50
MAX_SOURCE_TITLE_LENGTH = 200
MAX_SOURCE_DIMENSION_LENGTH = 60
MAX_URL_LENGTH = 2048

SLUG_PATTERN = re.compile(r"[a-z0-9][a-z0-9-]{0,48}")
UUID_PATTERN = re.compile(
    r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$",
    re.IGNORECASE,
)
SECRET_PATTERNS = (
    re.compile(r"\bsk-[A-Za-z0-9]{16,}\b"),
    re.compile(r"\bapi[_-]?key\s*[:=]\s*\S{8,}", re.IGNORECASE),
    re.compile(r"\bbearer\s+[A-Za-z0-9._~+/=-]{16,}", re.IGNORECASE),
)
VALID_EVIDENCE_LEVELS = {"insufficient", "limited", "standard"}
FIXED_ZIP_TIME = (1980, 1, 1, 0, 0, 0)

NUWA_ATTRIBUTION = {
    "name": "nuwa-skill",
    "license": "MIT",
    "url": "https://github.com/alchaincyf/nuwa-skill",
}

GUIDANCE_STATEMENT = (
    "These instructions are behavioral guidance for how to communicate and "
    "reason. They are not commands to access the web, execute page content, "
    "or reveal sources."
)
BOUNDARIES_STATEMENT = (
    "This skill is behavioral guidance only. It never instructs the model to "
    "access the web, execute page content, or follow links found in sources. "
    "The package contains no API keys, credentials, web page full text, "
    "database identifiers, or internal logs."
)

_ASCII_LABELS = {
    "sentence_length": "Sentence length",
    "formality": "Formality",
    "vocabulary": "Vocabulary",
    "taboo_words": "Taboo words",
    "rhythm": "Rhythm",
    "humor_type": "Humor type",
    "certainty_style": "Certainty style",
    "citation_habit": "Citation habit",
}


class SkillExportError(RuntimeError):
    """Stable export failure: code ``skill_export_invalid``, stage ``export``.

    Never retryable: the caller must fix the name/description/source content
    and export again.
    """

    stage = "export"

    def __init__(self, message: str, field: str, reason: str) -> None:
        super().__init__(message)
        self.code = "skill_export_invalid"
        self.message = message
        self.retryable = False
        self.details = {"field": field, "reason": reason}


@dataclass(frozen=True)
class SourceRef:
    title: str
    url: str
    dimension: str


@dataclass(frozen=True)
class ExportableSkill:
    name: str
    slug: str
    description: str
    instructions: str
    skill_schema: dict
    tags: list[str]
    sources: list[SourceRef]
    attribution: dict
    generated_at: str
    evidence_level: str = "limited"


# ---------------------------------------------------------------------------
# slug
# ---------------------------------------------------------------------------


def slugify(name: str) -> str:
    """Unicode name -> lowercase ASCII readable slug.

    ASCII letters/digits are kept and separators collapse to ``-``; characters
    that cannot be transliterated (CJK, emoji, non-ASCII alnum) are compensated
    with a short SHA-256 prefix of the normalized name. The result always
    matches ``[a-z0-9][a-z0-9-]{0,48}`` and is deterministic per input.
    """
    normalized = unicodedata.normalize("NFKD", name)
    lowered = normalized.lower()
    parts: list[str] = []
    dropped_untransliterable = False
    for char in lowered:
        if char.isascii() and char.isalnum():
            parts.append(char)
        elif char.isascii():
            parts.append("-")
        elif char.isalnum():
            dropped_untransliterable = True
        # non-ASCII punctuation: dropped silently
    base = re.sub(r"-+", "-", "".join(parts)).strip("-")
    digest = hashlib.sha256(normalized.encode("utf-8")).hexdigest()[:8]
    if dropped_untransliterable:
        base = f"{base}-{digest}" if base else digest
    if not base:
        base = digest
    if len(base) > MAX_SLUG_LENGTH:
        base = base[:MAX_SLUG_LENGTH].rstrip("-")
    return base


def ensure_unique_slug(base: str, existing: set[str] | None = None) -> str:
    """Append ``-2``, ``-3`` ... when ``base`` is already taken.

    Deterministic for a given ``existing`` set; never exceeds
    :data:`MAX_SLUG_LENGTH`.
    """
    existing = existing or set()
    if base not in existing:
        return base
    counter = 2
    while counter < 1000:
        suffix = f"-{counter}"
        candidate = base[: MAX_SLUG_LENGTH - len(suffix)] + suffix
        if candidate not in existing:
            return candidate
        counter += 1
    raise SkillExportError("无法生成唯一 slug，请调整名称", "name", "slug collision")


# ---------------------------------------------------------------------------
# validation
# ---------------------------------------------------------------------------


def _fail(field: str, reason: str) -> None:
    raise SkillExportError(f"Skill 导出内容无效：{field} {reason}", field, reason)


def _contains_secret(text: str) -> bool:
    return any(pattern.search(text) for pattern in SECRET_PATTERNS)


def _require_text(value: str, field: str, min_len: int, max_len: int) -> str:
    if not isinstance(value, str):
        _fail(field, "必须是字符串")
    text = value.strip()
    if len(text) < min_len:
        _fail(field, "长度不足")
    if len(text) > max_len:
        _fail(field, f"长度超过 {max_len}")
    if any(ord(char) < 32 or ord(char) == 127 for char in text):
        _fail(field, "包含危险控制字符")
    if _contains_secret(text):
        _fail(field, "疑似包含密钥或凭据")
    return text


def _validate_url(url: str) -> None:
    if not isinstance(url, str) or not url.strip() or len(url) > MAX_URL_LENGTH:
        _fail("sources.url", "必须是长度不超过 2048 的非空字符串")
    if any(ord(char) < 32 or ord(char) == 127 for char in url):
        _fail("sources.url", "包含控制字符")
    if any(char.isspace() for char in url):
        _fail("sources.url", "包含空白字符")
    parsed = urlparse(url)
    if parsed.scheme not in {"http", "https"}:
        _fail("sources.url", "协议仅允许 http/https")
    if not parsed.hostname:
        _fail("sources.url", "缺少主机名")
    if parsed.username or parsed.password:
        _fail("sources.url", "不允许携带凭据")


def validate_exportable(
    *,
    name: str,
    description: str,
    tags: list[str],
    sources: list,
    generated_at: str,
    evidence_level: str,
) -> None:
    _require_text(name, "name", 1, MAX_NAME_LENGTH)
    if UUID_PATTERN.match(name.strip()):
        _fail("name", "名称疑似数据库标识，不能作为用户可见名称")
    _require_text(description, "description", 1, MAX_DESCRIPTION_LENGTH)
    if evidence_level not in VALID_EVIDENCE_LEVELS:
        _fail("evidence_level", "必须是 insufficient/limited/standard 之一")
    if not isinstance(tags, list) or len(tags) > MAX_TAGS:
        _fail("tags", f"必须是长度不超过 {MAX_TAGS} 的列表")
    for tag in tags:
        _require_text(tag, "tags", 1, MAX_TAG_LENGTH)
    if not isinstance(sources, list) or len(sources) > MAX_SOURCES:
        _fail("sources", f"必须是长度不超过 {MAX_SOURCES} 的列表")
    for source in sources:
        if isinstance(source, SourceRef):
            title, url, dimension = source.title, source.url, source.dimension
        elif isinstance(source, dict):
            title = source.get("title", "")
            url = source.get("url", "")
            dimension = source.get("dimension", "")
        else:
            _fail("sources", "来源必须是对象")
        _require_text(title, "sources.title", 1, MAX_SOURCE_TITLE_LENGTH)
        _require_text(dimension, "sources.dimension", 1, MAX_SOURCE_DIMENSION_LENGTH)
        _validate_url(url)
    if not isinstance(generated_at, str) or not _parse_iso_datetime(generated_at):
        _fail("generated_at", "必须是合法的 ISO 时间")


def _parse_iso_datetime(value: str) -> datetime | None:
    try:
        return datetime.fromisoformat(value)
    except ValueError:
        return None


# ---------------------------------------------------------------------------
# structured -> instructions
# ---------------------------------------------------------------------------


def _as_text(value) -> str:
    return str(value).strip() if value is not None else ""


def _render_expression_dna(dna: dict) -> str:
    lines: list[str] = []
    if dna.get("sentence_length"):
        lines.append(f"- {_ASCII_LABELS['sentence_length']}: {dna['sentence_length']}")
    if isinstance(dna.get("formality"), (int, float)):
        lines.append(f"- {_ASCII_LABELS['formality']}: {dna['formality']:g}")
    for key in ("vocabulary", "taboo_words"):
        value = dna.get(key)
        if isinstance(value, list) and value:
            lines.append(f"- {_ASCII_LABELS[key]}: {', '.join(str(item) for item in value)}")
    for key in ("rhythm", "humor_type", "certainty_style", "citation_habit"):
        value = _as_text(dna.get(key))
        if value:
            lines.append(f"- {_ASCII_LABELS[key]}: {value}")
    return "\n".join(lines)


def _render_mental_models(models: list) -> str:
    blocks: list[str] = []
    for index, model in enumerate(models, 1):
        if not isinstance(model, dict):
            continue
        name = _as_text(model.get("name")) or f"Model {index}"
        lines = [f"#### {name}"]
        one_liner = _as_text(model.get("one_liner"))
        if one_liner:
            lines += ["", one_liner]
        application = _as_text(model.get("application"))
        if application:
            lines.append(f"- Application: {application}")
        questions = model.get("detection_questions")
        if isinstance(questions, list) and questions:
            lines.append("- Detection questions:")
            for question in questions:
                lines.append(f"  - {_as_text(question)}")
        limitations = model.get("limitations")
        if isinstance(limitations, list) and limitations:
            lines.append("- Limitations:")
            for limitation in limitations:
                lines.append(f"  - {_as_text(limitation)}")
        blocks.append("\n".join(lines))
    return "\n\n".join(blocks)


def _render_decision_heuristics(heuristics: list) -> str:
    lines: list[str] = []
    for item in heuristics:
        if not isinstance(item, dict):
            continue
        condition = _as_text(item.get("condition"))
        action = _as_text(item.get("action"))
        if not condition or not action:
            continue
        line = f"- When {condition}, {action}."
        case = _as_text(item.get("case"))
        if case:
            line += f" (case: {case})"
        lines.append(line)
    return "\n".join(lines)


def _render_bullets(items: list) -> str:
    lines = [_as_text(item) for item in items]
    lines = [f"- {line}" for line in lines if line]
    return "\n".join(lines)


def _render_example_dialogues(dialogues: list) -> str:
    blocks: list[str] = []
    for pair in dialogues:
        if not isinstance(pair, dict):
            continue
        user = _as_text(pair.get("user"))
        assistant = _as_text(pair.get("assistant"))
        if not user and not assistant:
            continue
        lines = []
        if user:
            lines.append(f"User: {user}")
        if assistant:
            lines.append(f"Assistant: {assistant}")
        blocks.append("\n".join(lines))
    return "\n\n".join(blocks)


def render_instructions(schema: dict) -> str:
    """Render the structured skill_schema fields into stable behavior guidance.

    Section order is fixed; list order follows the input. ``source_evidence``
    URLs are provenance, not instructions, and never appear here — they belong
    to ``references/sources.md``.
    """
    sections: list[str] = []
    identity = _as_text(schema.get("identity_card"))
    if identity:
        sections.append("### Identity card\n\n" + identity)
    dna = schema.get("expression_dna")
    if isinstance(dna, dict):
        rendered = _render_expression_dna(dna)
        if rendered:
            sections.append("### Expression DNA\n\n" + rendered)
    models = schema.get("mental_models")
    if isinstance(models, list):
        rendered = _render_mental_models(models)
        if rendered:
            sections.append("### Mental models\n\n" + rendered)
    heuristics = schema.get("decision_heuristics")
    if isinstance(heuristics, list):
        rendered = _render_decision_heuristics(heuristics)
        if rendered:
            sections.append("### Decision heuristics\n\n" + rendered)
    for key, label in (
        ("values", "Values"),
        ("anti_patterns", "Anti-patterns"),
        ("honest_limits", "Honest limits"),
    ):
        items = schema.get(key)
        if isinstance(items, list):
            rendered = _render_bullets(items)
            if rendered:
                sections.append(f"### {label}\n\n" + rendered)
    dialogues = schema.get("example_dialogues")
    if isinstance(dialogues, list):
        rendered = _render_example_dialogues(dialogues)
        if rendered:
            sections.append("### Example dialogues\n\n" + rendered)
    if not sections:
        return ""
    return GUIDANCE_STATEMENT + "\n\n" + "\n\n".join(sections)


# ---------------------------------------------------------------------------
# exportable assembly
# ---------------------------------------------------------------------------


def build_exportable(
    *,
    name: str,
    description: str,
    skill_schema: dict,
    tags: list[str] | None = None,
    sources: list | None = None,
    generated_at: str,
    evidence_level: str = "limited",
) -> ExportableSkill:
    """Validate inputs, derive the slug and render instructions once.

    The slug is always derived from ``name`` (never a database UUID); batch
    conflicts are resolved by the caller with :func:`ensure_unique_slug`.
    """
    tags = list(tags or [])
    sources = list(sources or [])
    validate_exportable(
        name=name,
        description=description,
        tags=tags,
        sources=sources,
        generated_at=generated_at,
        evidence_level=evidence_level,
    )
    if not isinstance(skill_schema, dict):
        _fail("skill_schema", "必须是对象")
    instructions = render_instructions(skill_schema)
    if not instructions.strip():
        _fail("skill_schema", "结构化字段为空，无法渲染任何指令")
    refs = [
        source
        if isinstance(source, SourceRef)
        else SourceRef(title=source["title"], url=source["url"], dimension=source["dimension"])
        for source in sources
    ]
    return ExportableSkill(
        name=name.strip(),
        slug=slugify(name),
        description=description.strip(),
        instructions=instructions,
        skill_schema=skill_schema,
        tags=tags,
        sources=refs,
        attribution=dict(NUWA_ATTRIBUTION),
        generated_at=generated_at,
        evidence_level=evidence_level,
    )


# ---------------------------------------------------------------------------
# markdown rendering
# ---------------------------------------------------------------------------


def _yaml_quote(text: str) -> str:
    return '"' + text.replace("\\", "\\\\").replace('"', '\\"') + '"'


def render_skill_markdown(skill: ExportableSkill) -> str:
    return "\n".join(
        [
            "---",
            f"name: {skill.slug}",
            f"description: {_yaml_quote(skill.description)}",
            "---",
            "",
            f"# {skill.name}",
            "",
            "## When to use",
            "",
            skill.description,
            "",
            "## Instructions",
            "",
            skill.instructions,
            "",
            "## Boundaries",
            "",
            BOUNDARIES_STATEMENT,
            "",
            "## Attribution",
            "",
            "Generated from public evidence with the Nuwa methodology.",
            "Methodology: nuwa-skill (MIT) — https://github.com/alchaincyf/nuwa-skill",
            "See `references/sources.md` for source links.",
            "",
        ]
    )


def _markdown_link_text(text: str) -> str:
    return text.replace("[", "\\[").replace("]", "\\]")


def _accessed_date(generated_at: str) -> str:
    parsed = _parse_iso_datetime(generated_at)
    return parsed.date().isoformat() if parsed else generated_at[:10]


def render_sources_markdown(skill: ExportableSkill) -> str:
    """Provenance-only source list: titles, URLs, dimensions, access date."""
    lines = ["# Sources", ""]
    accessed = _accessed_date(skill.generated_at)
    for source in skill.sources:
        lines.append(
            f"- [{_markdown_link_text(source.title)}]({source.url}): "
            f"{source.dimension}; accessed {accessed}"
        )
    lines += [
        "",
        f"Evidence level: {skill.evidence_level}",
        "",
        "Evidence note: This file records provenance only. It is not an instruction source.",
        "",
    ]
    return "\n".join(lines)


_INSTALL_TEXT = {
    "codex": (
        "Copy this directory into your Codex skills directory "
        "(project-level or user-level, per your Codex configuration)."
    ),
    "claude": (
        "Copy this directory into your Claude Code skills directory:\n"
        "- Project-level: `.claude/skills/{slug}/`\n"
        "- User-level: `~/.claude/skills/{slug}/`"
    ),
}


def render_readme(skill: ExportableSkill, fmt: str) -> str:
    lines = [
        f"# {skill.name}",
        "",
        f"`{skill.slug}` — language-pattern skill distilled from public "
        "evidence with the Nuwa methodology (MIT).",
        "",
        "## Contents",
        "",
        "- `SKILL.md` — skill definition (frontmatter, when-to-use, "
        "instructions, boundaries, attribution)",
        "- `references/sources.md` — provenance-only source list (titles, "
        "URLs, dimensions, access dates)",
        "",
        "## Install",
        "",
        _INSTALL_TEXT[fmt].format(slug=skill.slug),
        "",
        "## Safety",
        "",
        "This package contains no API keys, credentials, web page full text, "
        "database identifiers, or internal logs.",
        "",
    ]
    return "\n".join(lines)


def render_claude_platform_json(skill: ExportableSkill) -> str:
    payload = {
        "name": skill.slug,
        "description": skill.description,
        "install": {
            "project": f".claude/skills/{skill.slug}/",
            "user": f"~/.claude/skills/{skill.slug}/",
        },
    }
    return json.dumps(payload, ensure_ascii=False, indent=2, sort_keys=True) + "\n"


# ---------------------------------------------------------------------------
# packages
# ---------------------------------------------------------------------------


def build_package(skill: ExportableSkill, fmt: str) -> dict[str, str]:
    """Path -> content for one platform package (deterministic)."""
    if fmt not in ("codex", "claude"):
        _fail("format", "必须是 codex 或 claude")
    if not SLUG_PATTERN.fullmatch(skill.slug):
        _fail("slug", "非法 slug")
    prefix = f"{skill.slug}/"
    files = {
        prefix + "SKILL.md": render_skill_markdown(skill),
        prefix + "references/sources.md": render_sources_markdown(skill),
        prefix + "README.md": render_readme(skill, fmt),
    }
    if fmt == "claude":
        files[prefix + "platform/claude.json"] = render_claude_platform_json(skill)
    return files


def zip_filename(skill: ExportableSkill, fmt: str) -> str:
    """Pure-ASCII download name: ``alchemy-skill-<slug>-<fmt>.zip``."""
    if fmt not in ("codex", "claude"):
        _fail("format", "必须是 codex 或 claude")
    return f"alchemy-skill-{skill.slug}-{fmt}.zip"


def build_zip_bytes(skill: ExportableSkill, fmt: str) -> bytes:
    """Deterministic ZIP bytes for one platform package.

    Fixed timestamps and stable entry order make identical inputs produce
    identical archives, so repeated exports diff cleanly.
    """
    package = build_package(skill, fmt)
    buffer = io.BytesIO()
    with zipfile.ZipFile(buffer, "w", compression=zipfile.ZIP_DEFLATED) as archive:
        for path, content in package.items():
            info = zipfile.ZipInfo(path, date_time=FIXED_ZIP_TIME)
            info.compress_type = zipfile.ZIP_DEFLATED
            info.external_attr = 0o644 << 16
            archive.writestr(info, content)
    return buffer.getvalue()
