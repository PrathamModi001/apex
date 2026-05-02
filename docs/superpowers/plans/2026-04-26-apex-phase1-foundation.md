# APEX Phase 1: Foundation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Monorepo scaffolded, all infra running via `docker-compose up`, DB schema migrated with pgvector, every service answers `GET /health → 200`.

**Architecture:** 4 backend services (3 Go, 1 Python FastAPI) + Next.js frontend. All infra (Postgres 16+pgvector, Redpanda, Redis 7, Neo4j 5, MinIO) in Docker Compose. Migrations auto-apply on Postgres start. Kafka topics created by init container.

**Tech Stack:** Go 1.22 + Echo v4.11, Python 3.12 + FastAPI 0.110 + uvicorn, Next.js 15 + TypeScript + Tailwind + shadcn/ui, Docker Compose v2, pgvector/pgvector:pg16

**Prerequisites:** Docker Desktop, Go 1.22+, Python 3.12+, Node 20+, Git, make (via Git Bash on Windows)

---

## File Map

```
apex/
├── .env.example
├── .gitignore
├── docker-compose.yml
├── go.work
├── Makefile
├── migrations/
│   ├── 001_init.sql
│   └── 002_pgvector.sql
├── services/
│   ├── api-gateway/
│   │   ├── cmd/server/main.go
│   │   ├── internal/health/handler.go
│   │   ├── internal/health/handler_test.go
│   │   ├── go.mod
│   │   └── Dockerfile
│   ├── ingestor/
│   │   ├── cmd/ingestor/main.go
│   │   ├── internal/health/handler.go
│   │   ├── internal/health/handler_test.go
│   │   ├── go.mod
│   │   └── Dockerfile
│   ├── event-worker/
│   │   ├── cmd/worker/main.go
│   │   ├── internal/health/handler.go
│   │   ├── internal/health/handler_test.go
│   │   ├── go.mod
│   │   └── Dockerfile
│   └── agent-service/
│       ├── app/
│       │   ├── main.py
│       │   └── health.py
│       ├── tests/
│       │   └── test_health.py
│       ├── requirements.txt
│       └── Dockerfile
└── frontend/
    ├── app/
    │   ├── layout.tsx
    │   ├── page.tsx
    │   └── api/health/route.ts
    ├── next.config.ts
    ├── tailwind.config.ts
    ├── tsconfig.json
    └── Dockerfile
```

---

### Task 1: Git init + project scaffold files

**Files:**
- Create: `.gitignore`
- Create: `.env.example`
- Create: `Makefile`
- Create: `go.work`

- [ ] **Step 1: Init repo**

```bash
cd "C:/Coding/New folder"
git init
mkdir -p migrations services/api-gateway/cmd/server services/api-gateway/internal/health
mkdir -p services/ingestor/cmd/ingestor services/ingestor/internal/health
mkdir -p services/event-worker/cmd/worker services/event-worker/internal/health
mkdir -p services/agent-service/app services/agent-service/tests
mkdir -p frontend/app/api/health docs/superpowers/specs docs/superpowers/plans
```

- [ ] **Step 2: Write `.gitignore`**

```
.env
*.env.local
services/*/vendor/
__pycache__/
*.pyc
.pytest_cache/
.venv/
venv/
frontend/node_modules/
frontend/.next/
frontend/out/
*.pem
secrets/
.vscode/
.idea/
.DS_Store
Thumbs.db
```

- [ ] **Step 3: Write `.env.example`**

