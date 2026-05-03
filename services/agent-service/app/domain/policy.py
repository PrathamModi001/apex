from pydantic import BaseModel
from typing import List, Literal, Optional, Any
from datetime import datetime


class PolicyCondition(BaseModel):
    field: str
    op: Literal["lt", "gt", "eq", "lte", "gte", "neq"]
    value: Any


class CompiledRule(BaseModel):
    conditions: List[PolicyCondition]
    action: Literal["auto_approve", "reject", "flag"]
    logic: Literal["AND", "OR"] = "AND"


class Policy(BaseModel):
    id: Optional[str] = None
    raw_text: str
    compiled_rule: Optional[CompiledRule] = None
    created_by: str = "system"
    active: bool = True
    last_triggered_at: Optional[datetime] = None
