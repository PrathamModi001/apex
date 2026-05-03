-- Migration 004: Add missing columns and tables needed by service implementations

-- -----------------------------------------------------------------------
-- users: add name column
-- -----------------------------------------------------------------------
ALTER TABLE users ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';

-- -----------------------------------------------------------------------
-- invoices: add columns written by event-worker and api-gateway
-- -----------------------------------------------------------------------
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS sender TEXT;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS received_at TIMESTAMPTZ DEFAULT now();
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS invoice_no TEXT;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS po_confidence NUMERIC(5,4);
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS po_matched BOOLEAN DEFAULT false;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS processed_at TIMESTAMPTZ;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS decided_at TIMESTAMPTZ;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS approved_by TEXT;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS approved_at TIMESTAMPTZ;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS rejected_by TEXT;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS reject_reason TEXT;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS rejected_at TIMESTAMPTZ;

-- -----------------------------------------------------------------------
-- invoice_decisions: full agent decision record (written by api-gateway
-- Kafka consumer when it receives invoice.decision events)
-- -----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS invoice_decisions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    invoice_id      UUID NOT NULL REFERENCES invoices(id),
    decision        TEXT NOT NULL,
    risk_score      NUMERIC(5,2),
    reasoning_steps JSONB DEFAULT '[]',
    audit_hash      TEXT,
    decided_at      TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_invoice_decisions_invoice_id
    ON invoice_decisions(invoice_id);

-- -----------------------------------------------------------------------
-- vendors: ensure indexed columns exist (already in 001, just safe-guard)
-- -----------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_invoices_received_at
    ON invoices(received_at DESC);
