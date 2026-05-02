CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email            TEXT UNIQUE NOT NULL,
    role             TEXT NOT NULL DEFAULT 'viewer',
    gmail_token      JSONB,
    telegram_user_id TEXT,
    created_at       TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE vendors (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name             TEXT NOT NULL,
    bank_accounts    JSONB NOT NULL DEFAULT '[]',
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
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    invoice_id         UUID NOT NULL REFERENCES invoices(id),
    agent_decision     TEXT NOT NULL,
    human_decision     TEXT NOT NULL,
    correction_payload JSONB DEFAULT '{}',
    actor_id           UUID REFERENCES users(id),
    created_at         TIMESTAMPTZ DEFAULT now()
);
