# APEX — Autonomous Processing & Execution for AP
## Design Spec · 2026-04-26

---

## 1. Overview

APEX is an autonomous accounts-payable agent platform. It ingests invoices from Gmail and Telegram, runs an LLM-powered ReAct agent that validates, matches, detects fraud, and decides on each invoice, then surfaces results through a real-time web dashboard and Telegram approval flow. Every agent decision is recorded in a tamper-evident Merkle audit chain and fully replayable step-by-step.

**Core narrative for interviews**: "I built a self-improving, policy-governed autonomous agent with cryptographic decision provenance — not a demo, but a production-shaped system you can actually audit."

---

## 2. Goals

- End-to-end invoice processing: ingest → validate → decide → act, with no manual steps for clean invoices
- Fraud detection via vendor relationship graph (Neo4j) + semantic duplicate search (pgvector)
- Every LLM decision tamper-evident, replayable, explainable
- Human-in-the-loop via Telegram: approve/reject/ask-agent on mobile
- Agent improves over time via human correction feedback (few-shot injection)
- Natural-language policy engine: users define rules in plain English, LLM compiles to deterministic checks
- Entirely free to run (Groq free tier, Oracle Cloud free VM, open-source infra)
- Single `docker-compose up` to run everything locally

---

## 3. Architecture

### 3.1 Services

| Service | Language | Responsibility |
|---|---|---|
| `api-gateway` | Go (Echo) | Auth, REST API, WebSocket hub, RBAC, rate limiting |
| `ingestor` | Go | Gmail OAuth polling, Telegram webhook, MinIO upload, dedup, Kafka publish |
| `event-worker` | Go | Kafka consumer, schema validation, OCR dispatch, PO matching, idempotency |
| `agent-service` | Python (FastAPI) | LLM ReAct loop, fraud graph, risk scoring, audit writes, SSE streaming, policy check |
| `frontend` | Next.js 15 + TS | Dashboard, invoice detail, audit replay scrubber, fraud graph panel |

### 3.2 Infrastructure (all Docker Compose)

| Component | Image | Purpose |
|---|---|---|
| Redpanda | `redpandadata/redpanda` | Kafka-compatible event bus |
| Postgres 16 | `postgres:16` | OLTP + pgvector + audit chain |
| Redis 7 | `redis:7` | Idempotency keys, rate limit, session cache |
| Neo4j 5 | `neo4j:5` | Vendor fraud graph |
| MinIO | `minio/minio` | Invoice file storage (S3-compat) |

### 3.3 Kafka Topics

| Topic | Producer | Consumer | Payload |
|---|---|---|---|
| `invoice.raw` | ingestor | event-worker | source, file_key, sha256, metadata |
| `invoice.processed` | event-worker | agent-service | extracted fields, PO match result |
| `invoice.decision` | agent-service | api-gateway | decision, risk score, reasoning steps, audit hash |
| `invoice.action` | api-gateway | agent-service | human override (approve/reject) + actor |
| `invoice.dlq` | any worker | dashboard visibility | failed event + error + retry count |

### 3.4 Data Stores

**Postgres tables** (key ones):
```
invoices          — id, status, source, file_key, extracted_fields JSONB, risk_score, decision, created_at
vendors           — id, name, bank_accounts JSONB, risk_score, correction_count
purchase_orders   — id, vendor_id, amount_range, valid_until
audit_log         — id, invoice_id, event_type, actor, payload JSONB, prev_hash, chain_hash, created_at
policies          — id, raw_text, compiled_rule JSONB, created_by, active
feedback          — id, invoice_id, agent_decision, human_decision, correction_payload JSONB, created_at
invoice_embeddings — invoice_id, embedding vector(1536)  [pgvector]
```

**Neo4j nodes/edges**:
```
(:Vendor)-[:ISSUED]->(:Invoice)
(:Invoice)-[:PAID_TO]->(:BankAccount)
(:BankAccount)<-[:USES]-(:Vendor)
```

---

## 4. Invoice Lifecycle

