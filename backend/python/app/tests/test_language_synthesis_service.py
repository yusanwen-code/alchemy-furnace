# -*- coding: utf-8 -*-
"""
LanguageSynthesisService 单元测试 (T059)

覆盖：
1. 单金丹：结构化合并降级提示词、自洽金丹无 inner_tensions、指纹 sha256 前缀与确定性
2. 双相冲金丹（文言文 vs 嘻哈）：inner_tensions 冲突检测、加权 blending、
   LLM 涌现推导路径（OpenAI 调用全程 mock，无真实 API）
3. 指纹稳定性：相同 sort_order 语义下乱序输入指纹一致；改变权重指纹改变
4. 边界：空金丹列表仅性格提示词；mental_models 上限 20、example_dialogues 上限 10

运行：cd backend/python && python3 -m pytest app/tests/ -q
（第三方依赖 openai/httpx/pydantic 由 conftest.py 打桩）
"""
import json
from types import SimpleNamespace

import pytest

from app.core.config import settings
from app.services.language_synthesis_service import LanguageSynthesisService


# ==================== 指纹跨端一致性金标准 (contracts/python-synthesis.md) ====================
# 固定样本：Go 端 T043 须以同一 personality + pills 计算出同一 SHA256。
# 排序键：(sort_order, str(id)) 字典序；序列化：json.dumps(ensure_ascii=False, sort_keys=True)。
_GOLDEN_PERSONALITY = "测试道人"
_UUID_A = "11111111-1111-1111-1111-111111111111"  # sort_order 0
_UUID_B = "22222222-2222-2222-2222-222222222222"  # sort_order 1
_UUID_C = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"  # sort_order 1
_GOLDEN_FINGERPRINT = "sha256:c8fecc755f7d242c6e6871a1726c14110975125931eff8a9771d2a620f98f95a"


# ==================== 测试夹具 ====================


def _wenyan_pill(**overrides) -> dict:
    """文言文金丹：正式度 0.9，长句"""
    pill = {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "文言文金丹",
        "weight": 1.0,
        "sort_order": 0,
        "skill_schema": {
            "identity_card": "吾乃古文大家，出口成章。",
            "description": "以文言笔法应答",
            "expression_dna": {
                "sentence_length": "long",
                "formality": 0.9,
                "vocabulary": ["之乎者也", "盖", "夫"],
                "taboo_words": ["yyds"],
                "rhythm": "平仄相间",
                "certainty_style": "引经据典式断言",
            },
            "mental_models": [
                {"name": "起承转合", "one_liner": "行文四段式"},
            ],
            "decision_heuristics": [
                {"condition": "论事", "action": "先引古训", "case": "论学"},
            ],
            "values": ["典雅"],
            "anti_patterns": ["网络烂梗"],
            "honest_limits": ["不通白话俚语"],
            "example_dialogues": [
                {"user": "何谓道？", "assistant": "道者，万物之奥也。"},
            ],
        },
    }
    pill.update(overrides)
    return pill


def _hiphop_pill(**overrides) -> dict:
    """嘻哈金丹：正式度 0.2，短句"""
    pill = {
        "id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
        "name": "嘻哈金丹",
        "weight": 1.0,
        "sort_order": 1,
        "skill_schema": {
            "identity_card": "Yo，我是街头说唱手。",
            "description": "用嘻哈腔调聊天",
            "expression_dna": {
                "sentence_length": "short",
                "formality": 0.2,
                "vocabulary": ["yo", "skr", "yyds"],
                "taboo_words": ["之乎者也"],
                "rhythm": "押韵flow",
                "certainty_style": "夸张自信",
            },
            "mental_models": [
                {"name": "Freestyle", "one_liner": "即兴为上"},
            ],
            "decision_heuristics": [
                {"condition": "被质疑", "action": "用押韵回击", "case": "battle"},
            ],
            "values": ["真实"],
            "anti_patterns": ["掉书袋"],
            "honest_limits": ["不擅正式文书"],
            "example_dialogues": [
                {"user": "你会rap吗", "assistant": "Yo，张口就来不带停。"},
            ],
        },
    }
    pill.update(overrides)
    return pill


