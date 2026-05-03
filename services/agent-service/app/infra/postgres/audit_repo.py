import hashlib
import json
import logging
import os
from datetime import datetime, timezone
from typing import List, Dict

logger = logging.getLogger(__name__)


def _chain_hash(prev_hash: str, payload: dict) -> str:
    """Compute SHA-256 of prev_hash concatenated with canonicalised JSON of payload."""
    data = prev_hash + json.dumps(payload, sort_keys=True)
    return hashlib.sha256(data.encode()).hexdigest()


class AuditRepository:
    """Append-only Merkle audit log stored in Postgres.

    Falls back to an in-memory list when DATABASE_URL is not set.
    """

    def __init__(self, db_url: str | None = None):
        self.db_url = db_url or os.getenv("DATABASE_URL", "")
        self._conn = None
        # In-memory fallback: list of dicts
        self._memory: List[Dict] = []

        if self.db_url:
            try:
                import psycopg2
                self._conn = psycopg2.connect(self.db_url)
                self._ensure_table()
                logger.info("AuditRepository connected to Postgres")
            except Exception as exc:
                logger.warning("Cannot connect to Postgres (%s) — using in-memory audit: %s", self.db_url, exc)
                self._conn = None
        else:
            logger.warning("DATABASE_URL not set — using in-memory audit log")

    def _ensure_table(self):
        if not self._conn:
            return
        with self._conn.cursor() as cur:
            cur.execute(
                """
                CREATE TABLE IF NOT EXISTS audit_log (
                    id SERIAL PRIMARY KEY,
                    invoice_id TEXT NOT NULL,
                    event_type TEXT NOT NULL,
                    actor TEXT NOT NULL,
                    payload JSONB NOT NULL,
                    prev_hash TEXT NOT NULL,
                    chain_hash TEXT NOT NULL,
                    created_at TIMESTAMPTZ DEFAULT now()
                )
                """
            )
            self._conn.commit()

    def _get_last_hash(self, invoice_id: str) -> str:
        """Return the most recent chain_hash for invoice_id, or '0'*64 if none."""
        if self._conn:
            try:
                with self._conn.cursor() as cur:
                    cur.execute(
                        "SELECT chain_hash FROM audit_log WHERE invoice_id = %s ORDER BY id DESC LIMIT 1",
                        (invoice_id,),
                    )
                    row = cur.fetchone()
                    return row[0] if row else "0" * 64
            except Exception as exc:
                logger.error("_get_last_hash DB error: %s", exc)

        # In-memory fallback
        matching = [r for r in self._memory if r["invoice_id"] == invoice_id]
        if matching:
            return matching[-1]["chain_hash"]
        return "0" * 64

    def append(self, invoice_id: str, event_type: str, actor: str, payload: dict) -> str:
        """Append an audit event and return the new chain_hash."""
        prev = self._get_last_hash(invoice_id)
        new_hash = _chain_hash(prev, payload)

        if self._conn:
            try:
                with self._conn.cursor() as cur:
                    cur.execute(
                        """
                        INSERT INTO audit_log (invoice_id, event_type, actor, payload, prev_hash, chain_hash)
                        VALUES (%s, %s, %s, %s, %s, %s)
                        """,
                        (invoice_id, event_type, actor, json.dumps(payload), prev, new_hash),
                    )
                    self._conn.commit()
                return new_hash
            except Exception as exc:
                logger.error("AuditRepository.append DB error: %s", exc)

        # In-memory fallback
        self._memory.append(
            {
                "invoice_id": invoice_id,
                "event_type": event_type,
                "actor": actor,
                "payload": payload,
                "prev_hash": prev,
                "chain_hash": new_hash,
                "created_at": datetime.now(timezone.utc).isoformat(),
            }
        )
        return new_hash

    def get_chain(self, invoice_id: str) -> List[dict]:
        """Return all audit entries for an invoice in insertion order."""
        if self._conn:
            try:
                with self._conn.cursor() as cur:
                    cur.execute(
                        "SELECT invoice_id, event_type, actor, payload, prev_hash, chain_hash, created_at "
                        "FROM audit_log WHERE invoice_id = %s ORDER BY id ASC",
                        (invoice_id,),
                    )
                    rows = cur.fetchall()
                    return [
                        {
                            "invoice_id": r[0],
                            "event_type": r[1],
                            "actor": r[2],
                            "payload": r[3],
                            "prev_hash": r[4],
                            "chain_hash": r[5],
                            "created_at": r[6].isoformat() if r[6] else None,
                        }
                        for r in rows
                    ]
            except Exception as exc:
                logger.error("AuditRepository.get_chain DB error: %s", exc)

        return [r for r in self._memory if r["invoice_id"] == invoice_id]