```
# Postgres
POSTGRES_DB=apex
POSTGRES_USER=apex
POSTGRES_PASSWORD=apex_secret
DATABASE_URL=postgresql://apex:apex_secret@postgres:5432/apex

# Redis
REDIS_URL=redis://redis:6379

# Kafka
KAFKA_BROKERS=redpanda:29092

# Neo4j
NEO4J_AUTH=neo4j/apex_neo4j_secret
NEO4J_URL=bolt://neo4j:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=apex_neo4j_secret

# MinIO
MINIO_ROOT_USER=apex_minio
MINIO_ROOT_PASSWORD=apex_minio_secret
MINIO_ENDPOINT=minio:9000
MINIO_BUCKET=invoices

# LLM
GROQ_API_KEY=your_groq_api_key_here

# Auth
JWT_SECRET=change_me_in_production

# Gmail OAuth
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
GOOGLE_REDIRECT_URI=http://localhost:8080/auth/google/callback

# Telegram
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
TELEGRAM_WEBHOOK_SECRET=your_webhook_secret

# Service URLs (for inter-service calls)
AGENT_SERVICE_URL=http://agent-service:8000

# Frontend
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080
```

- [ ] **Step 4: Copy `.env.example` to `.env`**

```bash
cp .env.example .env
```

- [ ] **Step 5: Write `Makefile`**

```makefile
.PHONY: up down infra-up infra-down logs test-go test-python test build

infra-up:
	docker-compose up -d postgres redis redpanda neo4j minio redpanda-init minio-init

infra-down:
	docker-compose stop postgres redis redpanda neo4j minio

up:
	docker-compose up -d

down:
	docker-compose down

logs:
	docker-compose logs -f

test-go:
	cd services/api-gateway && go test ./...
	cd services/ingestor && go test ./...
	cd services/event-worker && go test ./...

test-python:
	cd services/agent-service && python -m pytest tests/ -v

test: test-go test-python

build:
	docker-compose build
```

- [ ] **Step 6: Commit**

```bash
git add .gitignore .env.example Makefile
git commit -m "chore: init repo with project scaffold"
```

---

### Task 2: Docker Compose — infra services

**Files:**
- Create: `docker-compose.yml`

- [ ] **Step 1: Write `docker-compose.yml`**

```yaml
version: '3.9'

services:
  redpanda:
    image: redpandadata/redpanda:latest
    container_name: apex-redpanda
    command:
      - redpanda
      - start
      - --smp
      - "1"
      - --memory
      - 512M
      - --overprovisioned
      - --node-id
      - "0"
      - --kafka-addr
      - PLAINTEXT://0.0.0.0:29092,OUTSIDE://0.0.0.0:9092
      - --advertise-kafka-addr
      - PLAINTEXT://redpanda:29092,OUTSIDE://localhost:9092
    ports:
      - "9092:9092"
      - "29092:29092"
    healthcheck:
      test: ["CMD", "rpk", "cluster", "info"]
      interval: 10s
      timeout: 5s
      retries: 10

  redpanda-init:
    image: redpandadata/redpanda:latest
    container_name: apex-redpanda-init
    depends_on:
      redpanda:
        condition: service_healthy
    entrypoint: ["/bin/bash", "-c"]
    command: >
      "rpk --brokers redpanda:29092 topic create
      invoice.raw invoice.processed invoice.decision invoice.action invoice.dlq
      --partitions 2 --replicas 1 || true"
    restart: "no"

  postgres:
    image: pgvector/pgvector:pg16
    container_name: apex-postgres
    env_file: .env
    environment:
      POSTGRES_DB: ${POSTGRES_DB:-apex}
      POSTGRES_USER: ${POSTGRES_USER:-apex}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-apex_secret}
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-apex}"]
      interval: 5s
      timeout: 5s
      retries: 10

  redis:
    image: redis:7-alpine
    container_name: apex-redis
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  neo4j:
    image: neo4j:5
    container_name: apex-neo4j
    environment:
      NEO4J_AUTH: ${NEO4J_AUTH:-neo4j/apex_neo4j_secret}
    ports:
      - "7474:7474"
      - "7687:7687"
    volumes:
      - neo4j_data:/data
    healthcheck:
      test: ["CMD", "wget", "-O", "/dev/null", "-q", "http://localhost:7474"]
      interval: 10s
      timeout: 5s
      retries: 15

  minio:
    image: minio/minio:latest
    container_name: apex-minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER:-apex_minio}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD:-apex_minio_secret}
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 10s
      timeout: 5s
      retries: 5

  minio-init:
    image: minio/mc:latest
    container_name: apex-minio-init
    depends_on:
      minio:
        condition: service_healthy
    entrypoint: ["/bin/sh", "-c"]
    command: >
      "mc alias set apex http://minio:9000 ${MINIO_ROOT_USER:-apex_minio} ${MINIO_ROOT_PASSWORD:-apex_minio_secret}
      && mc mb apex/invoices --ignore-existing"
    restart: "no"

volumes:
  postgres_data:
  neo4j_data:
  minio_data:
```

