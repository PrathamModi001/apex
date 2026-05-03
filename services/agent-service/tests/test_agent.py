"""Unit tests for AgentUseCase.

All infra is mocked — no real Kafka/Postgres/Neo4j/Redis required.
"""
import json
import pytest
from datetime import datetime, timezone
from unittest.mock import AsyncMock, MagicMock, patch, call

from app.domain.invoice import Decision, ExtractedFields, POMatch, ProcessedInvoice
from app.domain.policy import CompiledRule, Policy, PolicyCondition
from app.app.agent.usecase import AgentUseCase, calculate_risk_score


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _make_invoice(**kwargs) -> ProcessedInvoice:
    defaults = dict(
        id="inv-001",
        source="email",
        file_key="invoices/inv-001.pdf",
        sha256="abc123",
        sender="vendor@example.com",
        received_at=datetime(2024, 1, 1, tzinfo=timezone.utc),
        extracted_fields=ExtractedFields(
            invoice_no="INV-001",
            amount=100.0,
            currency="USD",
            due_date="2024-02-01",
            vendor_name="Acme Corp",
        ),
        po_match=POMatch(po_id="PO-1", confidence=0.95, matched=True),
        processed_at=datetime(2024, 1, 1, tzinfo=timezone.utc),
    )
    defaults.update(kwargs)
    return ProcessedInvoice(**defaults)


def _make_mocks():
    llm = MagicMock()
    fraud_graph = MagicMock()
    audit_repo = MagicMock()
    invoice_repo = MagicMock()
    feedback_repo = MagicMock()
    publisher = MagicMock()
    policy_repo = MagicMock()

    # Sensible defaults
    fraud_graph.get_vendor_risk.return_value = {"invoice_count": 5, "risk_score": 10.0}
    fraud_graph.check_bank_account_sharing.return_value = False
    fraud_graph.get_betweenness_centrality.return_value = 5.0
    audit_repo.append.return_value = "a" * 64
    invoice_repo.get.return_value = _make_invoice()
    feedback_repo.get_similar_corrections.return_value = []
    llm.embed.return_value = [0.0] * 768
    policy_repo.get_active.return_value = []

    return llm, fraud_graph, audit_repo, invoice_repo, feedback_repo, publisher, policy_repo


def _make_usecase(**overrides):
    llm, fraud_graph, audit_repo, invoice_repo, feedback_repo, publisher, policy_repo = _make_mocks()
    # Apply overrides
    ns = {
        "llm": llm,
        "fraud_graph": fraud_graph,
        "audit_repo": audit_repo,
        "invoice_repo": invoice_repo,
        "feedback_repo": feedback_repo,
        "publisher": publisher,
        "policy_repo": policy_repo,
    }
    ns.update(overrides)
    uc = AgentUseCase(**ns)
    return uc, ns


# ---------------------------------------------------------------------------
# Test 1: Policy match skips LLM
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_policy_match_skips_llm():
    """If an active compiled policy matches the invoice, LLM should NOT be called."""
    llm = MagicMock()
    fraud_graph = MagicMock()
    audit_repo = MagicMock()
    invoice_repo = MagicMock()
    feedback_repo = MagicMock()
    publisher = MagicMock()
    policy_repo = MagicMock()

    audit_repo.append.return_value = "b" * 64
    invoice_repo.get.return_value = _make_invoice()
    llm.embed.return_value = [0.0] * 768

    # A policy that will match: amount >= 0 (always true for any positive amount)
    matching_policy = Policy(
        id="pol-1",
        raw_text="auto-approve all invoices",
        compiled_rule=CompiledRule(
            conditions=[PolicyCondition(field="amount", op="gte", value=0.0)],
            action="auto_approve",
            logic="AND",
        ),
        active=True,
    )
    policy_repo.get_active.return_value = [matching_policy]

    uc = AgentUseCase(
        llm=llm,
        fraud_graph=fraud_graph,
        audit_repo=audit_repo,
        invoice_repo=invoice_repo,
        feedback_repo=feedback_repo,
        publisher=publisher,
        policy_repo=policy_repo,
    )

    invoice = _make_invoice()
    decision = await uc.process(invoice)

    llm.chat.assert_not_called()
    assert decision.decision == "POLICY_MATCH"
    publisher.publish_decision.assert_called_once()


# ---------------------------------------------------------------------------
# Test 2: Happy path AUTO_APPROVE
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_happy_path_auto_approve():
    """Clean invoice with low risk → AUTO_APPROVE decision."""
    uc, ns = _make_usecase()
    llm = ns["llm"]

    # First call: give a tool use step
    # Second call: give Final Answer
    llm.chat.side_effect = [
        (
            "Thought: Invoice looks clean.\n"
            "Action: db_lookup\n"
            'Action Input: {"invoice_id": "inv-001"}\n'
        ),
        (
            "Thought: PO matched, low risk.\n"
            "Action: fraud_graph\n"
            'Action Input: {"vendor_name": "Acme Corp"}\n'
        ),
        'Final Answer: {"decision": "AUTO_APPROVE", "risk_score": 10, "reason": "Clean invoice"}',
    ]

    invoice = _make_invoice()
    decision = await uc.process(invoice)

    assert decision.decision == "AUTO_APPROVE"
    assert decision.risk_score == 10.0
    ns["publisher"].publish_decision.assert_called_once()


