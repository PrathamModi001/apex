import asyncio

from fastapi import APIRouter
from sse_starlette.sse import EventSourceResponse

router = APIRouter()

# In-memory store of active invoice streams (invoice_id -> asyncio.Queue)
_streams: dict[str, asyncio.Queue] = {}


def push_token(invoice_id: str, data: str) -> None:
    """Called by the agent loop to push reasoning tokens to SSE clients."""
    if invoice_id in _streams:
        try:
            _streams[invoice_id].put_nowait({"type": "token", "data": data})
        except asyncio.QueueFull:
            pass


def push_done(invoice_id: str) -> None:
    """Signal the SSE stream for invoice_id that processing is complete."""
    if invoice_id in _streams:
        try:
            _streams[invoice_id].put_nowait({"type": "done", "data": ""})
        except asyncio.QueueFull:
            pass


@router.get("/stream/invoice/{invoice_id}")
async def stream_invoice(invoice_id: str):
    """SSE endpoint that streams agent reasoning tokens for an invoice."""
    queue: asyncio.Queue = asyncio.Queue(maxsize=256)
    _streams[invoice_id] = queue

    async def event_generator():
        try:
            while True:
                event = await asyncio.wait_for(queue.get(), timeout=60.0)
                yield {"event": event["type"], "data": event["data"]}
                if event["type"] == "done":
                    break
        except asyncio.TimeoutError:
            yield {"event": "timeout", "data": ""}
        finally:
            _streams.pop(invoice_id, None)

    return EventSourceResponse(event_generator())
