import json
import re
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional

from app.domain.invoice import Decision, ProcessedInvoice
from app.domain.policy import Policy

TOOLS = {
    "db_lookup": "Look up invoice details by ID",
    "fetch_po": "Fetch purchase order by vendor name",
    "vendor_history": "Get vendor's invoice history (count, avg_amount, flags)",
    "fraud_graph": "Query Neo4j for vendor fraud signals",
    "draft_email": "Draft approval/rejection email to vendor",
    "calculate_risk_score": "Calculate risk score based on signals dict",
}

SYSTEM_PROMPT = """You are an AP (accounts payable) auditing agent.
Your job: analyze invoices and decide: AUTO_APPROVE, FLAGGED, or REJECTED.

Available tools: {tools}

Use ReAct format:
Thought: <your reasoning>
Action: <tool_name>
Action Input: <JSON dict>
Observation: <tool result>
...
Final Answer: <JSON: {{"decision": "AUTO_APPROVE|FLAGGED|REJECTED", "risk_score": 0-100, "reason": "..."}}>

Rules:
- Use at least 2 tools before Final Answer
- If risk_score > 60: FLAGGED
- If fraud_graph shows shared bank accounts: REJECTED
- If amount is within 10% of PO amount and vendor approved: AUTO_APPROVE
"""


def calculate_risk_score(
    amount_anomaly: float,
    is_duplicate: bool,
    graph_risk: float,
    history_risk: float,
) -> float:
    score = (
        amount_anomaly * 0.30
        + (100 if is_duplicate else 0) * 0.30
        + graph_risk * 0.20
        + history_risk * 0.20
    )
    return min(100.0, max(0.0, score))


def _parse_react_step(text: str) -> Optional[Dict[str, Any]]:
    """Parse a single ReAct Thought/Action/Action Input block from LLM output."""
    thought_match = re.search(r"Thought:\s*(.*?)(?=Action:|Final Answer:|$)", text, re.DOTALL)
    action_match = re.search(r"Action:\s*(\w+)", text)
    input_match = re.search(r"Action Input:\s*(\{.*?\})", text, re.DOTALL)

    if not action_match:
        return None

    thought = thought_match.group(1).strip() if thought_match else ""
    action = action_match.group(1).strip()
    action_input: Dict[str, Any] = {}
    if input_match:
        try:
            action_input = json.loads(input_match.group(1))
        except json.JSONDecodeError:
            action_input = {"raw": input_match.group(1)}

    return {"thought": thought, "action": action, "action_input": action_input}


def _parse_final_answer(text: str) -> Optional[Dict[str, Any]]:
    """Extract the Final Answer JSON from LLM output."""
    match = re.search(r"Final Answer:\s*(\{.*\})", text, re.DOTALL)
    if not match:
        return None
    try:
        return json.loads(match.group(1))
    except json.JSONDecodeError:
        return None


