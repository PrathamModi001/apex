import json
import logging
import os

from app.domain.invoice import Decision

logger = logging.getLogger(__name__)


class KafkaPublisher:
    """Confluent-Kafka producer for invoice.decision events.

    Gracefully stubs when KAFKA_BROKERS is not available.
    """

    def __init__(self, brokers: str | None = None, topic: str | None = None):
        self.brokers = brokers or os.getenv("KAFKA_BROKERS", "redpanda:29092")
        self.topic = topic or os.getenv("KAFKA_TOPIC_DECISION", "invoice.decision")
        self._producer = None

        try:
            from confluent_kafka import Producer

            self._producer = Producer({"bootstrap.servers": self.brokers})
            logger.info("KafkaPublisher ready on %s → %s", self.brokers, self.topic)
        except Exception as exc:
            logger.warning("Could not initialise Kafka producer — stub mode: %s", exc)
            self._producer = None

    def publish_decision(self, decision: Decision) -> None:
        if self._producer is None:
            logger.debug(
                "KafkaPublisher stub: would publish decision %s for %s",
                decision.decision,
                decision.invoice_id,
            )
            return

        payload = json.dumps(
            {
                "invoice_id": decision.invoice_id,
                "decision": decision.decision,
                "risk_score": decision.risk_score,
                "audit_hash": decision.audit_hash,
                "decided_at": decision.decided_at.isoformat(),
            }
        ).encode("utf-8")

        try:
            self._producer.produce(
                self.topic,
                key=decision.invoice_id.encode("utf-8"),
                value=payload,
            )
            self._producer.flush(timeout=5)
            logger.info("Published decision %s for invoice %s", decision.decision, decision.invoice_id)
        except Exception as exc:
            logger.error("KafkaPublisher.publish_decision error: %s", exc)
