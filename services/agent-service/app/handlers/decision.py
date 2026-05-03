from fastapi import APIRouter, HTTPException

from app.domain.invoice import Decision

router = APIRouter(prefix="/invoices", tags=["invoices"])


def _get_agent_uc():
    from app.main import _agent_uc
    return _agent_uc


def _get_invoice_repo():
    from app.main import _invoice_repo
    return _invoice_repo


def _get_fraud_graph():
    from app.main import _fraud_graph
    return _fraud_graph


@router.get("/{invoice_id}/decision", response_model=Decision)
async def get_decision(invoice_id: str):
    """Trigger or retrieve the agent decision for an invoice."""
    invoice_repo = _get_invoice_repo()
    invoice = invoice_repo.get(invoice_id)
    if invoice is None:
        raise HTTPException(status_code=404, detail=f"Invoice {invoice_id} not found")

    agent_uc = _get_agent_uc()
    decision = await agent_uc.process(invoice)
    return decision


@router.get("/{invoice_id}/fraud-graph")
def get_fraud_graph(invoice_id: str):
    """Return vendor + bank account nodes/edges for fraud graph visualization."""
    invoice_repo = _get_invoice_repo()
    invoice = invoice_repo.get(invoice_id)
    if invoice is None:
        return {"nodes": [], "edges": []}

    vendor_name = getattr(invoice, "vendor_name", None) or ""
    if not vendor_name:
        return {"nodes": [], "edges": []}

    nodes = []
    edges = []

    vendor_id = f"v_{vendor_name}"
    nodes.append({"id": vendor_id, "type": "vendor", "label": vendor_name})

    inv_node_id = f"i_{invoice_id[:8]}"
    amount = getattr(invoice, "amount", None)
    label = f"INV {invoice_id[:8]}" + (f" ${amount}" if amount else "")
    nodes.append({"id": inv_node_id, "type": "invoice", "label": label})
    edges.append({"id": f"e_{vendor_id}_{inv_node_id}", "source": vendor_id, "target": inv_node_id, "label": "ISSUED"})

    try:
        fraud_graph = _get_fraud_graph()
        if fraud_graph.check_bank_account_sharing(vendor_name):
            bank_id = f"b_{vendor_name}"
            nodes.append({"id": bank_id, "type": "bank_account", "label": "Shared Account"})
            edges.append({"id": f"e_{vendor_id}_{bank_id}", "source": vendor_id, "target": bank_id, "label": "PAID_TO"})
    except Exception:
        pass

    return {"nodes": nodes, "edges": edges}
