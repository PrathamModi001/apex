'use client'

import { useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { RiskBadge, RiskBadgeLarge } from '@/components/risk-badge'
import { SseReasoningPanel } from '@/components/sse-reasoning-panel'
import { FraudGraph } from '@/components/fraud-graph'
import { AuditTimeline } from '@/components/audit-timeline'
import { fetchInvoice, fetchDecision, fetchAuditChain, approveInvoice, rejectInvoice } from '@/lib/api'
import { ArrowLeft, CheckCheck, XCircle } from 'lucide-react'

interface ExtractedFields {
  invoice_no?: string
  amount?: number
  currency?: string
  due_date?: string
  vendor_name?: string
  [key: string]: unknown
}

interface PoMatch {
  matched?: boolean
  po_id?: string
  confidence?: number
}

interface Invoice {
  id: string
  vendor_name: string
  status: string
  risk_score: number
  source: string
  received_at: string
  extracted_fields?: ExtractedFields
  po_match?: PoMatch
  risk_description?: string
}

interface Decision {
  decision?: string
  risk_score?: number
  risk_description?: string
  po_match?: PoMatch
}

interface AuditEvent {
  id: string
  event_type: string
  actor: string
  payload: Record<string, unknown>
  prev_hash: string
  chain_hash: string
  created_at: string
}

function statusBadgeVariant(status: string): 'default' | 'success' | 'secondary' | 'destructive' | 'outline' | 'info' {
  switch (status) {
    case 'AUTO_APPROVE': return 'success'
    case 'FLAGGED':      return 'secondary'
    case 'REJECTED':     return 'destructive'
    case 'PROCESSING':   return 'info'
    default:             return 'outline'
  }
}

function FieldRow({ label, value }: { label: string; value?: string | number | null }) {
  return (
    <div className="flex items-center justify-between py-2 border-b border-border/50 last:border-0">
      <span className="text-[10px] font-mono tracking-wide text-muted-foreground/70 uppercase">{label}</span>
      <span className="text-xs font-mono text-foreground/90 max-w-[160px] truncate text-right">
        {value ?? <span className="text-muted-foreground/40">—</span>}
      </span>
    </div>
  )
}

export default function InvoiceDetailPage() {
  const params = useParams()
  const router = useRouter()
  const id = params.id as string
  const queryClient = useQueryClient()

  const [actionLoading, setActionLoading] = useState<'approve' | 'reject' | null>(null)
  const [rejectReason, setRejectReason] = useState('')
  const [showRejectInput, setShowRejectInput] = useState(false)
  const [actionError, setActionError] = useState<string | null>(null)

  const { data: invoice, isLoading: invoiceLoading } = useQuery<Invoice | null>({
    queryKey: ['invoice', id],
    queryFn: () => fetchInvoice(id),
    refetchInterval: 15_000,
  })

  const { data: decision, isLoading: decisionLoading } = useQuery<Decision | null>({
    queryKey: ['decision', id],
    queryFn: () => fetchDecision(id),
  })

  const { data: auditEvents, isLoading: auditLoading } = useQuery<AuditEvent[]>({
    queryKey: ['audit', id],
    queryFn: () => fetchAuditChain(id),
  })

  const handleApprove = async () => {
    setActionLoading('approve')
    setActionError(null)
    try {
      await approveInvoice(id)
      queryClient.invalidateQueries({ queryKey: ['invoice', id] })
      queryClient.invalidateQueries({ queryKey: ['invoices'] })
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to approve')
    } finally {
      setActionLoading(null)
    }
  }

  const handleReject = async () => {
    if (!rejectReason.trim()) { setShowRejectInput(true); return }
    setActionLoading('reject')
    setActionError(null)
    try {
      await rejectInvoice(id, rejectReason)
      queryClient.invalidateQueries({ queryKey: ['invoice', id] })
      queryClient.invalidateQueries({ queryKey: ['invoices'] })
      setShowRejectInput(false)
      setRejectReason('')
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to reject')
    } finally {
      setActionLoading(null)
    }
  }

  if (invoiceLoading || decisionLoading) {
    return (
      <div className="p-6 space-y-4">
        <Skeleton className="h-6 w-36 rounded-sm" />
        <Skeleton className="h-12 w-96 rounded-sm" />
        <div className="grid grid-cols-3 gap-3">
          <Skeleton className="h-44 rounded-sm" />
          <Skeleton className="h-44 rounded-sm" />
          <Skeleton className="h-44 rounded-sm" />
        </div>
        <Skeleton className="h-64 rounded-sm" />
      </div>
    )
  }

  if (!invoice) {
    return (
      <div className="p-6">
        <Button variant="outline" size="sm" onClick={() => router.back()} className="mb-4 gap-1.5">
          <ArrowLeft className="h-3 w-3" /> Back
        </Button>
        <p className="text-sm text-muted-foreground">Invoice not found.</p>
      </div>
    )
  }

  const status    = invoice.status
  const riskScore = decision?.risk_score ?? invoice.risk_score ?? 0
  const riskDesc  = decision?.risk_description ?? invoice.risk_description
  const poMatch   = decision?.po_match ?? invoice.po_match
  const fields    = invoice.extracted_fields ?? {}
  const canAct    = status === 'FLAGGED' || status === 'PROCESSING'

  return (
    <div className="p-6 space-y-4 max-w-[1280px]">

      {/* Breadcrumb + back */}
      <div className="apex-fade-up">
        <Button variant="ghost" size="sm" onClick={() => router.push('/invoices')} className="gap-1.5 -ml-1">
          <ArrowLeft className="h-3 w-3" />
          Invoices
        </Button>
      </div>

      {/* Header */}
      <div className="apex-fade-up apex-delay-1 rounded-sm border border-border bg-card px-5 py-4">
        <div className="flex items-start justify-between gap-4 flex-wrap">
          <div>
            <p className="text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground mb-1">
              Invoice Detail
            </p>
            <h1
              className="text-xl font-bold leading-tight text-foreground"
              style={{ fontFamily: 'var(--font-syne, sans-serif)' }}
            >
              {invoice.vendor_name}
            </h1>
            <p className="mt-1 font-mono text-[10px] text-muted-foreground/60 tracking-wider">
              {invoice.id}
            </p>
          </div>

          <div className="flex items-center gap-3 flex-wrap">
            <Badge variant={statusBadgeVariant(status)} className="text-[10px]">{status}</Badge>
            <RiskBadge score={riskScore} />

            {canAct && (
              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  variant="success"
                  disabled={actionLoading !== null}
                  onClick={handleApprove}
                  className="gap-1.5"
                >
                  <CheckCheck className="h-3 w-3" />
                  {actionLoading === 'approve' ? 'Approving…' : 'Approve'}
                </Button>
                <Button
                  size="sm"
                  variant="destructive"
                  disabled={actionLoading !== null}
                  onClick={() => setShowRejectInput((v) => !v)}
                  className="gap-1.5"
                >
                  <XCircle className="h-3 w-3" />
                  Reject
                </Button>
              </div>
            )}
          </div>
        </div>

        {/* Reject input */}
        {showRejectInput && (
          <div className="mt-4 flex items-center gap-2 pt-4 border-t border-border/60">
            <input
              className="flex-1 h-8 rounded-sm border border-input bg-[var(--input)] px-3 text-xs font-mono text-foreground outline-none focus-visible:border-[var(--apex-amber)] focus-visible:ring-1 focus-visible:ring-[var(--apex-amber)]/30"
              placeholder="Reason for rejection…"
              value={rejectReason}
              onChange={(e) => setRejectReason(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleReject()}
            />
            <Button
              size="sm"
              variant="destructive"
              disabled={actionLoading !== null || !rejectReason.trim()}
              onClick={handleReject}
            >
              {actionLoading === 'reject' ? 'Rejecting…' : 'Confirm'}
            </Button>
            <Button size="sm" variant="outline" onClick={() => { setShowRejectInput(false); setRejectReason('') }}>
              Cancel
            </Button>
          </div>
        )}

        {actionError && (
          <p className="mt-2 text-xs font-mono text-[var(--apex-red)]">{actionError}</p>
        )}
      </div>

      {/* 3 info cards */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-3 apex-fade-up apex-delay-2">
        {/* Extracted fields */}
        <Card>
          <CardHeader>
            <CardTitle className="text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground">
              Extracted Fields
            </CardTitle>
          </CardHeader>
          <CardContent>
            <FieldRow label="Invoice No." value={fields.invoice_no as string} />
            <FieldRow label="Amount"      value={fields.amount != null ? String(fields.amount) : null} />
            <FieldRow label="Currency"    value={fields.currency as string} />
            <FieldRow label="Due Date"    value={fields.due_date as string} />
            <FieldRow label="Vendor"      value={(fields.vendor_name as string) ?? invoice.vendor_name} />
          </CardContent>
        </Card>

        {/* PO Match */}
        <Card>
          <CardHeader>
            <CardTitle className="text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground">
              PO Match
            </CardTitle>
          </CardHeader>
          <CardContent>
            <FieldRow
              label="Matched"
              value={
                poMatch?.matched == null ? null
                : poMatch.matched ? 'Yes'
                : 'No'
              }
            />
            <FieldRow label="PO ID"      value={poMatch?.po_id} />
            <FieldRow
              label="Confidence"
              value={poMatch?.confidence != null ? `${(poMatch.confidence * 100).toFixed(0)}%` : null}
            />
          </CardContent>
        </Card>

        {/* Risk */}
        <Card accent>
          <CardHeader>
            <CardTitle className="text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground">
              Risk Assessment
            </CardTitle>
          </CardHeader>
          <CardContent>
            <RiskBadgeLarge score={riskScore} />
            {riskDesc && (
              <p className="mt-3 text-[11px] text-muted-foreground leading-relaxed border-t border-border/60 pt-3">
                {riskDesc}
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Agent reasoning */}
      <div className="apex-fade-up apex-delay-3">
        <SseReasoningPanel invoiceId={id} />
      </div>

      {/* Fraud graph */}
      <div className="apex-fade-up apex-delay-4">
        <FraudGraph invoiceId={id} />
      </div>

      {/* Audit trail */}
      <div className="apex-fade-up apex-delay-4">
        {auditLoading ? (
          <Skeleton className="h-48 rounded-sm" />
        ) : (
          <AuditTimeline events={auditEvents ?? []} />
        )}
      </div>
    </div>
  )
}
