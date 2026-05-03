import json
import re
from typing import Any, Dict, Optional

from app.domain.invoice import ProcessedInvoice
from app.domain.policy import CompiledRule, Policy, PolicyCondition


class PolicyUseCase:
    def __init__(self, llm, policy_repo=None):
        self.llm = llm
        self.policy_repo = policy_repo

    def evaluate(self, policy: Policy, invoice: ProcessedInvoice) -> bool:
        """Evaluate compiled policy rules against invoice fields. No LLM at runtime."""
        if not policy.compiled_rule:
            return False

        fields: Dict[str, Any] = {
            "amount": invoice.extracted_fields.amount,
            "currency": invoice.extracted_fields.currency,
            "vendor_approved": invoice.po_match.matched,
            "po_confidence": invoice.po_match.confidence,
        }

        ops = {
            "lt": lambda a, b: a < b,
            "gt": lambda a, b: a > b,
            "eq": lambda a, b: a == b,
            "lte": lambda a, b: a <= b,
            "gte": lambda a, b: a >= b,
            "neq": lambda a, b: a != b,
        }

        results = []
        for cond in policy.compiled_rule.conditions:
            val = fields.get(cond.field)
            if val is None:
                results.append(False)
                continue
            op_fn = ops.get(cond.op)
            if op_fn is None:
                results.append(False)
                continue
            results.append(op_fn(val, cond.value))

        if not results:
            return False

        if policy.compiled_rule.logic == "AND":
            return all(results)
        return any(results)

    def compile(self, raw_text: str) -> CompiledRule:
        """Use LLM to compile natural-language rule to JSON."""
        prompt = f"""Convert this AP policy rule to JSON:
Rule: {raw_text}

Output ONLY valid JSON:
{{"conditions": [{{"field": "amount", "op": "lt", "value": 500}}], "action": "auto_approve", "logic": "AND"}}

Valid fields: amount, currency, vendor_approved, po_confidence
Valid ops: lt, gt, eq, lte, gte, neq
Valid actions: auto_approve, reject, flag"""
        response = self.llm.chat([{"role": "user", "content": prompt}])
        # Strip markdown code fences if present
        cleaned = response.strip()
        if cleaned.startswith("```"):
            cleaned = re.sub(r"^```[a-z]*\n?", "", cleaned)
            cleaned = re.sub(r"\n?```$", "", cleaned)
            cleaned = cleaned.strip()
        try:
            return CompiledRule.model_validate_json(cleaned)
        except Exception as exc:
            raise ValueError(
                f"Failed to parse CompiledRule from LLM response: {exc}\nResponse was: {response}"
            )

    def save_policy(self, raw_text: str, created_by: str = "system") -> Policy:
        """Compile and persist a new policy."""
        compiled_rule = self.compile(raw_text)
        policy = Policy(raw_text=raw_text, compiled_rule=compiled_rule, created_by=created_by)
        if self.policy_repo:
            policy_id = self.policy_repo.save(policy)
            policy.id = policy_id
        return policy
