'use client'

import { useState, useEffect, useRef } from 'react'
import { ChevronDown, ChevronRight, Cpu, Zap } from 'lucide-react'

interface AgentStep {
  id: number
  thought: string
  action?: string
  actionInput?: string
  observation?: string
  complete: boolean
}

interface SseReasoningPanelProps {
  invoiceId: string
  agentServiceUrl?: string
}

export function SseReasoningPanel({ invoiceId, agentServiceUrl }: SseReasoningPanelProps) {
  const [tokens, setTokens]           = useState<string>('')
  const [steps, setSteps]             = useState<AgentStep[]>([])
  const [connected, setConnected]     = useState(false)
  const [expanded, setExpanded]       = useState<Set<number>>(new Set())
  const esRef                         = useRef<EventSource | null>(null)
  const stepIdRef                     = useRef(0)

  const baseUrl = agentServiceUrl || process.env.NEXT_PUBLIC_AGENT_SERVICE_URL || 'http://localhost:8000'

  useEffect(() => {
    const url = `${baseUrl}/stream/invoice/${invoiceId}`
    const es  = new EventSource(url)
    esRef.current = es
    setConnected(true)

    es.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (data.type === 'token') {
          setTokens((prev) => prev + data.content)
        } else if (data.type === 'step_complete') {
          const id = stepIdRef.current++
          setSteps((prev) => [
            ...prev,
            {
              id,
              thought:      data.thought || '',
              action:       data.action,
              actionInput:  data.action_input,
              observation:  data.observation,
              complete:     true,
            },
          ])
          setTokens('')
        }
      } catch {
        setTokens((prev) => prev + event.data)
      }
    }

    es.onerror = () => {
      setConnected(false)
      es.close()
    }

    return () => {
      es.close()
      esRef.current = null
    }
  }, [invoiceId, baseUrl])

  const toggle = (id: number) =>
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })

  const hasContent = steps.length > 0 || tokens !== ''

  return (
    <div className="rounded-sm border border-border bg-card overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-5 pt-4 pb-3 border-b border-border/60">
        <div className="flex items-center gap-2">
          <Cpu className="h-3.5 w-3.5 text-muted-foreground" strokeWidth={1.5} />
          <p className="text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground">
            Agent Reasoning
          </p>
        </div>
        <span className="flex items-center gap-1.5 text-[10px] font-mono text-muted-foreground/60">
          <span
            className={`inline-block w-[5px] h-[5px] rounded-full ${connected ? 'apex-blink' : ''}`}
            style={{
              background: connected ? 'var(--apex-green)' : 'var(--border)',
              boxShadow: connected ? '0 0 4px var(--apex-green)' : 'none',
            }}
          />
          {connected ? 'streaming' : 'idle'}
        </span>
      </div>

      <div className="divide-y divide-border/50">
        {/* Completed steps */}
        {steps.map((step) => {
          const isOpen = expanded.has(step.id)
          return (
            <div key={step.id}>
              <button
                className="w-full flex items-center gap-3 px-5 py-3 text-left hover:bg-[var(--apex-amber-glow)] transition-colors group"
                onClick={() => toggle(step.id)}
              >
                {/* Step number */}
                <span
                  className="flex-shrink-0 w-5 h-5 rounded-sm flex items-center justify-center text-[9px] font-mono font-bold"
                  style={{
                    background: 'var(--apex-amber-glow)',
                    color: 'var(--apex-amber-bright)',
                    border: '1px solid var(--apex-amber-line)',
                  }}
                >
                  {step.id + 1}
                </span>
                <span className="flex-1 text-xs font-mono text-foreground/80 truncate">
                  {step.action
                    ? <><span style={{ color: 'var(--apex-amber)' }}>{step.action}</span> — {step.thought.slice(0, 60)}{step.thought.length > 60 ? '…' : ''}</>
                    : step.thought.slice(0, 80) + (step.thought.length > 80 ? '…' : '')}
                </span>
                {isOpen
                  ? <ChevronDown className="h-3 w-3 text-muted-foreground/50 flex-shrink-0" />
                  : <ChevronRight className="h-3 w-3 text-muted-foreground/50 flex-shrink-0" />}
              </button>

              {isOpen && (
                <div className="px-5 pb-4 pt-2 space-y-3 bg-[var(--muted)]/30">
                  {step.thought && (
                    <div>
                      <p className="text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 mb-1">
                        Thought
                      </p>
                      <p className="text-[11px] text-foreground/80 leading-relaxed">{step.thought}</p>
                    </div>
                  )}
                  {step.action && (
                    <div>
                      <p className="text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 mb-1">
                        Tool
                      </p>
                      <span
                        className="inline-flex items-center gap-1 px-2 py-0.5 rounded-[3px] text-[10px] font-mono"
                        style={{
                          background: 'var(--apex-amber-glow)',
                          color: 'var(--apex-amber-bright)',
                          border: '1px solid var(--apex-amber-line)',
                        }}
                      >
                        <Zap className="h-2.5 w-2.5" strokeWidth={2} />
                        {step.action}
                      </span>
                    </div>
                  )}
                  {step.actionInput && (
                    <div>
                      <p className="text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 mb-1">
                        Input
                      </p>
                      <pre
                        className="text-[10px] font-mono p-2 rounded-sm overflow-auto max-h-32 text-foreground/70"
                        style={{ background: 'var(--background)', border: '1px solid var(--border)' }}
                      >
                        {typeof step.actionInput === 'string'
                          ? step.actionInput
                          : JSON.stringify(step.actionInput, null, 2)}
                      </pre>
                    </div>
                  )}
                  {step.observation && (
                    <div>
                      <p className="text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 mb-1">
                        Result
                      </p>
                      <pre
                        className="text-[10px] font-mono p-2 rounded-sm overflow-auto max-h-32 text-foreground/70"
                        style={{ background: 'var(--background)', border: '1px solid var(--border)' }}
                      >
                        {step.observation}
                      </pre>
                    </div>
                  )}
                </div>
              )}
            </div>
          )
        })}

        {/* Live token stream */}
        {tokens && (
          <div className="px-5 py-3">
            <p className="text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 mb-2">
              Streaming…
            </p>
            <pre
              className="text-[10px] font-mono text-foreground/70 whitespace-pre-wrap leading-relaxed p-2 rounded-sm"
              style={{ background: 'var(--background)', border: '1px solid var(--border)' }}
            >
              {tokens}
              <span
                className="inline-block w-[5px] h-[12px] ml-0.5 apex-blink"
                style={{ background: 'var(--apex-amber)', verticalAlign: 'text-bottom' }}
              />
            </pre>
          </div>
        )}

        {/* Empty state */}
        {!hasContent && (
          <div className="px-5 py-10 text-center">
            <Cpu className="h-6 w-6 text-muted-foreground/20 mx-auto mb-2" strokeWidth={1} />
            <p className="text-[11px] font-mono text-muted-foreground/40">
              No reasoning stream available.
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
