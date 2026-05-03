import os
import json
import logging
from typing import List

logger = logging.getLogger(__name__)

_STUB_CHAT_RESPONSE = (
    "Thought: Analyzing the invoice.\n"
    "Action: db_lookup\n"
    'Action Input: {"invoice_id": "stub-id"}\n'
)

_STUB_FINAL_ANSWER = (
    "Thought: Based on the analysis, this invoice appears normal.\n"
    'Final Answer: {"decision": "AUTO_APPROVE", "risk_score": 10, "reason": "Low risk invoice"}'
)


class GroqLLMClient:
    """HTTP wrapper around Groq's chat completions API.

    Falls back to stub responses when GROQ_API_KEY is not set, so local
    development and unit tests work without credentials.
    """

    def __init__(self, api_key: str | None = None, base_url: str = "https://api.groq.com/openai/v1"):
        self.api_key = api_key or os.getenv("GROQ_API_KEY", "")
        self.base_url = base_url
        self._stub_mode = not bool(self.api_key)
        if self._stub_mode:
            logger.warning("GROQ_API_KEY not set — using stub LLM responses")
        self._stub_call_count = 0

    def chat(self, messages: list, model: str = "llama-3.3-70b-versatile") -> str:
        if self._stub_mode:
            self._stub_call_count += 1
            # After 2 stub calls return Final Answer to avoid infinite loop
            if self._stub_call_count % 3 == 0:
                self._stub_call_count = 0
                return _STUB_FINAL_ANSWER
            return _STUB_CHAT_RESPONSE

        import httpx  # lazy import to avoid mandatory dep in test isolation

        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
        }
        payload = {
            "model": model,
            "messages": messages,
            "temperature": 0.1,
            "max_tokens": 1024,
        }
        with httpx.Client(timeout=30) as client:
            resp = client.post(f"{self.base_url}/chat/completions", headers=headers, json=payload)
            resp.raise_for_status()
            data = resp.json()
            return data["choices"][0]["message"]["content"]

    def embed(self, text: str) -> list[float]:
        """Return a text embedding vector.

        Groq does not currently offer an embeddings endpoint, so we always
        return a zero vector stub.  Replace with OpenAI / local model as needed.
        """
        return [0.0] * 768
