from fastapi import APIRouter, HTTPException
from pydantic import BaseModel
from typing import Optional

from app.domain.policy import Policy, CompiledRule

router = APIRouter(prefix="/policies", tags=["policies"])


class CompileRequest(BaseModel):
    raw_text: str
    created_by: str = "system"


class PatchPolicyRequest(BaseModel):
    active: bool


def _get_policy_uc():
    """Lazy import to avoid circular imports at module level."""
    from app.app.policy.usecase import PolicyUseCase
    from app.main import _policy_uc
    return _policy_uc


def _get_policy_repo():
    from app.main import _policy_repo
    return _policy_repo


@router.post("/compile", response_model=Policy)
def compile_policy(req: CompileRequest):
    """Compile a natural-language policy rule using the LLM and persist it."""
    uc = _get_policy_uc()
    try:
        policy = uc.save_policy(req.raw_text, created_by=req.created_by)
        return policy
    except ValueError as exc:
        raise HTTPException(status_code=422, detail=str(exc))
    except Exception as exc:
        raise HTTPException(status_code=500, detail=f"Failed to compile policy: {exc}")


@router.get("", response_model=list[Policy])
def list_policies():
    """Return all policies (active and inactive)."""
    repo = _get_policy_repo()
    try:
        return repo.get_all()
    except AttributeError:
        return repo.get_active()


@router.patch("/{policy_id}", response_model=dict)
def patch_policy(policy_id: str, req: PatchPolicyRequest):
    """Enable or disable a policy."""
    repo = _get_policy_repo()
    try:
        repo.update_active(policy_id, req.active)
        return {"policy_id": policy_id, "active": req.active}
    except Exception as exc:
        raise HTTPException(status_code=500, detail=str(exc))
