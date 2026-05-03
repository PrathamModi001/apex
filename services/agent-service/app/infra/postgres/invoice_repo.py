import json
import logging
import os
from datetime import datetime
from typing import Optional

from app.domain.invoice import Decision, ExtractedFields, POMatch, ProcessedInvoice

logger = logging.getLogger(__name__)


class InvoiceRepository:
    """Postgres-backed invoice storage.

    Falls back to an in-memory dict when DATABASE_URL is not set.
    """

    def __init__(self, db_url: str | None = None):
        self.db_url = db_url or os.getenv("DATABASE_URL", "")
        self._conn = None
        self._memory: dict = {}  # invoice_id -> ProcessedInvoice

        if self.db_url:
            try:
                import psycopg2
                self._conn = psycopg2.connect(self.db_url)
                logger.info("InvoiceRepository connected to Postgres")
            except Exception as exc:
                logger.warning("Cannot connect to Postgres — using in-memory invoice store: %s", exc)
                self._conn = None
        else:
            logger.warning("DATABASE_URL not set — using in-memory invoice store")

    def get(self, invoice_id: str) -> Optional[ProcessedInvoice]:
        if self._conn:
            try:
                with self._conn.cursor() as cur:
                    cur.execute(
                        "SELECT data FROM invoices WHERE id = %s",
                        (invoice_id,),
                    )
                    row = cur.fetchone()
                    if row:
                        return ProcessedInvoice.model_validate(row[0])
            except Exception as exc:
                logger.error("InvoiceRepository.get error: %s", exc)
        return self._memory.get(invoice_id)

    def update_decision(self, invoice_id: str, decision: Decision) -> None:
        if self._conn:
            try:
                with self._conn.cursor() as cur:
                    cur.execute(
                        """
                        UPDATE invoices
                        SET decision = %s, decided_at = %s
                        WHERE id = %s
                        """,
                        (decision.decision, decision.decided_at, invoice_id),
                    )
                    self._conn.commit()
                return
            except Exception as exc:
                logger.error("InvoiceRepository.update_decision error: %s", exc)
        # In-memory: no-op (decision is tracked via audit log)
        logger.debug("InvoiceRepository.update_decision (in-memory): %s -> %s", invoice_id, decision.decision)

    def upsert(self, invoice: ProcessedInvoice) -> None:
        """Store a ProcessedInvoice in the in-memory fallback."""
        self._memory[invoice.id] = invoice