@pytest.fixture()
def service():
    """未配置 API key 的服务实例（走降级路径）"""
    return LanguageSynthesisService(api_key="", base_url="http://stub")


@pytest.fixture()
def llm_service(monkeypatch):
    """配置了伪 API key 的服务实例，LLM 调用被 mock"""
    monkeypatch.setattr(settings, "openai_api_key", "sk-test-key")
    svc = LanguageSynthesisService(api_key="sk-test-key", base_url="http://stub")

    captured = {}

    def fake_create(**kwargs):
        captured["kwargs"] = kwargs
        content = json.dumps(
            {
                "system_prompt": "你是一位文白相间、雅俗共赏的 AI 道人，正式场合用文言，闲聊时来段 flow。",
                "emergence_rules": [
                    "文言丹性与嘻哈丹性相互作用：按场景正式度切换文白比例",
                    "双丹合璧：押韵时可化用典故，掉书袋时须带节奏感",
                ],
            },
            ensure_ascii=False,
        )
        message = SimpleNamespace(content=content)
        choice = SimpleNamespace(message=message)
        usage = SimpleNamespace(
            prompt_tokens=800, completion_tokens=200, total_tokens=1000
        )
        return SimpleNamespace(choices=[choice], usage=usage)

    monkeypatch.setattr(svc.client.chat.completions, "create", fake_create)
    return svc, captured


# ==================== 1. 单金丹 ====================


class TestSinglePill:
    def test_fallback_prompt_contains_personality_and_dna(self, service):
        result = service.combine(
            personality="沉稳内敛，喜好引经据典",
            pills=[_wenyan_pill()],
        )
        prompt = result["system_prompt"]
        assert "沉稳内敛，喜好引经据典" in prompt
        # formality=0.9 加权平均即 0.9
        assert "0.9" in prompt
        assert "long" in prompt  # 句式偏好写入降级提示词

    def test_self_consistent_pill_has_no_inner_tensions(self, service):
        result = service.combine(
            personality="沉稳内敛",
            pills=[_wenyan_pill()],
        )
        assert result["inner_tensions"] == []

    def test_fingerprint_prefix_and_determinism(self, service):
        pills = [_wenyan_pill()]
        r1 = service.combine(personality="沉稳内敛", pills=pills)
        r2 = service.combine(personality="沉稳内敛", pills=pills)
        assert r1["fingerprint"].startswith("sha256:")
        assert len(r1["fingerprint"]) == len("sha256:") + 64
        assert r1["fingerprint"] == r2["fingerprint"]
        # 与静态方法结果一致
        assert r1["fingerprint"] == LanguageSynthesisService.compute_fingerprint(
            "沉稳内敛", pills
        )

    def test_emergence_rules_empty_on_fallback(self, service):
        result = service.combine(personality="沉稳内敛", pills=[_wenyan_pill()])
        assert result["emergence_rules"] == []
        assert result["usage"] == {}


# ==================== 2. 双相冲金丹 ====================


