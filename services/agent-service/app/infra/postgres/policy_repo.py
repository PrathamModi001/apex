import json
import logging
import os
import uuid
from typing import List, Optional

from app.domain.policy import CompiledRule, Policy, PolicyCondition

logger = logging.getLogger(__name__)


class PolicyRepository:
    """Postgres-backed policy storage.

    Falls back to an in-memory list when DATABASE_URL is not set.
    """

    def __init__(self, db_url: str | None = None):
        self.db_url = db_url or os.getenv("DATABASE_URL", "")
        self._conn = None
        self._memory: List[Policy] = []

        if self.db_url:
            try:
                import psycopg2
                self._conn = psycopg2.connect(self.db_url)
                self._ensure_table()
                logger.info("PolicyRepository connected to Postgres")
            except Exception as exc:
                logger.warning("Cannot connect to Postgres — using in-memory policy store: %s", exc)
                self._conn = None
        else:
            logger.warning("DATABASE_URL not set — using in-memory policy store")

    def _ensure_table(self):
        if not self._conn:
            return
        try:
            with self._conn.cursor() as cur:
                cur.execute(
                    """
                    CREATE TABLE IF NOT EXISTS policies (
                        id TEXT PRIMARY KEY,
                        raw_text TEXT NOT NULL,
                        compiled_rule JSONB,
                        created_by TEXT DEFAULT 'system',
                        active BOOLEAN DEFAULT TRUE,
                        last_triggered_at TIMESTAMPTZ
                    )
                    """
                )
                self._conn.commit()
        except Exception as exc:
            logger.warning("PolicyRepository._ensure_table error: %s", exc)

    def get_active(self) -> List[Policy]:
        if self._conn:
            try:
                with self._conn.cursor() as cur:
                    cur.execute(
                        "SELECT id, raw_text, compiled_rule, created_by, active, last_triggered_at "
                        "FROM policies WHERE active = TRUE"
                    )
                    rows = cur.fetchall()
                    return [self._row_to_policy(r) for r in rows]
            except Exception as exc:
                logger.error("PolicyRepository.get_active error: %s", exc)
        return [p for p in self._memory if p.active]

    def get_all(self) -> List[Policy]:
        if self._conn:
            try:
                with self._conn.cursor() as cur:
                    cur.execute(
                        "SELECT id, raw_text, compiled_rule, created_by, active, last_triggered_at FROM policies"
                    )
                    rows = cur.fetchall()
                    return [self._row_to_policy(r) for r in rows]
            except Exception as exc:
                logger.error("PolicyRepository.get_all error: %s", exc)
        return list(self._memory)

    def save(self, policy: Policy) -> str:
        policy_id = policy.id or str(uuid.uuid4())
        compiled_json = policy.compiled_rule.model_dump() if policy.compiled_rule else None

        if self._conn:
            try:
                with self._conn.cursor() as cur:
                    cur.execute(
                        """
                        INSERT INTO policies (id, raw_text, compiled_rule, created_by, active)
                        VALUES (%s, %s, %s, %s, %s)
                        ON CONFLICT (id) DO UPDATE SET
                            raw_text = EXCLUDED.raw_text,
                            compiled_rule = EXCLUDED.compiled_rule,
                            active = EXCLUDED.active
                        """,
                        (
                            policy_id,
                            policy.raw_text,
                            json.dumps(compiled_json) if compiled_json else None,
                            policy.created_by,
                            policy.active,
                        ),
                    )
                    self._conn.commit()
                return policy_id
            except Exception as exc:
                logger.error("PolicyRepository.save error: %s", exc)

        # In-memory fallback
        policy.id = policy_id
        # Replace if exists
        self._memory = [p for p in self._memory if p.id != policy_id]
        self._memory.append(policy)
        return policy_id

    def update_active(self, policy_id: str, active: bool) -> None:
        if self._conn:
            try:
                with self._conn.cursor() as cur:
                    cur.execute(
                        "UPDATE policies SET active = %s WHERE id = %s",
                        (active, policy_id),
                    )
                    self._conn.commit()
                return
            except Exception as exc:
                logger.error("PolicyRepository.update_active error: %s", exc)
        for p in self._memory:
            if p.id == policy_id:
                p.active = active
                break

    @staticmethod
    def _row_to_policy(row) -> Policy:
        policy_id, raw_text, compiled_rule_json, created_by, active, last_triggered_at = row
        compiled_rule = None
        if compiled_rule_json:
            try:
                if isinstance(compiled_rule_json, str):
                    compiled_rule_json = json.loads(compiled_rule_json)
                compiled_rule = CompiledRule.model_validate(compiled_rule_json)
            except Exception:
                pass
        return Policy(
            id=policy_id,
            raw_text=raw_text,
            compiled_rule=compiled_rule,
            created_by=created_by or "system",
            active=active,
            last_triggered_at=last_triggered_at,
        )