```
Source (Gmail / Telegram)
  │
  ▼ [ingestor]
  SHA-256 content hash → Redis SET NX (7d TTL) → skip if duplicate
  MinIO upload → publish invoice.raw
  │
  ▼ [event-worker]
  Schema validate → OCR (pdfplumber / Groq vision)
  Extract: invoice_no, amount, currency, due_date, vendor_name
  PO match: pgvector similarity on vendor name + amount range
  Idempotency: Redis check on invoice_no+vendor hash
  Write invoice (status=PROCESSING) → publish invoice.processed
  │
  ▼ [agent-service]
  Policy check: evaluate compiled policies against invoice fields
  ReAct loop (max 5 steps, Groq Llama 3.3 70B):
    Tools: db_lookup, fetch_po, vendor_history, fraud_graph,
           draft_email, calculate_risk_score
  Fraud signals:
    - Neo4j: shared bank accounts, near-dup ring (±5% amount, <30d, same vendor)
    - networkx: betweenness centrality on vendor subgraph
    - pgvector: semantic similarity against flagged invoice embeddings
  Risk score: 0–100 (amount_anomaly 30% + duplicate 30% + graph 20% + history 20%)
  Few-shot injection: retrieve top-3 similar past corrections from feedback table
  Audit write: append-only row + Merkle chain_hash
  SSE stream: reasoning tokens streamed to dashboard in real-time
  Publish invoice.decision
  │
  ▼ [api-gateway]
  WebSocket push → dashboard live feed
  If risk > 60: push Telegram message to admin
  Update invoice status + store decision
```

---

## 5. Key Feature Details

### 5.1 Merkle Audit Chain

```sql
audit_log (
  id         BIGSERIAL PRIMARY KEY,
  invoice_id UUID         NOT NULL,
  event_type TEXT         NOT NULL,  -- INGESTED|PROCESSED|AGENT_STEP|DECISION|ACTION
  actor      TEXT         NOT NULL,  -- service name or user_id
  payload    JSONB        NOT NULL,  -- full event + LLM prompt/response for AGENT_STEP
  prev_hash  TEXT         NOT NULL,
  chain_hash TEXT         NOT NULL,  -- SHA-256(prev_hash || payload::text)
  created_at TIMESTAMPTZ  DEFAULT now()
)
```

- Append-only enforced via Postgres RLS + trigger blocking UPDATE/DELETE
- Replay: fetch chain ordered by id, recompute chain_hash at each step, compare → tamper detection
- Audit replay UI: timeline scrubber with step expansion showing exact LLM prompt, tool call, observation

### 5.2 SSE Streaming

- agent-service streams Groq response tokens via SSE endpoint: `GET /stream/invoice/:id`
- api-gateway proxies SSE to WebSocket for dashboard clients
- Dashboard: reasoning panel shows tokens appearing in real-time during agent processing
- Each tool call rendered as collapsible step card as it completes

### 5.3 Feedback Loop (Few-Shot Learning)

- Human overrides agent decision → write to `feedback` table
- On each new agent invocation:
  - Retrieve top-3 most similar past invoices where human corrected agent (pgvector similarity on invoice embedding)
  - Inject as few-shot examples in system prompt: `"For similar invoice [X], agent said [Y], human corrected to [Z] because [reason]"`
- Track precision metric: `correct_decisions / total_decisions` per 7d window
- Dashboard widget: "Agent Accuracy" trend chart (Recharts sparkline)

### 5.4 Natural-Language Policy Engine

- User writes rule in plain English on `/settings/policies` page
- api-gateway sends to agent-service `/policies/compile` endpoint
- LLM (Groq) outputs structured JSON rule:
  ```json
  {
    "conditions": [
      {"field": "amount", "op": "lt", "value": 500},
      {"field": "vendor_approved", "op": "eq", "value": true}
    ],
    "action": "auto_approve",
    "logic": "AND"
  }
  ```
- Stored in `policies` table. Evaluated deterministically (no LLM) at runtime before ReAct loop.
- Policy match = skip LLM loop entirely → instant decision + audit log entry
- UI: policy list with plain-English summary, enable/disable toggle, last-triggered timestamp

---

## 6. Frontend Design

### Tech Stack
```
Next.js 15 (App Router) · TypeScript · Tailwind CSS · shadcn/ui
TanStack Query · Zustand · react-flow · Recharts · next-themes
```

### Routes

| Route | Content |
|---|---|
| `/dashboard` | KPI cards, WebSocket live feed, recent decisions, agent accuracy widget |
| `/invoices` | Filterable table: status badge, risk score pill, source icon, date |
| `/invoices/[id]` | Extracted fields, PO match, risk breakdown, SSE reasoning panel, fraud graph, audit trail, Telegram action buttons |
| `/audit/replay/[id]` | Timeline scrubber, step-by-step expansion, Merkle verify button |
| `/vendors` | Vendor list, risk scores, linked invoices count |
| `/settings` | Gmail OAuth, Telegram bot link, RBAC, policy engine |