class TestTwoConflictingPills:
    def test_inner_tensions_detected_with_valid_severity(self, llm_service):
        svc, _ = llm_service
        result = svc.combine(
            personality="沉稳内敛",
            pills=[_wenyan_pill(), _hiphop_pill()],
        )
        tensions = result["inner_tensions"]
        assert len(tensions) >= 1
        for t in tensions:
            assert t["severity"] in ("low", "medium", "high")
            assert t["dimension"]
            assert t["description"]
        dimensions = {t["dimension"] for t in tensions}
        # 正式度极差 0.7 (>0.4) 必报；句式 long vs short 必报
        assert "formality" in dimensions
        assert "sentence_length" in dimensions
        # 正式度极差 0.7 达到 high 阈值
        formality_t = next(t for t in tensions if t["dimension"] == "formality")
        assert formality_t["severity"] == "high"
        # 高频词 yyds 被文言文金丹列为禁忌（跨丹相冲）
        assert "taboo_words" in dimensions

    def test_weighted_blend_in_merged_dna(self, llm_service):
        svc, _ = llm_service
        pills = [
            _wenyan_pill(weight=3.0),
            _hiphop_pill(weight=1.0),
        ]
        merged = svc._structured_merge("沉稳内敛", pills)
        # (0.9*3 + 0.2*1) / 4 = 0.725
        assert merged["expression_dna"]["formality"] == pytest.approx(0.725)
        # 句式取权重最高者（文言文 long）
        assert merged["expression_dna"]["sentence_length"] == "long"
        # 词汇按权重降序合并去重：文言文词在前
        vocab = merged["expression_dna"]["vocabulary"]
        assert vocab[:3] == ["之乎者也", "盖", "夫"]
        assert "yo" in vocab

    def test_emergence_derivation_with_mocked_llm(self, llm_service):
        svc, captured = llm_service
        result = svc.combine(
            personality="沉稳内敛",
            pills=[_wenyan_pill(), _hiphop_pill()],
            model="gpt-4o-mini",
        )
        # LLM 确实被调用（非降级路径）
        assert "kwargs" in captured
        call = captured["kwargs"]
        assert call["model"] == "gpt-4o-mini"
        user_prompt = call["messages"][1]["content"]
        # 合并后的加权 formality 0.55 与冲突信息进入提示词
        assert "0.55" in user_prompt
        assert "内在冲突" in user_prompt
        # 响应字段按契约返回
        assert "雅俗共赏" in result["system_prompt"]
        assert len(result["emergence_rules"]) == 2
        assert result["usage"]["total_tokens"] == 1000
        assert result["model"] == "gpt-4o-mini"

    def test_emergence_llm_failure_falls_back(self, service, monkeypatch):
        """LLM 抛异常时降级为结构化提示词且不抛出"""
        monkeypatch.setattr(settings, "openai_api_key", "sk-test-key")

        def boom(**kwargs):
            raise RuntimeError("network down")

        monkeypatch.setattr(service.client.chat.completions, "create", boom)
        result = service.combine(
            personality="沉稳内敛",
            pills=[_wenyan_pill(), _hiphop_pill()],
        )
        assert "沉稳内敛" in result["system_prompt"]
        assert result["emergence_rules"] == []
        assert result["inner_tensions"]  # 冲突检测不受 LLM 失败影响


# ==================== 3. 指纹稳定性 ====================