- [ ] **Step 2: Start infra and verify**

```bash
make infra-up
```

Wait ~30s, then:

```bash
docker-compose ps
```

Expected: all 5 infra containers show `healthy` or `exited 0` for init containers.

- [ ] **Step 3: Commit**

```bash
git add docker-compose.yml
git commit -m "infra: add docker-compose with Redpanda, Postgres, Redis, Neo4j, MinIO"
```

---

### Task 3: Database migrations

**Files:**
- Create: `migrations/001_init.sql`
- Create: `migrations/002_pgvector.sql`

- [ ] **Step 1: Write `migrations/001_init.sql`**

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email        TEXT UNIQUE NOT NULL,
    role         TEXT NOT NULL DEFAULT 'viewer',
    gmail_token  JSONB,
    telegram_user_id TEXT,
    created_at   TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE vendors (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name             TEXT NOT NULL,
    bank_accounts    JSONB DEFAULT '[]',
    risk_score       NUMERIC(5,2) DEFAULT 0,
    correction_count INTEGER DEFAULT 0,
    created_at       TIMESTAMPTZ DEFAULT now(),
    updated_at       TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE purchase_orders (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vendor_id    UUID REFERENCES vendors(id),
    po_number    TEXT UNIQUE NOT NULL,
    amount_min   NUMERIC(15,2),
    amount_max   NUMERIC(15,2),
    valid_until  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE invoices (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source              TEXT NOT NULL,
    file_key            TEXT NOT NULL,
    sha256              TEXT NOT NULL UNIQUE,
    status              TEXT NOT NULL DEFAULT 'INGESTED',
    vendor_id           UUID REFERENCES vendors(id),
    vendor_name         TEXT,
    invoice_number      TEXT,
    amount              NUMERIC(15,2),
    currency            TEXT DEFAULT 'USD',
    due_date            DATE,
    extracted_fields    JSONB DEFAULT '{}',
    po_id               UUID REFERENCES purchase_orders(id),
    risk_score          NUMERIC(5,2),
    decision            TEXT,
    decision_confidence NUMERIC(5,2),
    draft_reply         TEXT,
    error_message       TEXT,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE audit_log (
    id         BIGSERIAL PRIMARY KEY,
    invoice_id UUID NOT NULL REFERENCES invoices(id),
    event_type TEXT NOT NULL,
    actor      TEXT NOT NULL,
    payload    JSONB NOT NULL,
    prev_hash  TEXT NOT NULL,
    chain_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE OR REPLACE FUNCTION prevent_audit_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_log is append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_log_append_only
    BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();

CREATE TABLE policies (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    raw_text          TEXT NOT NULL,
    compiled_rule     JSONB NOT NULL,
    created_by        UUID REFERENCES users(id),
    active            BOOLEAN DEFAULT true,
    last_triggered_at TIMESTAMPTZ,
    trigger_count     INTEGER DEFAULT 0,
    created_at        TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE feedback (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    invoice_id      UUID NOT NULL REFERENCES invoices(id),
    agent_decision  TEXT NOT NULL,
    human_decision  TEXT NOT NULL,
    correction_payload JSONB DEFAULT '{}',
    actor_id        UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ DEFAULT now()
);
```

- [ ] **Step 2: Write `migrations/002_pgvector.sql`**

```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE invoice_embeddings (
    invoice_id UUID PRIMARY KEY REFERENCES invoices(id) ON DELETE CASCADE,
    embedding  vector(1536) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX ON invoice_embeddings
    USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);
```

- [ ] **Step 3: Verify migrations applied**

Migrations auto-apply when Postgres container first starts (mounts to `/docker-entrypoint-initdb.d`). If Postgres is already running from Task 2, recreate it to apply migrations:

```bash
docker-compose stop postgres
docker volume rm "$(basename $(pwd))_postgres_data" 2>/dev/null || docker volume rm new_folder_postgres_data
docker-compose up -d postgres
```

Wait ~10s, then verify:

```bash
docker exec apex-postgres psql -U apex -d apex -c "\dt"
```

Expected output includes: `invoices`, `vendors`, `purchase_orders`, `audit_log`, `policies`, `feedback`, `users`, `invoice_embeddings`

- [ ] **Step 4: Commit**

```bash
git add migrations/
git commit -m "db: add initial schema with audit_log Merkle chain and pgvector"
```

---

### Task 4: api-gateway Go scaffold

**Files:**
- Create: `services/api-gateway/go.mod`
- Create: `services/api-gateway/internal/health/handler.go`
- Create: `services/api-gateway/internal/health/handler_test.go`
- Create: `services/api-gateway/cmd/server/main.go`
- Create: `services/api-gateway/Dockerfile`

- [ ] **Step 1: Write failing test**

`services/api-gateway/internal/health/handler_test.go`:
```go
package health_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"apex/api-gateway/internal/health"
)

func TestHandler_ReturnsOK(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := health.Handler("api-gateway")(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}

	var resp health.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("want status=ok, got %q", resp.Status)
	}
	if resp.Service != "api-gateway" {
		t.Errorf("want service=api-gateway, got %q", resp.Service)
	}
}
```

- [ ] **Step 2: Init Go module**

```bash
cd services/api-gateway
go mod init apex/api-gateway
go get github.com/labstack/echo/v4@v4.11.4
```

- [ ] **Step 3: Run test — verify FAIL**

```bash
go test ./internal/health/...
```

Expected: `FAIL — cannot find package "apex/api-gateway/internal/health"`

- [ ] **Step 4: Write handler**

`services/api-gateway/internal/health/handler.go`:
```go
package health

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Response struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func Handler(serviceName string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, Response{
			Status:  "ok",
			Service: serviceName,
		})
	}
}
```

- [ ] **Step 5: Run test — verify PASS**

```bash
go test ./internal/health/... -v
```

Expected:
```
--- PASS: TestHandler_ReturnsOK (0.00s)
PASS
```

- [ ] **Step 6: Write main.go**

`services/api-gateway/cmd/server/main.go`:
```go
package main

import (
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"apex/api-gateway/internal/health"
)

func main() {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/health", health.Handler("api-gateway"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(e.Start(":" + port))
}
```

- [ ] **Step 7: Write Dockerfile**

`services/api-gateway/Dockerfile`:
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/server ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata wget
WORKDIR /app
COPY --from=builder /bin/server .
EXPOSE 8080
CMD ["./server"]
```

- [ ] **Step 8: Commit**

```bash
cd ../..
git add services/api-gateway/
git commit -m "feat(api-gateway): scaffold with /health endpoint"
```

---

### Task 5: ingestor Go scaffold

**Files:**
- Create: `services/ingestor/go.mod`
- Create: `services/ingestor/internal/health/handler.go`
- Create: `services/ingestor/internal/health/handler_test.go`
- Create: `services/ingestor/cmd/ingestor/main.go`
- Create: `services/ingestor/Dockerfile`

- [ ] **Step 1: Write failing test**

`services/ingestor/internal/health/handler_test.go`:
```go
package health_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"apex/ingestor/internal/health"
)

func TestHandler_ReturnsOK(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := health.Handler("ingestor")(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	var resp health.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("want status=ok, got %q", resp.Status)
	}
	if resp.Service != "ingestor" {
		t.Errorf("want service=ingestor, got %q", resp.Service)
	}
}
```

- [ ] **Step 2: Init module + get deps**

```bash
cd services/ingestor
go mod init apex/ingestor
go get github.com/labstack/echo/v4@v4.11.4
```

- [ ] **Step 3: Run test — verify FAIL**

```bash
go test ./internal/health/...
```

Expected: `FAIL — cannot find package`

- [ ] **Step 4: Write handler**

`services/ingestor/internal/health/handler.go`:
```go
package health

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Response struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func Handler(serviceName string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, Response{
			Status:  "ok",
			Service: serviceName,
		})
	}
}
```

- [ ] **Step 5: Run test — verify PASS**

```bash
go test ./internal/health/... -v
```

Expected: `--- PASS: TestHandler_ReturnsOK`

- [ ] **Step 6: Write main.go**

`services/ingestor/cmd/ingestor/main.go`:
```go
package main

import (
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"apex/ingestor/internal/health"
)

func main() {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/health", health.Handler("ingestor"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	log.Fatal(e.Start(":" + port))
}
```

- [ ] **Step 7: Write Dockerfile**

`services/ingestor/Dockerfile`:
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/ingestor ./cmd/ingestor

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata wget
WORKDIR /app
COPY --from=builder /bin/ingestor .
EXPOSE 8081
CMD ["./ingestor"]
```

- [ ] **Step 8: Commit**

```bash
cd ../..
git add services/ingestor/
git commit -m "feat(ingestor): scaffold with /health endpoint"
```

---

### Task 6: event-worker Go scaffold

**Files:**
- Create: `services/event-worker/go.mod`
- Create: `services/event-worker/internal/health/handler.go`
- Create: `services/event-worker/internal/health/handler_test.go`
- Create: `services/event-worker/cmd/worker/main.go`
- Create: `services/event-worker/Dockerfile`

- [ ] **Step 1: Write failing test**

`services/event-worker/internal/health/handler_test.go`:
```go
package health_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"apex/event-worker/internal/health"
)