# ---------------------------------------------------------------------------
# Test 3: Flagged by risk
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_flagged_by_high_risk_score():
    """If Final Answer returns risk_score > 60 with AUTO_APPROVE, decision becomes FLAGGED."""
    uc, ns = _make_usecase()
    ns["llm"].chat.return_value = (
        'Final Answer: {"decision": "AUTO_APPROVE", "risk_score": 80, "reason": "suspicious amount"}'
    )

    invoice = _make_invoice()
    decision = await uc.process(invoice)

    # risk_score > 60 must override AUTO_APPROVE to FLAGGED
    assert decision.decision == "FLAGGED"
    assert decision.risk_score == 80.0


# ---------------------------------------------------------------------------
# Test 4: Rejected by fraud
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_rejected_by_fraud_graph():
    """When LLM detects shared bank account via fraud_graph tool → REJECTED."""
    uc, ns = _make_usecase()
    ns["llm"].chat.side_effect = [
        (
            "Thought: Checking fraud signals.\n"
            "Action: fraud_graph\n"
            'Action Input: {"vendor_name": "BadVendor"}\n'
        ),
        (
            "Thought: Shared bank account detected!\n"
            'Final Answer: {"decision": "REJECTED", "risk_score": 95, "reason": "shared bank account"}'
        ),
    ]
    ns["fraud_graph"].check_bank_account_sharing.return_value = True

    invoice = _make_invoice()
    decision = await uc.process(invoice)

    assert decision.decision == "REJECTED"
    assert decision.risk_score == 95.0


# ---------------------------------------------------------------------------
# Test 5: Max steps reached → FLAGGED default
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_max_steps_reached_defaults_to_flagged():
    """When LLM never returns Final Answer in 5 steps, default to FLAGGED."""
    uc, ns = _make_usecase()
    # Always return a tool step, never Final Answer
    ns["llm"].chat.return_value = (
        "Thought: Still analyzing.\n"
        "Action: db_lookup\n"
        'Action Input: {"invoice_id": "inv-001"}\n'
    )

    invoice = _make_invoice()
    decision = await uc.process(invoice)

    assert decision.decision == "FLAGGED"
    assert decision.risk_score == 50.0
    # LLM should be called at most MAX_STEPS times
    assert ns["llm"].chat.call_count <= 5


# ---------------------------------------------------------------------------
# Test 6: Few-shot injection
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_few_shot_injection_in_system_prompt():
    """Feedback corrections should appear in the system prompt sent to the LLM."""
    uc, ns = _make_usecase()
    ns["feedback_repo"].get_similar_corrections.return_value = [
        {
            "invoice_id": "old-inv",
            "agent_decision": "AUTO_APPROVE",
            "human_decision": "REJECTED",
            "reason": "duplicate payment",
            "invoice_summary": "Acme Corp $500",
        }
    ]
    ns["llm"].chat.return_value = (
        'Final Answer: {"decision": "FLAGGED", "risk_score": 40, "reason": "checking history"}'
    )

    invoice = _make_invoice()
    await uc.process(invoice)

    # Grab the messages passed in the first LLM call
    first_call_args = ns["llm"].chat.call_args_list[0]
    messages = first_call_args[0][0]  # positional arg
    system_content = messages[0]["content"]

    assert "duplicate payment" in system_content or "human_decision" in system_content or "REJECTED" in system_content


# ---------------------------------------------------------------------------
# Test 7: Audit entries written per step
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_audit_entries_written_for_each_step():
    """Each ReAct step should write an audit entry."""
    uc, ns = _make_usecase()
    ns["llm"].chat.side_effect = [
        (
            "Thought: Step 1.\n"
            "Action: db_lookup\n"
            'Action Input: {"invoice_id": "inv-001"}\n'
        ),
        (
            "Thought: Step 2.\n"
            "Action: fraud_graph\n"
            'Action Input: {"vendor_name": "Acme Corp"}\n'
        ),
        'Final Answer: {"decision": "AUTO_APPROVE", "risk_score": 5, "reason": "ok"}',
    ]

    invoice = _make_invoice()
    await uc.process(invoice)

    # Should have at least 3 audit.append calls: 2 steps + 1 final
    assert ns["audit_repo"].append.call_count >= 3


# ---------------------------------------------------------------------------
# Test 8: Decision published exactly once
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_decision_published_once():
    """publisher.publish_decision must be called exactly once per invoice."""
    uc, ns = _make_usecase()
    ns["llm"].chat.return_value = (
        'Final Answer: {"decision": "AUTO_APPROVE", "risk_score": 20, "reason": "clean"}'
    )

    invoice = _make_invoice()
    await uc.process(invoice)

    ns["publisher"].publish_decision.assert_called_once()
    call_arg = ns["publisher"].publish_decision.call_args[0][0]
    assert isinstance(call_arg, Decision)
    assert call_arg.invoice_id == invoice.id
