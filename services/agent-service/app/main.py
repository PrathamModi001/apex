import os

from fastapi import FastAPI

from app.handlers.health import router as health_router

app = FastAPI(title="APEX Agent Service", version="0.1.0")

# Routes — Phase 1: health only. LLM agent, OCR, fraud endpoints added in Phase 4.
app.include_router(health_router)

if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("PORT", "8000"))
    uvicorn.run("app.main:app", host="0.0.0.0", port=port, reload=False)
