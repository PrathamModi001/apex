import asyncio
import json
import logging
import os
import threading
from datetime import datetime, timezone

from fastapi import FastAPI

from app.handlers.health import router as health_router
from app.handlers.stream import router as stream_router
from app.handlers.policy import router as policy_router
from app.handlers.decision import router as decision_router

logger = logging.getLogger(__name__)

app = FastAPI(title="APEX Agent Service", version="0.2.0")

# ---------------------------------------------------------------------------
# Infrastructure singletons (created once at startup)
# ---------------------------------------------------------------------------

from app.infra.groq.llm import GroqLLMClient
from app.infra.neo4j.fraud_graph import FraudGraphClient
from app.infra.postgres.audit_repo import AuditRepository
from app.infra.postgres.invoice_repo import InvoiceRepository
from app.infra.postgres.feedback_repo import FeedbackRepository
from app.infra.postgres.policy_repo import PolicyRepository
from app.infra.kafka.consumer import KafkaConsumer
from app.infra.kafka.publisher import KafkaPublisher
from app.app.agent.usecase import AgentUseCase
from app.app.policy.usecase import PolicyUseCase

_llm = GroqLLMClient()
_fraud_graph = FraudGraphClient()
_audit_repo = AuditRepository()
_invoice_repo = InvoiceRepository()
_feedback_repo = FeedbackRepository()
_policy_repo = PolicyRepository()
_publisher = KafkaPublisher()

_agent_uc = AgentUseCase(
    llm=_llm,
    fraud_graph=_fraud_graph,
    audit_repo=_audit_repo,
    invoice_repo=_invoice_repo,
    feedback_repo=_feedback_repo,
    publisher=_publisher,
    policy_repo=_policy_repo,
)

_policy_uc = PolicyUseCase(llm=_llm, policy_repo=_policy_repo)

# ---------------------------------------------------------------------------
# Kafka consumer
# ---------------------------------------------------------------------------

_kafka_consumer = KafkaConsumer()
_kafka_stop_event = threading.Event()


def _process_invoice_event(data: dict) -> None:
    """Handler called by Kafka consumer thread for each invoice.processed message."""
    try:
        from app.domain.invoice import ProcessedInvoice
        invoice = ProcessedInvoice.model_validate(data)
        # Run the async use-case from a sync thread
        loop = asyncio.new_event_loop()
        try:
            loop.run_until_complete(_agent_uc.process(invoice))
        finally:
            loop.close()
    except Exception as exc:
        logger.error("Failed to process invoice event: %s — data: %s", exc, data)


# ---------------------------------------------------------------------------
# Routes
# ---------------------------------------------------------------------------

app.include_router(health_router)
app.include_router(stream_router)
app.include_router(policy_router)
app.include_router(decision_router)

# ---------------------------------------------------------------------------
# Lifecycle
# ---------------------------------------------------------------------------


@app.on_event("startup")
def start_kafka_consumer():
    """Start the Kafka consumer in a daemon background thread."""
    t = threading.Thread(
        target=_kafka_consumer.consume,
        args=(_process_invoice_event, _kafka_stop_event),
        daemon=True,
        name="kafka-consumer",
    )
    t.start()
    logger.info("Kafka consumer thread started")


@app.on_event("shutdown")
def stop_kafka_consumer():
    _kafka_stop_event.set()
    logger.info("Kafka consumer stop requested")


if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("PORT", "8000"))
    uvicorn.run("app.main:app", host="0.0.0.0", port=port, reload=False)
