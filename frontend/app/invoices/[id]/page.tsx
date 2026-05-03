'use client'

import { useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { RiskBadge } from '@/components/risk-badge'
import { SseReasoningPanel } from '@/components/sse-reasoning-panel'
import { FraudGraph } from '@/components/fraud-graph'
import { AuditTimeline } from '@/components/audit-timeline'
import { fetchInvoice, fetchDecision, fetchAuditChain, approveInvoice, rejectInvoice } from '@/lib/api'

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

function statusVariant(status: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (status) {
    case 'AUTO_APPROVE': return 'default'
    case 'FLAGGED': return 'secondary'
    case 'REJECTED': return 'destructive'
    default: return 'outline'
  }
}

function FieldRow({ label, value }: { label: string; value?: string | number | null }) {
  return (
    <div className="flex items-center justify-between py-1.5 border-b border-border/50 last:border-0">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-xs font-medium text-foreground">{value ?? '—'}</span>
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
    if (!rejectReason.trim()) {
      setShowRejectInput(true)
      return
    }
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

  const isLoading = invoiceLoading || decisionLoading

  if (isLoading) {
    return (
      <div className="p-6 space-y-6">
        <Skeleton className="h-8 w-64" />
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <Skeleton className="h-48" />
          <Skeleton className="h-48" />
        </div>
        <Skeleton className="h-64" />
      </div>
    )
  }

  if (!invoice) {
    return (
      <div className="p-6">
        <Button variant="outline" size="sm" onClick={() => router.back()} className="mb-4">
          ← Back
        </Button>
        <p className="text-muted-foreground">Invoice not found.</p>
      </div>
    )
  }

  const status = invoice.status
  const riskScore = decision?.risk_score ?? invoice.risk_score ?? 0
  const riskDesc = decision?.risk_description ?? invoice.risk_description
  const poMatch = decision?.po_match ?? invoice.po_match
  const fields = invoice.extracted_fields ?? {}
  const canAct = status === 'FLAGGED' || status === 'PROCESSING'

  return (
    <div className="p-6 space-y-6">
      {/* Back button */}
      <Button variant="outline" size="sm" onClick={() => router.push('/invoices')}>
        ← Back to Invoices
      </Button>

      {/* Header */}
      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div className="space-y-1">
          <h1 className="text-2xl font-bold">{invoice.vendor_name}</h1>
          <p className="text-xs text-muted-foreground font-mono">{invoice.id}</p>
        </div>
        <div className="flex items-center gap-3 flex-wrap">
          <Badge variant={statusVariant(status)}>{status}</Badge>
          <RiskBadge score={riskScore} />
          {canAct && (
            <div className="flex items-center gap-2">
              <Button
                size="sm"
                variant="default"
                disabled={actionLoading !== null}
                onClick={handleApprove}
              >
                {actionLoading === 'approve' ? 'Approving…' : 'Approve'}
              </Button>
              <Button
                size="sm"
                variant="destructive"
                disabled={actionLoading !== null}
                onClick={() => setShowRejectInput((v) => !v)}
              >
                Reject
              </Button>
            </div>
          )}
        </div>
      </div>

      {/* Reject input */}
      {showRejectInput && (
        <Card>
          <CardContent className="pt-4">
            <div className="flex items-center gap-2">
              <input
                className="flex-1 h-8 rounded-lg border border-input bg-background px-3 text-sm text-foreground outline-none focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/50"
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
                {actionLoading === 'reject' ? 'Rejecting…' : 'Confirm Reject'}
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => { setShowRejectInput(false); setRejectReason('') }}
              >
                Cancel
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {actionError && (
        <p className="text-sm text-destructive">{actionError}</p>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Extracted fields */}
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Extracted Fields</CardTitle>
          </CardHeader>
          <CardContent>
            <FieldRow label="Invoice No." value={fields.invoice_no as string} />
            <FieldRow label="Amount" value={fields.amount != null ? String(fields.amount) : null} />
            <FieldRow label="Currency" value={fields.currency as string} />
            <FieldRow label="Due Date" value={fields.due_date as string} />
            <FieldRow label="Vendor Name" value={fields.vendor_name as string ?? invoice.vendor_name} />
          </CardContent>
        </Card>

        {/* PO Match */}
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">PO Match</CardTitle>
          </CardHeader>
          <CardContent>
            <FieldRow
              label="Matched"
              value={poMatch?.matched == null ? '—' : poMatch.matched ? 'Yes' : 'No'}
            />
            <FieldRow label="PO ID" value={poMatch?.po_id} />
            <FieldRow
              label="Confidence"
              value={poMatch?.confidence != null ? `${(poMatch.confidence * 100).toFixed(0)}%` : null}
            />
          </CardContent>
        </Card>

        {/* Risk Breakdown */}
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Risk Breakdown</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-3 mb-3">
              <span className="text-4xl font-bold tabular-nums">{riskScore.toFixed(0)}</span>
              <RiskBadge score={riskScore} />
            </div>
            {riskDesc ? (
              <p className="text-xs text-muted-foreground leading-relaxed">{riskDesc}</p>
            ) : (
              <p className="text-xs text-muted-foreground">No risk description available.</p>
            )}
          </CardContent>
        </Card>
      </div>

      {/* SSE Reasoning Panel */}
      <SseReasoningPanel invoiceId={id} />

      {/* Fraud Graph */}
      <FraudGraph invoiceId={id} />

      {/* Audit Trail */}
      {auditLoading ? (
        <Skeleton className="h-48" />
      ) : (
        <AuditTimeline events={auditEvents ?? []} />
      )}
    </div>
  )
}
