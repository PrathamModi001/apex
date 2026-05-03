'use client'

import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

interface AuditEvent {
  id: string
  event_type: string
  actor: string
  payload: Record<string, unknown>
  prev_hash: string
  chain_hash: string
  created_at: string
}

interface VerifyResult {
  intact: boolean
  tamperedAt?: number
  steps: number
}

interface AuditTimelineProps {
  events: AuditEvent[]
}

async function sha256(data: string): Promise<string> {
  const encoder = new TextEncoder()
  const buf = await crypto.subtle.digest('SHA-256', encoder.encode(data))
  return Array.from(new Uint8Array(buf))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('')
}

function sortedJson(obj: unknown): string {
  if (typeof obj !== 'object' || obj === null) return JSON.stringify(obj)
  if (Array.isArray(obj)) return `[${obj.map(sortedJson).join(',')}]`
  const sorted = Object.keys(obj as Record<string, unknown>)
    .sort()
    .map((k) => `${JSON.stringify(k)}:${sortedJson((obj as Record<string, unknown>)[k])}`)
  return `{${sorted.join(',')}}`
}

export function AuditTimeline({ events }: AuditTimelineProps) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [verifyResult, setVerifyResult] = useState<VerifyResult | null>(null)
  const [verifying, setVerifying] = useState(false)

  const toggle = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const verifyIntegrity = async () => {
    setVerifying(true)
    try {
      let prevHash = ''
      for (let i = 0; i < events.length; i++) {
        const ev = events[i]
        const data = prevHash + sortedJson(ev.payload)
        const computed = await sha256(data)
        if (computed !== ev.chain_hash) {
          setVerifyResult({ intact: false, tamperedAt: i + 1, steps: events.length })
          return
        }
        prevHash = computed
      }
      setVerifyResult({ intact: true, steps: events.length })
    } finally {
      setVerifying(false)
    }
  }

  const statusColor: Record<string, string> = {
    INGESTED: 'bg-blue-500',
    PROCESSED: 'bg-purple-500',
    DECISION: 'bg-green-500',
    FLAGGED: 'bg-yellow-500',
    REJECTED: 'bg-red-500',
  }

  const getColor = (type: string) => {
    if (type.startsWith('AGENT_STEP')) return 'bg-cyan-500'
    return statusColor[type] || 'bg-zinc-500'
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="text-sm">Audit Trail</CardTitle>
        <div className="flex items-center gap-2">
          {verifyResult && (
            <span
              className={`text-xs font-medium ${
                verifyResult.intact ? 'text-green-400' : 'text-red-400'
              }`}
            >
              {verifyResult.intact
                ? `Chain intact (${verifyResult.steps} steps)`
                : `Tampered at step ${verifyResult.tamperedAt}`}
            </span>
          )}
          <Button size="sm" variant="outline" onClick={verifyIntegrity} disabled={verifying}>
            {verifying ? 'Verifying…' : 'Verify Integrity'}
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {events.length === 0 ? (
          <p className="text-xs text-muted-foreground">No audit events found.</p>
        ) : (
          <>
            {/* Horizontal timeline dots */}
            <div className="relative flex items-center gap-0 mb-6 overflow-x-auto pb-2">
              <div className="absolute top-3 left-0 right-0 h-0.5 bg-border" />
              {events.map((ev, i) => (
                <button
                  key={ev.id}
                  className="relative z-10 flex flex-col items-center gap-1 min-w-[72px] cursor-pointer group"
                  onClick={() => toggle(ev.id)}
                >
                  <span
                    className={`h-3 w-3 rounded-full border-2 border-background ${getColor(ev.event_type)} group-hover:scale-125 transition-transform`}
                  />
                  <span className="text-[9px] text-muted-foreground text-center leading-tight px-1">
                    {ev.event_type.replace('_', ' ')}
                  </span>
                  <span className="text-[9px] text-muted-foreground/60">#{i + 1}</span>
                </button>
              ))}
            </div>

            {/* Expanded detail cards */}
            <div className="space-y-2">
              {events.map((ev, i) =>
                expanded.has(ev.id) ? (
                  <div
                    key={ev.id}
                    className="rounded-lg border border-border bg-muted/30 p-3 text-xs space-y-1.5"
                  >
                    <div className="flex items-center justify-between">
                      <span className="font-semibold">
                        #{i + 1} {ev.event_type}
                      </span>
                      <button
                        className="text-muted-foreground hover:text-foreground"
                        onClick={() => toggle(ev.id)}
                      >
                        ✕
                      </button>
                    </div>
                    <div className="flex gap-4 text-muted-foreground">
                      <span>Actor: <span className="text-foreground">{ev.actor}</span></span>
                      <span>
                        Time:{' '}
                        <span className="text-foreground">
                          {new Date(ev.created_at).toLocaleString()}
                        </span>
                      </span>
                    </div>
                    <details className="mt-1">
                      <summary className="cursor-pointer text-muted-foreground hover:text-foreground">
                        Payload
                      </summary>
                      <pre className="mt-1 bg-muted p-2 rounded overflow-auto text-[10px] max-h-40">
                        {JSON.stringify(ev.payload, null, 2)}
                      </pre>
                    </details>
                    <div className="font-mono text-[9px] text-muted-foreground/60 truncate">
                      hash: {ev.chain_hash}
                    </div>
                  </div>
                ) : null
              )}
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}