func TestHandler_ReturnsOK(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := health.Handler("event-worker")(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	var resp health.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("want status=ok, got %q", resp.Status)
	}
	if resp.Service != "event-worker" {
		t.Errorf("want service=event-worker, got %q", resp.Service)
	}
}
```

- [ ] **Step 2: Init module + get deps**

```bash
cd services/event-worker
go mod init apex/event-worker
go get github.com/labstack/echo/v4@v4.11.4
```

- [ ] **Step 3: Run test — verify FAIL**

```bash
go test ./internal/health/...
```

Expected: `FAIL — cannot find package`

- [ ] **Step 4: Write handler**

`services/event-worker/internal/health/handler.go`:
```go
package health

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Response struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func Handler(serviceName string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, Response{
			Status:  "ok",
			Service: serviceName,
		})
	}
}
```

- [ ] **Step 5: Run test — verify PASS**

```bash
go test ./internal/health/... -v
```

Expected: `--- PASS: TestHandler_ReturnsOK`

- [ ] **Step 6: Write main.go**

`services/event-worker/cmd/worker/main.go`:
```go
package main

import (
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"apex/event-worker/internal/health"
)

func main() {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/health", health.Handler("event-worker"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	log.Fatal(e.Start(":" + port))
}
```

- [ ] **Step 7: Write Dockerfile**

`services/event-worker/Dockerfile`:
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/worker ./cmd/worker

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata wget
WORKDIR /app
COPY --from=builder /bin/worker .
EXPOSE 8082
CMD ["./worker"]
```