class TestFingerprint:
    def test_input_order_irrelevant_given_same_sort_order(self):
        """输入列表乱序但 sort_order 语义相同 -> 指纹一致"""
        a = _wenyan_pill()
        b = _hiphop_pill()
        f1 = LanguageSynthesisService.compute_fingerprint("沉稳内敛", [a, b])
        f2 = LanguageSynthesisService.compute_fingerprint("沉稳内敛", [b, a])
        assert f1 == f2

    def test_weight_change_changes_fingerprint(self):
        f1 = LanguageSynthesisService.compute_fingerprint(
            "沉稳内敛", [_wenyan_pill(weight=1.0)]
        )
        f2 = LanguageSynthesisService.compute_fingerprint(
            "沉稳内敛", [_wenyan_pill(weight=2.0)]
        )
        assert f1 != f2

    def test_personality_change_changes_fingerprint(self):
        f1 = LanguageSynthesisService.compute_fingerprint("沉稳内敛", [])
        f2 = LanguageSynthesisService.compute_fingerprint("活泼外向", [])
        assert f1 != f2

    # ---- UUID 契约对齐 (T042 / contracts/python-synthesis.md) ----

    def test_uuid_id_disorder_same_fingerprint(self):
        """乱序输入（含同 sort_order、id 字典序需重排）-> 指纹一致

        覆盖 contracts/python-synthesis.md 一致性用例：同一 pill 集合以不同
        顺序输入，(sort_order, str(id)) 字典序排序后归一为同一规范化 JSON
        -> 同一 SHA256。其中乙/丙同 sort_order=1，"2222..." 字典序先于
        "aaaa..."，故乱序输入 [丙, 甲, 乙] 必须重排为 [甲, 乙, 丙]。
        """
        ordered = [
            {"sort_order": 0, "id": _UUID_A, "name": "甲丹", "weight": 1.0, "skill_schema": {"k": "a"}},
            {"sort_order": 1, "id": _UUID_B, "name": "乙丹", "weight": 2.0, "skill_schema": {"k": "b"}},
            {"sort_order": 1, "id": _UUID_C, "name": "丙丹", "weight": 1.5, "skill_schema": {"k": "c"}},
        ]
        # 同集合、乱序：丙(sort1) 在前、甲(sort0) 居中、乙(sort1) 殿后
        disordered = [ordered[2], ordered[0], ordered[1]]
        f_ordered = LanguageSynthesisService.compute_fingerprint(_GOLDEN_PERSONALITY, ordered)
        f_disordered = LanguageSynthesisService.compute_fingerprint(_GOLDEN_PERSONALITY, disordered)
        assert f_ordered == f_disordered

    def test_fingerprint_golden_value_uuid_contract(self):
        """固定样本指纹金标准值（供 Go 端 T043 跨端一致性比对）

        排序后规范化 JSON（ensure_ascii=False, sort_keys=True）的 SHA256
        必须等于 _GOLDEN_FINGERPRINT；Go 端以同一输入计算应得到同一哈希。
        规范化 JSON：
        {"personality": "测试道人", "pills": [{"id": "11111111-...",
        "name": "甲丹", "skill_schema": {"k": "a"}, "sort_order": 0, "weight": 1.0},
        {"id": "22222222-...", "name": "乙丹", "skill_schema": {"k": "b"},
        "sort_order": 1, "weight": 2.0}, {"id": "aaaaaaaa-...", "name": "丙丹",
        "skill_schema": {"k": "c"}, "sort_order": 1, "weight": 1.5}]}
        """
        pills = [
            {"sort_order": 0, "id": _UUID_A, "name": "甲丹", "weight": 1.0, "skill_schema": {"k": "a"}},
            {"sort_order": 1, "id": _UUID_B, "name": "乙丹", "weight": 2.0, "skill_schema": {"k": "b"}},
            {"sort_order": 1, "id": _UUID_C, "name": "丙丹", "weight": 1.5, "skill_schema": {"k": "c"}},
        ]
        fp = LanguageSynthesisService.compute_fingerprint(_GOLDEN_PERSONALITY, pills)
        assert fp == _GOLDEN_FINGERPRINT


# ==================== 4. 边界与上限 ====================


class TestEdgeCases:
    def test_empty_pills_returns_personality_only_prompt(self, service):
        result = service.combine(personality="沉稳内敛，喜好引经据典", pills=[])
        assert "沉稳内敛，喜好引经据典" in result["system_prompt"]
        assert "正式程度" not in result["system_prompt"]
        assert result["inner_tensions"] == []
        assert result["fingerprint"].startswith("sha256:")

    def test_none_pills_treated_as_empty(self, service):
        result = service.combine(personality="沉稳内敛", pills=None)
        assert "沉稳内敛" in result["system_prompt"]

    def test_mental_models_capped_at_20(self, service):
        pill = _wenyan_pill()
        pill["skill_schema"]["mental_models"] = [
            {"name": f"模型{i:02d}", "one_liner": "x"} for i in range(25)
        ]
        merged = service._structured_merge("", [pill])
        assert len(merged["mental_models"]) == 20
        assert merged["mental_models"][0]["from_pill"] == "文言文金丹"

    def test_example_dialogues_capped_at_10(self, service):
        pill = _wenyan_pill()
        pill["skill_schema"]["example_dialogues"] = [
            {"user": f"问{i}", "assistant": f"答{i}"} for i in range(15)
        ]
        merged = service._structured_merge("", [pill])
        assert len(merged["example_dialogues"]) == 10

    def test_duplicate_mental_models_deduplicated(self, service):
        p1 = _wenyan_pill()
        p2 = _hiphop_pill()
        # 两颗丹含同名心智模型
        p2["skill_schema"]["mental_models"] = [
            {"name": "起承转合", "one_liner": "另一版本"},
        ]
        merged = service._structured_merge("", [p1, p2])
        names = [m["name"] for m in merged["mental_models"]]
        assert names.count("起承转合") == 1
