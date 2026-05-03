'use client'

import { useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { fetchAuditChain } from '@/lib/api'

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

const eventColor: Record<string, string> = {
  INGESTED: 'bg-blue-500 text-blue-100',
  PROCESSED: 'bg-purple-500 text-purple-100',
  DECISION: 'bg-green-500 text-green-100',
  FLAGGED: 'bg-yellow-500 text-yellow-100',
  REJECTED: 'bg-red-500 text-red-100',
}

function getEventColor(type: string): string {
  if (type.startsWith('AGENT_STEP')) return 'bg-cyan-500 text-cyan-100'
  return eventColor[type] ?? 'bg-zinc-500 text-zinc-100'
}

export default function AuditReplayPage() {
  const params = useParams()
  const router = useRouter()
  const id = params.id as string

  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [verifyResult, setVerifyResult] = useState<VerifyResponse | null>(null)
  const [verifying, setVerifying] = useState(false)

  const { data: events, isLoading } = useQuery<AuditEvent[]>({
    queryKey: ['audit', id],
    queryFn: () => fetchAuditChain(id),
  })

  const toggleExpand = (evId: string) => {
    setExpandedId((prev) => (prev === evId ? null : evId))
  }

  const verifyIntegrity = async () => {
    setVerifying(true)
    setVerifyResult(null)
    try {
      const res = await fetch(`${BASE_URL}/invoices/${id}/verify-chain`, {
        method: 'POST',
      })
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
  const invoiceId = allEvents[0]?.payload?.invoice_id as string | undefined ?? id

  return (
    <div className="p-6 space-y-6">
      <Button variant="outline" size="sm" onClick={() => router.push(`/invoices/${id}`)}>
        ← Back to Invoice
      </Button>

      <div className="flex items-center justify-between flex-wrap gap-4">
        <div>
          <h1 className="text-2xl font-bold">Audit Replay</h1>
          <p className="text-xs text-muted-foreground font-mono mt-0.5">{invoiceId}</p>
        </div>
        <div className="flex items-center gap-3">
          {verifyResult && (
            <span
              className={`text-sm font-medium ${
                verifyResult.intact ? 'text-green-400' : 'text-red-400'
              }`}
            >
              {verifyResult.intact ? '✓ Chain intact' : '⚠ Chain tampered'}
              {verifyResult.message ? ` — ${verifyResult.message}` : ''}
            </span>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={verifyIntegrity}
            disabled={verifying || isLoading}
          >
            {verifying ? 'Verifying…' : 'Verify Integrity'}
          </Button>
        </div>
      </div>

      {isLoading ? (
        <Card>
          <CardContent className="pt-4 space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </CardContent>
        </Card>
      ) : allEvents.length === 0 ? (
        <Card>
          <CardContent className="pt-4">
            <p className="text-sm text-muted-foreground text-center py-8">
              No audit events found for this invoice.
            </p>
          </CardContent>
        </Card>
      ) : (
        <>
          {/* Horizontal timeline scrubber */}
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Timeline — {allEvents.length} event{allEvents.length !== 1 ? 's' : ''}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="relative flex items-start gap-0 overflow-x-auto pb-4">
                {/* Connector line */}
                <div className="absolute top-4 left-0 right-0 h-0.5 bg-border z-0" />
                {allEvents.map((ev, i) => (
                  <button
                    key={ev.id}
                    className="relative z-10 flex flex-col items-center gap-1.5 min-w-[88px] cursor-pointer group"
                    onClick={() => toggleExpand(ev.id)}
                  >
                    <span
                      className={`h-8 w-8 rounded-full flex items-center justify-center text-[10px] font-bold border-2 border-background transition-transform group-hover:scale-110 ${getEventColor(ev.event_type)}`}
                    >
                      {i + 1}
                    </span>
                    <span className="text-[9px] text-muted-foreground text-center leading-tight px-1 max-w-[80px]">
                      {ev.event_type.replace(/_/g, ' ')}
                    </span>
                    <span className="text-[8px] text-muted-foreground/60">
                      {new Date(ev.created_at).toLocaleTimeString('en', { hour: '2-digit', minute: '2-digit' })}
                    </span>
                  </button>
                ))}
              </div>
            </CardContent>
          </Card>

          {/* Step detail cards */}
          <div className="space-y-3">
            {allEvents.map((ev, i) => (
              <Card
                key={ev.id}
                className={`cursor-pointer transition-colors ${
                  expandedId === ev.id ? 'ring-1 ring-primary/30' : ''
                }`}
              >
                <button
                  className="w-full text-left"
                  onClick={() => toggleExpand(ev.id)}
                >
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm flex items-center gap-3">
                      <span
                        className={`inline-flex h-6 w-6 rounded-full items-center justify-center text-[10px] font-bold ${getEventColor(ev.event_type)}`}
                      >
                        {i + 1}
                      </span>
                      <span>{ev.event_type}</span>
                      <span className="text-xs text-muted-foreground font-normal ml-auto">
                        {new Date(ev.created_at).toLocaleString()}
                      </span>
                      <span className="text-muted-foreground">
                        {expandedId === ev.id ? '▲' : '▼'}
                      </span>
                    </CardTitle>
                  </CardHeader>
                </button>

                {expandedId === ev.id && (
                  <CardContent className="space-y-3 border-t border-border pt-3">
                    <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
                      <div>
                        <span className="text-muted-foreground">Actor:</span>{' '}
                        <span className="font-medium">{ev.actor || '—'}</span>
                      </div>
                      <div>
                        <span className="text-muted-foreground">Timestamp:</span>{' '}
                        <span className="font-medium">{new Date(ev.created_at).toLocaleString()}</span>
                      </div>
                    </div>

                    <div>
                      <p className="text-xs text-muted-foreground mb-1">Payload</p>
                      <pre className="bg-muted rounded-lg p-3 text-[10px] font-mono overflow-auto max-h-48 text-foreground/80">
                        {JSON.stringify(ev.payload, null, 2)}
                      </pre>
                    </div>

                    <div>
                      <p className="text-[9px] text-muted-foreground/60 font-mono truncate">
                        chain_hash: {ev.chain_hash}
                      </p>
                    </div>
                  </CardContent>
                )}
              </Card>
            ))}
          </div>
        </>
      )}
    </div>
  )
}
