"""Unit tests for fraud detection logic (calculate_risk_score)."""
import pytest

from app.app.agent.usecase import calculate_risk_score


# ---------------------------------------------------------------------------
# Test 1: All components with known values
# ---------------------------------------------------------------------------

def test_calculate_risk_score_known_values():
    """
    amount_anomaly=40, is_duplicate=False, graph_risk=50, history_risk=60
    score = 40*0.30 + 0*0.30 + 50*0.20 + 60*0.20
          = 12.0   + 0.0    + 10.0    + 12.0
          = 34.0
    """
    score = calculate_risk_score(
        amount_anomaly=40.0,
        is_duplicate=False,
        graph_risk=50.0,
        history_risk=60.0,
    )
    assert abs(score - 34.0) < 0.001


# ---------------------------------------------------------------------------
# Test 2: Capped at 100 when all max inputs
# ---------------------------------------------------------------------------

def test_risk_score_capped_at_100():
    """All components at maximum → score clamped to 100."""
    score = calculate_risk_score(
        amount_anomaly=100.0,
        is_duplicate=True,
        graph_risk=100.0,
        history_risk=100.0,
    )
    assert score == 100.0


# ---------------------------------------------------------------------------
# Test 3: Zero risk — all zero inputs
# ---------------------------------------------------------------------------

def test_risk_score_zero():
    """All zero inputs → score is 0.0."""
    score = calculate_risk_score(
        amount_anomaly=0.0,
        is_duplicate=False,
        graph_risk=0.0,
        history_risk=0.0,
    )
    assert score == 0.0


# ---------------------------------------------------------------------------
# Test 4: is_duplicate=True adds 30 points
# ---------------------------------------------------------------------------

def test_duplicate_weight_adds_30():
    """is_duplicate=True contributes 100*0.30 = 30 points."""
    score_no_dup = calculate_risk_score(0.0, False, 0.0, 0.0)
    score_with_dup = calculate_risk_score(0.0, True, 0.0, 0.0)
    assert abs(score_with_dup - score_no_dup - 30.0) < 0.001


# ---------------------------------------------------------------------------
# Additional: floor at 0 (no negative scores)
# ---------------------------------------------------------------------------

def test_risk_score_floor_at_zero():
    """Ensure score never goes below 0.0 (inputs clamp at 0)."""
    score = calculate_risk_score(-10.0, False, -5.0, -100.0)
    assert score == 0.0


# ---------------------------------------------------------------------------
# Additional: individual weight checks
# ---------------------------------------------------------------------------

def test_amount_anomaly_weight():
    """amount_anomaly=100 with everything else 0 → 100*0.30 = 30."""
    score = calculate_risk_score(100.0, False, 0.0, 0.0)
    assert abs(score - 30.0) < 0.001


def test_graph_risk_weight():
    """graph_risk=100 with everything else 0 → 100*0.20 = 20."""
    score = calculate_risk_score(0.0, False, 100.0, 0.0)
    assert abs(score - 20.0) < 0.001


def test_history_risk_weight():
    """history_risk=100 with everything else 0 → 100*0.20 = 20."""
    score = calculate_risk_score(0.0, False, 0.0, 100.0)
    assert abs(score - 20.0) < 0.001


def test_partial_inputs():
    """
    amount_anomaly=50, is_duplicate=True, graph_risk=0, history_risk=0
    score = 50*0.30 + 100*0.30 = 15 + 30 = 45
    """
    score = calculate_risk_score(50.0, True, 0.0, 0.0)
    assert abs(score - 45.0) < 0.001
