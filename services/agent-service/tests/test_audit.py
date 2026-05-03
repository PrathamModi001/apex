"""Unit tests for Merkle audit chain logic."""
import hashlib
import json
import pytest

from app.infra.postgres.audit_repo import AuditRepository, _chain_hash


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _expected_hash(prev: str, payload: dict) -> str:
    data = prev + json.dumps(payload, sort_keys=True)
    return hashlib.sha256(data.encode()).hexdigest()


# ---------------------------------------------------------------------------
# Test 1: Chain hash formula correct
# ---------------------------------------------------------------------------

def test_chain_hash_formula():
    prev = "0" * 64
    payload = {"event": "TEST", "value": 42}
    expected = _expected_hash(prev, payload)
    assert _chain_hash(prev, payload) == expected


# ---------------------------------------------------------------------------
# Test 2: First entry uses prev_hash = "0"*64
# ---------------------------------------------------------------------------

def test_first_entry_prev_hash_is_zeros():
    """The first audit entry for an invoice must use prev_hash = '0'*64."""
    repo = AuditRepository(db_url="")  # force in-memory
    payload = {"step": 0, "action": "db_lookup"}
    h = repo.append("inv-001", "REACT_STEP", "agent", payload)

    assert len(repo._memory) == 1
    entry = repo._memory[0]
    assert entry["prev_hash"] == "0" * 64
    assert entry["chain_hash"] == _expected_hash("0" * 64, payload)
    assert h == entry["chain_hash"]


# ---------------------------------------------------------------------------
# Test 3: Chain integrity over 5 entries
# ---------------------------------------------------------------------------

def test_chain_integrity_five_entries():
    """After 5 appends, recomputing the chain from scratch must match stored hashes."""
    repo = AuditRepository(db_url="")
    invoice_id = "inv-chain"
    payloads = [{"step": i, "data": f"event-{i}"} for i in range(5)]

    for p in payloads:
        repo.append(invoice_id, "REACT_STEP", "agent", p)

    chain = repo.get_chain(invoice_id)
    assert len(chain) == 5

    # Recompute from scratch
    prev = "0" * 64
    for i, entry in enumerate(chain):
        expected = _expected_hash(prev, payloads[i])
        assert entry["chain_hash"] == expected, f"Chain broken at step {i}"
        assert entry["prev_hash"] == prev
        prev = entry["chain_hash"]


# ---------------------------------------------------------------------------
# Test 4: Tamper detection — modifying middle entry breaks recomputation
# ---------------------------------------------------------------------------

def test_tamper_detection():
    """Modifying the payload of an entry should cause hash recomputation to fail."""
    repo = AuditRepository(db_url="")
    invoice_id = "inv-tamper"
    payloads = [{"step": i, "data": f"event-{i}"} for i in range(5)]

    for p in payloads:
        repo.append(invoice_id, "REACT_STEP", "agent", p)

    # Tamper with the middle entry's payload
    entries = [e for e in repo._memory if e["invoice_id"] == invoice_id]
    entries[2]["payload"] = {"step": 2, "data": "TAMPERED"}

    # Recompute: use recomputed hash as prev so tamper cascades
    prev = "0" * 64
    mismatches = 0
    for entry in entries:
        expected = _expected_hash(prev, entry["payload"])
        if entry["chain_hash"] != expected:
            mismatches += 1
        prev = expected  # cascade: recomputed value, not stored

    # The tampered entry and all subsequent entries should have mismatches
    assert mismatches >= 3  # entry 2, 3, 4 should fail


# ---------------------------------------------------------------------------
# Additional: Multiple invoices don't interfere
# ---------------------------------------------------------------------------

def test_multiple_invoices_independent_chains():
    repo = AuditRepository(db_url="")
    repo.append("inv-A", "REACT_STEP", "agent", {"data": "a1"})
    repo.append("inv-B", "REACT_STEP", "agent", {"data": "b1"})
    repo.append("inv-A", "REACT_STEP", "agent", {"data": "a2"})

    chain_a = repo.get_chain("inv-A")
    chain_b = repo.get_chain("inv-B")

    assert len(chain_a) == 2
    assert len(chain_b) == 1
    # Chain A: second entry's prev_hash must equal first entry's chain_hash
    assert chain_a[1]["prev_hash"] == chain_a[0]["chain_hash"]
    # Chain B: first (and only) entry must start from zeros
    assert chain_b[0]["prev_hash"] == "0" * 64
