"""Unit tests for PolicyUseCase (evaluate + compile)."""
import pytest
from unittest.mock import MagicMock

from app.domain.invoice import ExtractedFields, POMatch, ProcessedInvoice
from app.domain.policy import CompiledRule, Policy, PolicyCondition
from app.app.policy.usecase import PolicyUseCase
from datetime import datetime, timezone


def _make_invoice(amount=100.0, currency="USD", matched=True, confidence=0.9) -> ProcessedInvoice:
    return ProcessedInvoice(
        id="inv-test",
        source="email",
        file_key="test.pdf",
        sha256="deadbeef",
        sender="v@example.com",
        received_at=datetime(2024, 1, 1, tzinfo=timezone.utc),
        extracted_fields=ExtractedFields(
            invoice_no="T-001",
            amount=amount,
            currency=currency,
            due_date="2024-06-01",
            vendor_name="TestVendor",
        ),
        po_match=POMatch(po_id="PO-T", confidence=confidence, matched=matched),
        processed_at=datetime(2024, 1, 1, tzinfo=timezone.utc),
    )


def _make_policy(conditions, action="auto_approve", logic="AND") -> Policy:
    return Policy(
        id="pol-test",
        raw_text="test policy",
        compiled_rule=CompiledRule(
            conditions=[PolicyCondition(**c) for c in conditions],
            action=action,
            logic=logic,
        ),
        active=True,
    )


# ---------------------------------------------------------------------------
# Test 1: AND logic — all conditions true → match
# ---------------------------------------------------------------------------

def test_and_logic_all_true():
    uc = PolicyUseCase(llm=MagicMock())
    invoice = _make_invoice(amount=200.0, matched=True)
    policy = _make_policy(
        conditions=[
            {"field": "amount", "op": "lt", "value": 500.0},
            {"field": "vendor_approved", "op": "eq", "value": True},
        ],
        logic="AND",
    )
    assert uc.evaluate(policy, invoice) is True


# ---------------------------------------------------------------------------
# Test 2: AND logic — one condition false → no match
# ---------------------------------------------------------------------------

def test_and_logic_one_false():
    uc = PolicyUseCase(llm=MagicMock())
    invoice = _make_invoice(amount=600.0, matched=True)
    policy = _make_policy(
        conditions=[
            {"field": "amount", "op": "lt", "value": 500.0},  # FALSE: 600 < 500
            {"field": "vendor_approved", "op": "eq", "value": True},  # TRUE
        ],
        logic="AND",
    )
    assert uc.evaluate(policy, invoice) is False


# ---------------------------------------------------------------------------
# Test 3: OR logic — any condition true → match
# ---------------------------------------------------------------------------

def test_or_logic_any_true():
    uc = PolicyUseCase(llm=MagicMock())
    invoice = _make_invoice(amount=600.0, matched=False)
    policy = _make_policy(
        conditions=[
            {"field": "amount", "op": "lt", "value": 500.0},   # FALSE: 600 < 500
            {"field": "vendor_approved", "op": "eq", "value": False},  # TRUE: not matched
        ],
        logic="OR",
    )
    assert uc.evaluate(policy, invoice) is True


# ---------------------------------------------------------------------------
# Test 4: Unknown field → False (no crash)
# ---------------------------------------------------------------------------

def test_unknown_field_returns_false():
    uc = PolicyUseCase(llm=MagicMock())
    invoice = _make_invoice()
    policy = _make_policy(
        conditions=[{"field": "nonexistent_field", "op": "eq", "value": 42}],
        logic="AND",
    )
    assert uc.evaluate(policy, invoice) is False


# ---------------------------------------------------------------------------
# Test 5: Compile valid JSON → CompiledRule parsed correctly
# ---------------------------------------------------------------------------

def test_compile_valid_json():
    llm = MagicMock()
    llm.chat.return_value = (
        '{"conditions": [{"field": "amount", "op": "lt", "value": 500}], '
        '"action": "auto_approve", "logic": "AND"}'
    )
    uc = PolicyUseCase(llm=llm)
    rule = uc.compile("auto-approve if amount < 500")
    assert rule.action == "auto_approve"
    assert rule.logic == "AND"
    assert len(rule.conditions) == 1
    assert rule.conditions[0].field == "amount"
    assert rule.conditions[0].op == "lt"
    assert rule.conditions[0].value == 500


# ---------------------------------------------------------------------------
# Test 6: Compile invalid JSON → raises ValueError
# ---------------------------------------------------------------------------

def test_compile_invalid_json_raises():
    llm = MagicMock()
    llm.chat.return_value = "This is not JSON at all!!"
    uc = PolicyUseCase(llm=llm)
    with pytest.raises(ValueError):
        uc.compile("some policy rule")


# ---------------------------------------------------------------------------
# Additional operator coverage
# ---------------------------------------------------------------------------

def test_all_operators():
    uc = PolicyUseCase(llm=MagicMock())

    cases = [
        ({"field": "amount", "op": "gt", "value": 50.0}, 100.0, True),
        ({"field": "amount", "op": "lte", "value": 100.0}, 100.0, True),
        ({"field": "amount", "op": "gte", "value": 100.0}, 100.0, True),
        ({"field": "amount", "op": "neq", "value": 99.0}, 100.0, True),
        ({"field": "amount", "op": "eq", "value": 100.0}, 100.0, True),
        ({"field": "amount", "op": "eq", "value": 200.0}, 100.0, False),
    ]

    for cond, amount, expected in cases:
        invoice = _make_invoice(amount=amount)
        policy = _make_policy(conditions=[cond], logic="AND")
        result = uc.evaluate(policy, invoice)
        assert result is expected, f"Failed for op={cond['op']}: expected {expected}"
