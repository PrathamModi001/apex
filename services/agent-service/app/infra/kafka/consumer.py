import json
import logging
import os
import threading
from typing import Callable

logger = logging.getLogger(__name__)


class KafkaConsumer:
    """Confluent-Kafka consumer for invoice.processed events.

    Runs in a background thread.  If Kafka is unavailable, logs a warning
    and consume() returns immediately.
    """

    def __init__(
        self,
        brokers: str | None = None,
        group_id: str = "agent-service",
        topic: str | None = None,
    ):
        self.brokers = brokers or os.getenv("KAFKA_BROKERS", "redpanda:29092")
        self.group_id = group_id
        self.topic = topic or os.getenv("KAFKA_TOPIC_PROCESSED", "invoice.processed")
        self._consumer = None

        try:
            from confluent_kafka import Consumer

            self._consumer = Consumer(
                {
                    "bootstrap.servers": self.brokers,
                    "group.id": self.group_id,
                    "auto.offset.reset": "earliest",
                    "enable.auto.commit": True,
                }
            )
            self._consumer.subscribe([self.topic])
            logger.info("KafkaConsumer subscribed to %s on %s", self.topic, self.brokers)
        except Exception as exc:
            logger.warning("Could not initialise Kafka consumer — stub mode: %s", exc)
            self._consumer = None

    def consume(self, handler: Callable[[dict], None], stop_event: threading.Event) -> None:
        """Poll Kafka and call handler for each message. Blocks until stop_event is set."""
        if self._consumer is None:
            logger.warning("KafkaConsumer: no consumer available, exiting consume loop")
            return

        from confluent_kafka import KafkaError

        while not stop_event.is_set():
            msg = self._consumer.poll(timeout=1.0)
            if msg is None:
                continue
            if msg.error():
                if msg.error().code() == KafkaError._PARTITION_EOF:
                    continue
                logger.error("Kafka error: %s", msg.error())
                continue
            try:
                data = json.loads(msg.value().decode("utf-8"))
                handler(data)
            except Exception as exc:
                logger.error("Consumer handler error: %s", exc)

        self._consumer.close()
        logger.info("KafkaConsumer closed")