- [ ] **Step 8: Commit**

```bash
cd ../..
git add services/event-worker/
git commit -m "feat(event-worker): scaffold with /health endpoint"
```

---

### Task 7: agent-service Python scaffold

**Files:**
- Create: `services/agent-service/requirements.txt`
- Create: `services/agent-service/app/health.py`
- Create: `services/agent-service/app/main.py`
- Create: `services/agent-service/tests/test_health.py`
- Create: `services/agent-service/tests/__init__.py`
- Create: `services/agent-service/Dockerfile`

- [ ] **Step 1: Write failing test**

`services/agent-service/tests/__init__.py`: (empty file)

`services/agent-service/tests/test_health.py`:
```python
from fastapi.testclient import TestClient


def test_health_returns_ok():
    from app.main import app
    client = TestClient(app)
    response = client.get("/health")
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "ok"
    assert data["service"] == "agent-service"
```

- [ ] **Step 2: Write `requirements.txt`**

```
fastapi==0.110.0
uvicorn[standard]==0.27.1
httpx==0.26.0
pytest==8.0.1
```

- [ ] **Step 3: Install deps**

```bash
cd services/agent-service
python -m venv .venv
source .venv/Scripts/activate   # Windows Git Bash
pip install -r requirements.txt
```

- [ ] **Step 4: Run test — verify FAIL**

