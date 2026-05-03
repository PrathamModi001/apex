from pydantic import BaseModel
from datetime import datetime
from typing import Optional, Dict, Any, List


class ExtractedFields(BaseModel):
    invoice_no: str = ""
    amount: float = 0.0
    currency: str = "USD"
    due_date: str = ""
    vendor_name: str = ""


class POMatch(BaseModel):
    po_id: str = ""
    confidence: float = 0.0
    matched: bool = False


class ProcessedInvoice(BaseModel):
    id: str
    source: str
    file_key: str
    sha256: str
    sender: str
    received_at: datetime
    extracted_fields: ExtractedFields
    po_match: POMatch
    processed_at: datetime


class Decision(BaseModel):
    invoice_id: str
    decision: str  # "AUTO_APPROVE" | "FLAGGED" | "REJECTED" | "POLICY_MATCH"
    risk_score: float  # 0-100
    reasoning_steps: List[Dict[str, Any]]
    audit_hash: str
    decided_at: datetime


class AgentStep(BaseModel):
    step: int
    thought: str
    action: str  # tool name
    action_input: Dict[str, Any]
    observation: str
