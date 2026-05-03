'use client'

import { useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { fetchAuditChain } from '@/lib/api'
import { ArrowLeft, ShieldCheck, ShieldAlert, ChevronDown, ChevronRight } from 'lucide-react'

const BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

interface AuditEvent {
  id: string
  event_type: string
  actor: string
  payload: Record<string, unknown>
  prev_hash: string
  chain_hash: string
  created_at: string
}

interface VerifyResponse {
  intact: boolean
  message?: string
}

const EVENT_COLOR: Record<string, string> = {
  INGESTED:  'var(--apex-blue)',
  PROCESSED: 'var(--apex-amber)',
  DECISION:  'var(--apex-green)',
  FLAGGED:   'var(--apex-orange)',
  REJECTED:  'var(--apex-red)',
}

function getEventColor(type: string): string {
  if (type.startsWith('AGENT_STEP')) return 'var(--apex-blue)'
  return EVENT_COLOR[type] ?? '#3A3E50'
}

export default function AuditReplayPage() {
  const params = useParams()
  const router = useRouter()
  const id     = params.id as string

  const [activeId, setActiveId]       = useState<string | null>(null)
  const [verifyResult, setVerifyResult] = useState<VerifyResponse | null>(null)
  const [verifying, setVerifying]     = useState(false)

  const { data: events, isLoading } = useQuery<AuditEvent[]>({
    queryKey: ['audit', id],
    queryFn: () => fetchAuditChain(id),
  })

  const verifyIntegrity = async () => {
    setVerifying(true)
    setVerifyResult(null)
    try {
      const res = await fetch(`${BASE_URL}/invoices/${id}/verify-chain`, { method: 'POST' })
      if (!res.ok) throw new Error('Verify request failed')
      const data = await res.json()
      setVerifyResult(data)
    } catch {
      setVerifyResult({ intact: false, message: 'Could not verify chain — request failed.' })
    } finally {
      setVerifying(false)
    }
  }

  const allEvents = events ?? []
  const invoiceId = (allEvents[0]?.payload?.invoice_id as string | undefined) ?? id
  const activeEvent = allEvents.find((ev) => ev.id === activeId) ?? null
  const activeIndex = allEvents.findIndex((ev) => ev.id === activeId)

  return (
    <div className="p-6 space-y-4 max-w-[1280px]">

      {/* Back */}
      <div className="apex-fade-up">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => router.push(`/invoices/${id}`)}
          className="gap-1.5 -ml-1"
        >
          <ArrowLeft className="h-3 w-3" /> Invoice
        </Button>
      </div>

      {/* Header */}
      <div className="apex-fade-up apex-delay-1 flex items-start justify-between flex-wrap gap-4">
        <div>
          <p className="text-[9px] font-mono tracking-[0.25em] uppercase text-muted-foreground mb-1">
            Audit Replay
          </p>
          <h1
            className="text-2xl font-bold leading-none tracking-tight text-foreground"
            style={{ fontFamily: 'var(--font-syne, sans-serif)' }}
          >
            Chain Replay
          </h1>
          <p className="mt-1 text-[10px] font-mono text-muted-foreground/60 tracking-wider">
            {invoiceId}
          </p>
        </div>

        <div className="flex items-center gap-3">
          {verifyResult && (
            <span
              className="flex items-center gap-1.5 text-xs font-mono"
              style={{ color: verifyResult.intact ? 'var(--apex-green)' : 'var(--apex-red)' }}
            >
              {verifyResult.intact
                ? <ShieldCheck className="h-3.5 w-3.5" strokeWidth={2} />
                : <ShieldAlert className="h-3.5 w-3.5" strokeWidth={2} />}
              {verifyResult.intact ? 'Chain intact' : 'Chain tampered'}
              {verifyResult.message ? ` — ${verifyResult.message}` : ''}
            </span>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={verifyIntegrity}
            disabled={verifying || isLoading || allEvents.length === 0}
            className="gap-1.5 text-[11px]"
          >
            <ShieldCheck className="h-3 w-3" />
            {verifying ? 'Verifying…' : 'Verify Integrity'}
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-14 w-full rounded-sm" />
          ))}
        </div>
      ) : allEvents.length === 0 ? (
        <div className="rounded-sm border border-border bg-card px-5 py-12 text-center">
          <p className="text-[11px] font-mono text-muted-foreground/50">
            No audit events found for this invoice.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-[320px_1fr] gap-3 apex-fade-up apex-delay-2">

          {/* Left: vertical timeline */}
          <div className="rounded-sm border border-border bg-card overflow-hidden">
            <div className="px-4 pt-4 pb-3 border-b border-border/60">
              <p className="text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground">
                Timeline · {allEvents.length} events
              </p>
            </div>
            <div className="relative">
              {/* Vertical connector line */}
              <div
                className="absolute left-[27px] top-0 bottom-0 w-px"
                style={{ background: 'var(--border)' }}
              />
              <div className="divide-y divide-border/40">
                {allEvents.map((ev, i) => {
                  const color = getEventColor(ev.event_type)
                  const isActive = activeId === ev.id
                  return (
                    <button
                      key={ev.id}
                      className="w-full flex items-start gap-3 px-4 py-3 text-left transition-colors group"
                      style={{ background: isActive ? 'var(--apex-amber-glow)' : undefined }}
                      onClick={() => setActiveId(ev.id === activeId ? null : ev.id)}
                    >
                      {/* Dot + number */}
                      <div className="relative flex-shrink-0 flex flex-col items-center pt-0.5">
                        <span
                          className="relative z-10 w-5 h-5 rounded-sm flex items-center justify-center text-[8px] font-mono font-bold"
                          style={{
                            background: isActive ? color : 'var(--card)',
                            border: `1.5px solid ${color}`,
                            color: isActive ? 'var(--background)' : color,
                          }}
                        >
                          {i + 1}
                        </span>
                      </div>
                      {/* Info */}
                      <div className="flex-1 min-w-0">
                        <p
                          className="text-[11px] font-mono font-semibold truncate"
                          style={{ color: isActive ? 'var(--apex-amber-bright)' : 'var(--foreground)' }}
                        >
                          {ev.event_type.replace(/_/g, ' ')}
                        </p>
                        <p className="text-[9px] font-mono text-muted-foreground/60 mt-0.5">
                          {new Date(ev.created_at).toLocaleTimeString('en', {
                            hour: '2-digit',
                            minute: '2-digit',
                            second: '2-digit',
                          })}
                        </p>
                      </div>
                      {isActive
                        ? <ChevronDown className="h-3 w-3 text-muted-foreground/50 flex-shrink-0 mt-1" />
                        : <ChevronRight className="h-3 w-3 text-muted-foreground/30 group-hover:text-muted-foreground/60 flex-shrink-0 mt-1 transition-colors" />}
                    </button>
                  )
                })}
              </div>
            </div>
          </div>

          {/* Right: event detail */}
          <div className="rounded-sm border border-border bg-card overflow-hidden">
            {!activeEvent ? (
              <div className="flex flex-col items-center justify-center h-full min-h-[300px] text-center px-8">
                <div
                  className="w-12 h-12 rounded-sm flex items-center justify-center mb-3"
                  style={{ background: 'var(--muted)', border: '1px solid var(--border)' }}
                >
                  <ShieldCheck className="h-5 w-5 text-muted-foreground/30" strokeWidth={1} />
                </div>
                <p className="text-[11px] font-mono text-muted-foreground/50">
                  Select an event to inspect
                </p>
              </div>
            ) : (
              <div>
                {/* Event header */}
                <div
                  className="px-5 pt-4 pb-3 border-b border-border/60"
                  style={{ borderLeftWidth: 3, borderLeftColor: getEventColor(activeEvent.event_type) }}
                >
                  <div className="flex items-center gap-2 mb-1">
                    <span
                      className="text-[9px] font-mono tracking-[0.15em] uppercase px-1.5 py-0.5 rounded-[2px]"
                      style={{
                        background: `${getEventColor(activeEvent.event_type)}20`,
                        color: getEventColor(activeEvent.event_type),
                        border: `1px solid ${getEventColor(activeEvent.event_type)}40`,
                      }}
                    >
                      {activeEvent.event_type}
                    </span>
                    <span className="text-[10px] font-mono text-muted-foreground/50">
                      Step {activeIndex + 1}
                    </span>
                  </div>
                  <div className="flex gap-6 text-[10px] font-mono text-muted-foreground mt-2">
                    <span>Actor: <span className="text-foreground">{activeEvent.actor || '—'}</span></span>
                    <span>Time: <span className="text-foreground">{new Date(activeEvent.created_at).toLocaleString()}</span></span>
                  </div>
                </div>

                {/* Payload */}
                <div className="px-5 py-4">
                  <p className="text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 mb-2">
                    Payload
                  </p>
                  <pre
                    className="text-[10px] font-mono p-3 rounded-sm overflow-auto max-h-72 text-foreground/70 leading-relaxed"
                    style={{ background: 'var(--background)', border: '1px solid var(--border)' }}
                  >
                    {JSON.stringify(activeEvent.payload, null, 2)}
                  </pre>
                </div>

                {/* Hash info */}
                <div className="px-5 pb-4 space-y-1.5">
                  <p className="text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60">
                    Chain Hashes
                  </p>
                  <div className="space-y-1">
                    <p className="text-[9px] font-mono text-muted-foreground/40 truncate">
                      <span className="text-muted-foreground/60 mr-1">prev:</span>
                      {activeEvent.prev_hash || '0'.repeat(16) + '…'}
                    </p>
                    <p className="text-[9px] font-mono truncate" style={{ color: getEventColor(activeEvent.event_type), opacity: 0.8 }}>
                      <span className="text-muted-foreground/60 mr-1">curr:</span>
                      {activeEvent.chain_hash}
                    </p>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
