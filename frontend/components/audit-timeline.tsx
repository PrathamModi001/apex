'use client'

import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { ShieldCheck, ShieldAlert, ChevronDown, ChevronRight, X } from 'lucide-react'

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

function eventDotColor(type: string): string {
  if (type.startsWith('AGENT_STEP')) return 'var(--apex-blue)'
  const map: Record<string, string> = {
    INGESTED:  'var(--apex-blue)',
    PROCESSED: 'var(--apex-amber)',
    DECISION:  'var(--apex-green)',
    FLAGGED:   'var(--apex-orange)',
    REJECTED:  'var(--apex-red)',
  }
  return map[type] ?? '#3A3E50'
}

export function AuditTimeline({ events }: AuditTimelineProps) {
  const [expanded, setExpanded]       = useState<Set<string>>(new Set())
  const [verifyResult, setVerifyResult] = useState<VerifyResult | null>(null)
  const [verifying, setVerifying]     = useState(false)

  const toggle = (id: string) =>
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })

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

  return (
    <div className="rounded-sm border border-border bg-card overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-5 pt-4 pb-3 border-b border-border/60">
        <p className="text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground">
          Audit Trail
        </p>
        <div className="flex items-center gap-3">
          {verifyResult && (
            <span
              className="flex items-center gap-1.5 text-[10px] font-mono"
              style={{ color: verifyResult.intact ? 'var(--apex-green)' : 'var(--apex-red)' }}
            >
              {verifyResult.intact
                ? <ShieldCheck className="h-3 w-3" strokeWidth={2} />
                : <ShieldAlert className="h-3 w-3" strokeWidth={2} />}
              {verifyResult.intact
                ? `Intact · ${verifyResult.steps} steps`
                : `Tampered at step ${verifyResult.tamperedAt}`}
            </span>
          )}
          <Button
            size="sm"
            variant="outline"
            onClick={verifyIntegrity}
            disabled={verifying || events.length === 0}
            className="h-6 text-[10px] font-mono"
          >
            {verifying ? 'Verifying…' : 'Verify Chain'}
          </Button>
        </div>
      </div>

      {events.length === 0 ? (
        <div className="px-5 py-10 text-center">
          <p className="text-[11px] font-mono text-muted-foreground/40">No audit events found.</p>
        </div>
      ) : (
        <div>
          {/* Horizontal step scrubber */}
          <div className="relative flex items-start gap-0 overflow-x-auto px-5 py-4 border-b border-border/60">
            <div
              className="absolute top-[28px] left-0 right-0 h-px"
              style={{ background: 'var(--border)' }}
            />
            {events.map((ev, i) => {
              const color = eventDotColor(ev.event_type)
              const isExp = expanded.has(ev.id)
              return (
                <button
                  key={ev.id}
                  className="relative flex flex-col items-center gap-1.5 min-w-[72px] cursor-pointer group transition-opacity"
                  onClick={() => toggle(ev.id)}
                  style={{ opacity: isExp ? 1 : 0.7 }}
                >
                  <span
                    className="relative h-[10px] w-[10px] rounded-full transition-transform group-hover:scale-125"
                    style={{
                      background: color,
                      boxShadow: isExp ? `0 0 8px ${color}` : 'none',
                      outline: isExp ? `2px solid ${color}40` : 'none',
                      outlineOffset: '2px',
                    }}
                  />
                  <span className="text-[8px] font-mono text-muted-foreground text-center leading-tight px-1 max-w-[68px]">
                    {ev.event_type.replace(/_/g, ' ')}
                  </span>
                  <span className="text-[8px] font-mono text-muted-foreground/40">#{i + 1}</span>
                </button>
              )
            })}
          </div>

          {/* Detail cards for expanded events */}
          {events.some((ev) => expanded.has(ev.id)) && (
            <div className="divide-y divide-border/50">
              {events.map((ev, i) =>
                expanded.has(ev.id) ? (
                  <div key={ev.id} className="px-5 py-4 space-y-3">
                    {/* Row header */}
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span
                          className="inline-block w-1.5 h-4 rounded-full"
                          style={{ background: eventDotColor(ev.event_type) }}
                        />
                        <span className="text-xs font-mono font-semibold text-foreground">
                          #{i + 1} {ev.event_type}
                        </span>
                      </div>
                      <button
                        className="text-muted-foreground/50 hover:text-foreground transition-colors"
                        onClick={() => toggle(ev.id)}
                      >
                        <X className="h-3.5 w-3.5" />
                      </button>
                    </div>

                    <div className="flex gap-6 text-[10px] font-mono text-muted-foreground">
                      <span>
                        Actor: <span className="text-foreground">{ev.actor || '—'}</span>
                      </span>
                      <span>
                        Time:{' '}
                        <span className="text-foreground">
                          {new Date(ev.created_at).toLocaleString()}
                        </span>
                      </span>
                    </div>

                    <details className="group/det">
                      <summary className="cursor-pointer text-[10px] font-mono text-muted-foreground hover:text-foreground flex items-center gap-1 list-none">
                        <ChevronRight className="h-3 w-3 group-open/det:rotate-90 transition-transform" />
                        Payload
                      </summary>
                      <pre
                        className="mt-1.5 text-[10px] font-mono p-3 rounded-sm overflow-auto max-h-48 text-foreground/70"
                        style={{ background: 'var(--background)', border: '1px solid var(--border)' }}
                      >
                        {JSON.stringify(ev.payload, null, 2)}
                      </pre>
                    </details>

                    <p className="text-[9px] font-mono text-muted-foreground/40 truncate">
                      hash: {ev.chain_hash}
                    </p>
                  </div>
                ) : null
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