```bash
python -m pytest tests/ -v
```

Expected: `ModuleNotFoundError: No module named 'app'`

- [ ] **Step 5: Write health router**

`services/agent-service/app/__init__.py`: (empty file)

`services/agent-service/app/health.py`:
```python
from fastapi import APIRouter

router = APIRouter()


@router.get("/health")
def health_check():
    return {"status": "ok", "service": "agent-service"}
```

- [ ] **Step 6: Write main.py**

`services/agent-service/app/main.py`:
```python
import os

from fastapi import FastAPI

from app.health import router as health_router

app = FastAPI(title="APEX Agent Service", version="0.1.0")
app.include_router(health_router)

if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("PORT", "8000"))
    uvicorn.run("app.main:app", host="0.0.0.0", port=port, reload=False)
```

- [ ] **Step 7: Run test — verify PASS**

```bash
python -m pytest tests/ -v
```

Expected:
```
PASSED tests/test_health.py::test_health_returns_ok
```

- [ ] **Step 8: Write Dockerfile**

`services/agent-service/Dockerfile`:
```dockerfile
FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 8000
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
```

- [ ] **Step 9: Commit**

```bash
cd ../..
git add services/agent-service/
git commit -m "feat(agent-service): scaffold with /health endpoint"
```

---

### Task 8: Next.js frontend scaffold

**Files:**
- Create: `frontend/` (via create-next-app)
- Create: `frontend/app/api/health/route.ts`
- Modify: `frontend/next.config.ts`
- Create: `frontend/Dockerfile`

- [ ] **Step 1: Scaffold Next.js app**

```bash
npx create-next-app@latest frontend \
  --typescript \
  --tailwind \
  --eslint \
  --app \
  --no-src-dir \
  --import-alias "@/*" \
  --no-turbopack
```

When prompted, answer No to all optional extras.

- [ ] **Step 2: Install shadcn/ui**

```bash
cd frontend
npx shadcn@latest init -d
```

Select: style=Default, base color=Slate, CSS variables=Yes.

- [ ] **Step 3: Write health API route**

`frontend/app/api/health/route.ts`:
```typescript
import { NextResponse } from 'next/server'

export function GET() {
  return NextResponse.json({ status: 'ok', service: 'frontend' })
}
```

- [ ] **Step 4: Update next.config.ts for standalone Docker output**

`frontend/next.config.ts`:
```typescript
import type { NextConfig } from 'next'

const nextConfig: NextConfig = {
  output: 'standalone',
}

export default nextConfig
```

- [ ] **Step 5: Verify dev server**

```bash
npm run dev
```

Open http://localhost:3000/api/health — expect `{"status":"ok","service":"frontend"}`

Stop dev server (Ctrl+C).

- [ ] **Step 6: Write Dockerfile**

`frontend/Dockerfile`:
```dockerfile
FROM node:20-alpine AS base

FROM base AS deps
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci

FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM base AS runner
WORKDIR /app
ENV NODE_ENV=production
RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs
COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
USER nextjs
EXPOSE 3000
ENV PORT=3000
CMD ["node", "server.js"]
```

- [ ] **Step 7: Commit**