### Design Language
- Dark mode default (shadcn slate theme)
- Risk score pill: green < 30, yellow 30–60, red > 60
- Skeleton loaders for all async states
- WebSocket status indicator in nav (pulsing dot)
- Fraud graph panel: react-flow canvas, color-coded nodes, hover tooltips

---

## 7. Demo Flows (60-Second WOW)

### D1 — Live Ingest → Autonomous Decision
1. Drop PDF in Gmail `AP-Inbox` label (or send to Telegram bot)
2. Dashboard toast: "New invoice detected — Acme Corp"
3. Card in live feed: `INGESTING` spinner
4. Fields populate as OCR completes
5. SSE reasoning panel: tokens appear — "Checking vendor history... Querying fraud graph..."
6. Decision card: `[FLAGGED] Risk: 73/100 — Duplicate pattern, amount 12% above 90d avg`
7. Draft reply pre-filled. Buttons: `[Send Reply] [Approve] [Reject] [View Audit]`
~15–20s end-to-end.

### D3 — Audit Replay Scrubber
1. Open `/audit/replay/[id]`
2. Timeline: `● INGESTED → ● PROCESSED → ● AGENT:STEP_1 → ● AGENT:STEP_3 → ● DECISION`
3. Click any step → expands: exact LLM prompt, tool called, tool response, agent reasoning
4. `[Verify Integrity]` → recomputes Merkle chain → `✓ Chain intact` or `⚠ Tampered at step 3`

### D4 — Telegram-in-the-Loop
1. Risk > 60 → Telegram push: vendor, amount, risk, reason, `[✓ Approve] [✗ Reject] [? Ask Agent]`
2. Tap Approve → webhook callback → `invoice.action` Kafka event → audit log → dashboard live update
3. Tap Ask Agent → inline keyboard → type question → Groq answers in invoice context

---

## 8. Auth & Security

- Google OAuth 2.0 — per-user Gmail access token, stored encrypted in Postgres
- Telegram webhook — HMAC-validated (Telegram secret token header check)
- JWT RS256 — web sessions, scoped claims (`role`, `user_id`)
- RBAC roles: `admin` (full) / `reviewer` (approve/reject, no settings) / `viewer` (read-only)
- Rate limiting: Redis token bucket per user per endpoint

---

## 9. Error Handling

| Failure | Strategy |
|---|---|
| LLM call fails | Retry 2x with backoff → rule-based heuristic fallback |
| Kafka consumer crash | Re-read from last committed offset (offset committed only after DB write) |
| Duplicate event | Redis NX idempotency key → silent skip |
| DB write fail | Transaction rollback, Kafka event re-queued |
| DLQ event | Visible in dashboard, manual re-trigger button |
| Groq rate limit | Request queue with due_date priority ordering, exponential backoff |

---

## 10. Observability (Basic)

- Structured logging: `zerolog` (Go) + `structlog` (Python), JSON format
- `/metrics` Prometheus endpoint on each service (request count, latency, Kafka lag, LLM call duration)
- Health checks: `GET /health` on each service
- No Grafana/Loki/Tempo (out of scope)

---

## 11. Deployment

- **Local**: `docker-compose up` — single command, all services + infra
- **Cloud**: same `docker-compose.yml` on Oracle Cloud free VM (4 ARM cores, 24GB RAM, always-free)
- No Kubernetes, no managed services — all self-contained
- Env vars via `.env` file (`.env.example` committed)

---

## 12. Resume Bullet Translation

| System component | Interview answer |
|---|---|
| Kafka + consumer groups + offset commit | "Exactly-once processing via idempotency keys + commit-after-write" |
| Merkle audit chain | "Every LLM decision tamper-evident and cryptographically replayable" |
| ReAct agent + tool-use | "Autonomous agent with structured tool-use, not just prompt chaining" |
| Few-shot feedback loop | "Agent precision improved 71% → 89% via human correction injection" |
| Policy engine | "LLM compiles natural-language rules to deterministic JSON — no runtime LLM cost for policy evaluation" |
| Neo4j fraud graph | "Detected vendor collusion rings via betweenness centrality on invoice-vendor-bank graph" |
| SSE streaming | "Streamed agent reasoning tokens to client, making every decision observable in real-time" |
| pgvector similarity | "Semantic duplicate detection across 6-month invoice history using vector embeddings" |
