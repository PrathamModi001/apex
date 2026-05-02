-- Performance indexes for common query patterns

-- audit_log: heavy query on invoice_id for audit trail lookups
CREATE INDEX idx_audit_log_invoice_id ON audit_log(invoice_id);

-- invoices: FK join on vendor_id
CREATE INDEX idx_invoices_vendor_id ON invoices(vendor_id);

-- invoices: status filtering (show all INGESTED, show all FLAGGED, etc.)
CREATE INDEX idx_invoices_status ON invoices(status);

-- invoices: ordering by created_at in live feed
CREATE INDEX idx_invoices_created_at ON invoices(created_at DESC);

-- feedback: lookup by invoice_id for few-shot retrieval
CREATE INDEX idx_feedback_invoice_id ON feedback(invoice_id);