class AgentUseCase:
    def __init__(
        self,
        llm,
        fraud_graph,
        audit_repo,
        invoice_repo,
        feedback_repo,
        publisher,
        policy_repo=None,
    ):
        self.llm = llm
        self.fraud_graph = fraud_graph
        self.audit_repo = audit_repo
        self.invoice_repo = invoice_repo
        self.feedback_repo = feedback_repo
        self.publisher = publisher
        self.policy_repo = policy_repo

    def _execute_tool(self, action: str, action_input: Dict[str, Any], invoice: ProcessedInvoice) -> str:
        """Execute a named tool and return its string observation."""
        if action == "db_lookup":
            invoice_id = action_input.get("invoice_id", invoice.id)
            inv = self.invoice_repo.get(invoice_id)
            if inv:
                return json.dumps({
                    "id": inv.id,
                    "vendor": inv.extracted_fields.vendor_name,
                    "amount": inv.extracted_fields.amount,
                    "currency": inv.extracted_fields.currency,
                })
            return json.dumps({"error": "invoice not found"})

        elif action == "fetch_po":
            vendor = action_input.get("vendor_name", invoice.extracted_fields.vendor_name)
            return json.dumps({
                "po_id": invoice.po_match.po_id,
                "confidence": invoice.po_match.confidence,
                "matched": invoice.po_match.matched,
                "vendor": vendor,
            })

        elif action == "vendor_history":
            vendor = action_input.get("vendor_name", invoice.extracted_fields.vendor_name)
            risk_info = self.fraud_graph.get_vendor_risk(vendor)
            return json.dumps({
                "vendor": vendor,
                "invoice_count": risk_info.get("invoice_count", 0),
                "risk_score": risk_info.get("risk_score", 0.0),
            })

        elif action == "fraud_graph":
            vendor = action_input.get("vendor_name", invoice.extracted_fields.vendor_name)
            shared = self.fraud_graph.check_bank_account_sharing(vendor)
            centrality = self.fraud_graph.get_betweenness_centrality(vendor)
            risk_info = self.fraud_graph.get_vendor_risk(vendor)
            return json.dumps({
                "vendor": vendor,
                "shared_bank_account": shared,
                "betweenness_centrality": centrality,
                "risk_score": risk_info.get("risk_score", 0.0),
            })

        elif action == "draft_email":
            decision = action_input.get("decision", "PENDING")
            vendor = action_input.get("vendor", invoice.extracted_fields.vendor_name)
            return json.dumps({
                "email_drafted": True,
                "to": vendor,
                "subject": f"Invoice {invoice.id} - {decision}",
            })

        elif action == "calculate_risk_score":
            amount_anomaly = float(action_input.get("amount_anomaly", 0.0))
            is_duplicate = bool(action_input.get("is_duplicate", False))
            graph_risk = float(action_input.get("graph_risk", 0.0))
            history_risk = float(action_input.get("history_risk", 0.0))
            score = calculate_risk_score(amount_anomaly, is_duplicate, graph_risk, history_risk)
            return json.dumps({"risk_score": score})

        else:
            return json.dumps({"error": f"unknown tool: {action}"})

    async def process(self, invoice: ProcessedInvoice) -> Decision:
        reasoning_steps: List[Dict[str, Any]] = []
        final_audit_hash = "0" * 64

        # Step 1: Deterministic policy evaluation (skip LLM if policy matches)
        if self.policy_repo is not None:
            from app.app.policy.usecase import PolicyUseCase
            policy_uc = PolicyUseCase(llm=self.llm, policy_repo=self.policy_repo)
            active_policies = self.policy_repo.get_active()
            for policy in active_policies:
                if policy.compiled_rule and policy_uc.evaluate(policy, invoice):
                    action_map = {
                        "auto_approve": "AUTO_APPROVE",
                        "reject": "REJECTED",
                        "flag": "FLAGGED",
                    }
                    decision_str = action_map.get(policy.compiled_rule.action, "POLICY_MATCH")
                    audit_hash = self.audit_repo.append(
                        invoice_id=invoice.id,
                        event_type="POLICY_MATCH",
                        actor="policy_engine",
                        payload={
                            "policy_id": policy.id,
                            "policy_action": policy.compiled_rule.action,
                            "invoice_id": invoice.id,
                        },
                    )
                    decision = Decision(
                        invoice_id=invoice.id,
                        decision="POLICY_MATCH",
                        risk_score=0.0,
                        reasoning_steps=[
                            {
                                "step": 0,
                                "thought": f"Policy {policy.id} matched",
                                "action": "policy_check",
                                "action_input": {},
                                "observation": f"action={policy.compiled_rule.action}",
                            }
                        ],
                        audit_hash=audit_hash,
                        decided_at=datetime.now(timezone.utc),
                    )
                    self.invoice_repo.update_decision(invoice.id, decision)
                    self.publisher.publish_decision(decision)
                    return decision

        # Step 2: Retrieve few-shot corrections from feedback
        few_shot_examples = ""
        try:
            vendor_text = f"{invoice.extracted_fields.vendor_name} {invoice.extracted_fields.amount}"
            embedding = self.llm.embed(vendor_text)
            corrections = self.feedback_repo.get_similar_corrections(embedding, limit=3)
            if corrections:
                examples = []
                for c in corrections:
                    examples.append(
                        f"Invoice: {c.get('invoice_summary', 'N/A')} | "
                        f"Agent said: {c.get('agent_decision', 'N/A')} | "
                        f"Human corrected to: {c.get('human_decision', 'N/A')} | "
                        f"Reason: {c.get('reason', 'N/A')}"
                    )
                few_shot_examples = "\n\nFew-shot corrections from human feedback:\n" + "\n".join(examples)
        except Exception:
            few_shot_examples = ""

        # Step 3: Build system prompt with tools and few-shot examples
        tools_str = "\n".join(f"- {name}: {desc}" for name, desc in TOOLS.items())
        system_content = SYSTEM_PROMPT.format(tools=tools_str) + few_shot_examples

        invoice_summary = (
            f"Invoice ID: {invoice.id}\n"
            f"Vendor: {invoice.extracted_fields.vendor_name}\n"
            f"Amount: {invoice.extracted_fields.amount} {invoice.extracted_fields.currency}\n"
            f"PO Matched: {invoice.po_match.matched} (confidence: {invoice.po_match.confidence})\n"
            f"Due Date: {invoice.extracted_fields.due_date}"
        )

        messages = [
            {"role": "system", "content": system_content},
            {"role": "user", "content": f"Analyze this invoice:\n{invoice_summary}"},
        ]

        # Step 4: ReAct loop (max 5 steps)
        MAX_STEPS = 5
        final_answer: Optional[Dict[str, Any]] = None

        for step_num in range(MAX_STEPS):
            response = self.llm.chat(messages)

            # Check for Final Answer
            if "Final Answer:" in response:
                final_answer = _parse_final_answer(response)
                # Record the final answer as a step
                reasoning_steps.append({
                    "step": step_num,
                    "thought": "Final Answer reached",
                    "action": "final_answer",
                    "action_input": {},
                    "observation": response,
                })
                audit_hash = self.audit_repo.append(
                    invoice_id=invoice.id,
                    event_type="REACT_FINAL",
                    actor="agent",
                    payload={"step": step_num, "response": response[:500]},
                )
                final_audit_hash = audit_hash
                break

            # Parse the Thought/Action/Action Input
            parsed = _parse_react_step(response)
            if parsed is None:
                # No action found, treat as done
                reasoning_steps.append({
                    "step": step_num,
                    "thought": response,
                    "action": "none",
                    "action_input": {},
                    "observation": "no action found",
                })
                break

            thought = parsed["thought"]
            action = parsed["action"]
            action_input = parsed["action_input"]

            # Execute tool
            observation = self._execute_tool(action, action_input, invoice)

            # Record step
            step_record = {
                "step": step_num,
                "thought": thought,
                "action": action,
                "action_input": action_input,
                "observation": observation,
            }
            reasoning_steps.append(step_record)

            # Write audit entry for this step
            audit_hash = self.audit_repo.append(
                invoice_id=invoice.id,
                event_type="REACT_STEP",
                actor="agent",
                payload=step_record,
            )
            final_audit_hash = audit_hash

            # Append to messages for next iteration
            messages.append({"role": "assistant", "content": response})
            messages.append({"role": "user", "content": f"Observation: {observation}\nContinue."})

        # Step 5: Parse Final Answer → Decision
        if final_answer:
            decision_str = final_answer.get("decision", "FLAGGED")
            risk_score = float(final_answer.get("risk_score", 50.0))
        else:
            # Max steps reached without Final Answer — default to FLAGGED
            decision_str = "FLAGGED"
            risk_score = 50.0

        # Clamp risk score
        risk_score = min(100.0, max(0.0, risk_score))

        # Step 6: Enforce risk score rules
        if risk_score > 60 and decision_str == "AUTO_APPROVE":
            decision_str = "FLAGGED"

        # Step 7–9 covered via audit entries above; create decision object
        decision = Decision(
            invoice_id=invoice.id,
            decision=decision_str,
            risk_score=risk_score,
            reasoning_steps=reasoning_steps,
            audit_hash=final_audit_hash,
            decided_at=datetime.now(timezone.utc),
        )

        # Step 8: Update invoice status in Postgres
        self.invoice_repo.update_decision(invoice.id, decision)

        # Step 9: Publish to invoice.decision
        self.publisher.publish_decision(decision)

        return decision
