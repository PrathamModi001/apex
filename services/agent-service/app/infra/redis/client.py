import logging
import os
from typing import Optional

logger = logging.getLogger(__name__)


class RedisClient:
    """Thin wrapper around redis-py.

    Degrades gracefully when REDIS_URL is not configured.
    """

    def __init__(self, url: str | None = None):
        self.url = url or os.getenv("REDIS_URL", "")
        self._client = None

        if self.url:
            try:
                import redis
                self._client = redis.from_url(self.url, decode_responses=True)
                self._client.ping()
                logger.info("Redis connected: %s", self.url)
            except Exception as exc:
                logger.warning("Cannot connect to Redis — stub mode: %s", exc)
                self._client = None
        else:
            logger.warning("REDIS_URL not set — Redis stub mode")

    def get(self, key: str) -> Optional[str]:
        if self._client:
            try:
                return self._client.get(key)
            except Exception as exc:
                logger.error("Redis GET error: %s", exc)
        return None

    def set(self, key: str, value: str, ex: Optional[int] = None) -> None:
        if self._client:
            try:
                self._client.set(key, value, ex=ex)
            except Exception as exc:
                logger.error("Redis SET error: %s", exc)

    def delete(self, key: str) -> None:
        if self._client:
            try:
                self._client.delete(key)
            except Exception as exc:
                logger.error("Redis DELETE error: %s", exc)

    def publish(self, channel: str, message: str) -> None:
        if self._client:
            try:
                self._client.publish(channel, message)
            except Exception as exc:
                logger.error("Redis PUBLISH error: %s", exc)