```bash
cd ..
git add frontend/
git commit -m "feat(frontend): scaffold Next.js 15 with shadcn/ui and /api/health"
```

---

### Task 9: Add Go workspace + wire all services into docker-compose

**Files:**
- Create: `go.work`
- Modify: `docker-compose.yml` (add 5 service entries)

- [ ] **Step 1: Write go.work**

`go.work`:
```
go 1.22

use (
	./services/api-gateway
	./services/ingestor
	./services/event-worker
)
```

- [ ] **Step 2: Add service entries to docker-compose.yml**

Append to the `services:` block in `docker-compose.yml` (before the `volumes:` section):

```yaml
  api-gateway:
    build: ./services/api-gateway
    container_name: apex-api-gateway
    env_file: .env
    environment:
      PORT: "8080"
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-O", "/dev/null", "-q", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 5

  ingestor:
    build: ./services/ingestor
    container_name: apex-ingestor
    env_file: .env
    environment:
      PORT: "8081"
    ports:
      - "8081:8081"
    depends_on:
      redpanda:
        condition: service_healthy
      minio:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-O", "/dev/null", "-q", "http://localhost:8081/health"]
      interval: 10s
      timeout: 5s
      retries: 5

  event-worker:
    build: ./services/event-worker
    container_name: apex-event-worker
    env_file: .env
    environment:
      PORT: "8082"
    ports:
      - "8082:8082"
    depends_on:
      redpanda:
        condition: service_healthy
      postgres:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-O", "/dev/null", "-q", "http://localhost:8082/health"]
      interval: 10s
      timeout: 5s
      retries: 5

  agent-service:
    build: ./services/agent-service
    container_name: apex-agent-service
    env_file: .env
    environment:
      PORT: "8000"
    ports:
      - "8000:8000"
    depends_on:
      postgres:
        condition: service_healthy
      neo4j:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-O", "/dev/null", "-q", "http://localhost:8000/health"]
      interval: 10s
      timeout: 5s
      retries: 5

  frontend:
    build: ./frontend
    container_name: apex-frontend
    environment:
      NODE_ENV: production
      NEXT_PUBLIC_API_URL: http://localhost:8080
    ports:
      - "3000:3000"
    depends_on:
      api-gateway:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-O", "/dev/null", "-q", "http://localhost:3000/api/health"]
      interval: 10s
      timeout: 5s
      retries: 5
```

- [ ] **Step 3: Build and start everything**

```bash
make build
make up
```

Wait ~2 minutes for all builds + Neo4j startup.

- [ ] **Step 4: Verify all services healthy**

```bash
docker-compose ps
```

Expected: all 10 containers show `healthy` or `exited 0` (for init containers).

```bash
curl http://localhost:8080/health
curl http://localhost:8081/health
curl http://localhost:8082/health
curl http://localhost:8000/health
curl http://localhost:3000/api/health
```

Each expected to return: `{"status":"ok","service":"<name>"}`

- [ ] **Step 5: Commit**

```bash
git add go.work docker-compose.yml
git commit -m "chore: wire all services into docker-compose, add go.work"
```

---

## Phase 1 Complete

`docker-compose up` → all 5 services healthy + all DB tables exist + Kafka topics created + MinIO bucket exists.

**Next plans (in order):**
1. `2026-04-26-apex-phase2-ingest-pipeline.md` — Gmail OAuth + Telegram webhook → Kafka `invoice.raw`
2. `2026-04-26-apex-phase3-event-worker.md` — OCR + PO matching + idempotency → Kafka `invoice.processed`
3. `2026-04-26-apex-phase4-agent-core.md` — Groq ReAct loop + fraud graph + Merkle audit + SSE streaming
4. `2026-04-26-apex-phase5-agent-advanced.md` — Feedback loop + policy engine
5. `2026-04-26-apex-phase6-api-gateway.md` — JWT auth + RBAC + REST API + WebSocket
6. `2026-04-26-apex-phase7-frontend.md` — Full dashboard + invoice detail + audit replay + fraud graph panel
