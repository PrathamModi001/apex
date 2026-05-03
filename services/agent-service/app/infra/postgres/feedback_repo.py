import json
import logging
import os
from typing import List

logger = logging.getLogger(__name__)


class FeedbackRepository:
    """Postgres + pgvector-backed feedback storage for few-shot injection.

    Falls back to an empty in-memory list when DATABASE_URL is not set.
    """

    def __init__(self, db_url: str | None = None):
        self.db_url = db_url or os.getenv("DATABASE_URL", "")
        self._conn = None
        self._memory: List[dict] = []

        if self.db_url:
            try:
                import psycopg2
                self._conn = psycopg2.connect(self.db_url)
                self._ensure_table()
                logger.info("FeedbackRepository connected to Postgres")
            except Exception as exc:
                logger.warning("Cannot connect to Postgres — using in-memory feedback store: %s", exc)
                self._conn = None
        else:
            logger.warning("DATABASE_URL not set — using in-memory feedback store")

    def _ensure_table(self):
        if not self._conn:
            return
        try:
            with self._conn.cursor() as cur:
                cur.execute("CREATE EXTENSION IF NOT EXISTS vector")
                cur.execute(
                    """
                    CREATE TABLE IF NOT EXISTS feedback (
                        id SERIAL PRIMARY KEY,
                        invoice_id TEXT NOT NULL,
                        agent_decision TEXT,
                        human_decision TEXT,
                        reason TEXT,
                        invoice_summary TEXT,
                        embedding vector(768),
                        created_at TIMESTAMPTZ DEFAULT now()
                    )
                    """
                )
                self._conn.commit()
        except Exception as exc:
            logger.warning("FeedbackRepository._ensure_table error: %s", exc)

    def get_similar_corrections(self, embedding: List[float], limit: int = 3) -> List[dict]:
        """Return up to `limit` corrections closest to the given embedding."""
        if self._conn:
            try:
                vec_str = "[" + ",".join(str(x) for x in embedding) + "]"
                with self._conn.cursor() as cur:
                    cur.execute(
                        """
                        SELECT invoice_id, agent_decision, human_decision, reason, invoice_summary
                        FROM feedback
                        ORDER BY embedding <-> %s::vector
                        LIMIT %s
                        """,
                        (vec_str, limit),
                    )
                    rows = cur.fetchall()
                    return [
                        {
                            "invoice_id": r[0],
                            "agent_decision": r[1],
                            "human_decision": r[2],
                            "reason": r[3],
                            "invoice_summary": r[4],
                        }
                        for r in rows
                    ]
            except Exception as exc:
                logger.error("FeedbackRepository.get_similar_corrections error: %s", exc)
        return self._memory[:limit]

    def save_correction(
        self,
        invoice_id: str,
        agent_decision: str,
        human_decision: str,
        reason: str,
        invoice_summary: str = "",
        embedding: List[float] | None = None,
    ) -> None:
        """Persist a human correction."""
        if self._conn and embedding:
            try:
                vec_str = "[" + ",".join(str(x) for x in embedding) + "]"
                with self._conn.cursor() as cur:
                    cur.execute(
                        """
                        INSERT INTO feedback (invoice_id, agent_decision, human_decision, reason, invoice_summary, embedding)
                        VALUES (%s, %s, %s, %s, %s, %s::vector)
                        """,
                        (invoice_id, agent_decision, human_decision, reason, invoice_summary, vec_str),
                    )
                    self._conn.commit()
                return
            except Exception as exc:
                logger.error("FeedbackRepository.save_correction error: %s", exc)

        self._memory.append(
            {
                "invoice_id": invoice_id,
                "agent_decision": agent_decision,
                "human_decision": human_decision,
                "reason": reason,
                "invoice_summary": invoice_summary,
            }
        )
