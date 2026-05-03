from fastapi import APIRouter, HTTPException

from app.domain.invoice import Decision

router = APIRouter(prefix="/invoices", tags=["invoices"])


def _get_agent_uc():
    from app.main import _agent_uc
    return _agent_uc


def _get_invoice_repo():
    from app.main import _invoice_repo
    return _invoice_repo


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
