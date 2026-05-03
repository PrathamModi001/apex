'use client'

import { useState, useEffect, useRef } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

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
  const [tokens, setTokens] = useState<string>('')
  const [steps, setSteps] = useState<AgentStep[]>([])
  const [connected, setConnected] = useState(false)
  const [expandedSteps, setExpandedSteps] = useState<Set<number>>(new Set())
  const esRef = useRef<EventSource | null>(null)
  const stepIdRef = useRef(0)

  const baseUrl = agentServiceUrl || process.env.NEXT_PUBLIC_AGENT_SERVICE_URL || 'http://localhost:8081'

  useEffect(() => {
    const url = `${baseUrl}/stream/invoice/${invoiceId}`
    const es = new EventSource(url)
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
              thought: data.thought || '',
              action: data.action,
              actionInput: data.action_input,
              observation: data.observation,
              complete: true,
            },
          ])
          setTokens('')
        }
      } catch {
        // raw text token
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

  const toggleStep = (id: number) => {
    setExpandedSteps((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-sm">
          Agent Reasoning
          <span
            className={`inline-flex h-2 w-2 rounded-full ${
              connected ? 'bg-green-500 animate-pulse' : 'bg-zinc-500'
            }`}
          />
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {steps.map((step) => (
          <div key={step.id} className="rounded-lg border border-border bg-muted/30">
            <button
              className="w-full flex items-center justify-between px-3 py-2 text-xs font-medium text-left"
              onClick={() => toggleStep(step.id)}
            >
              <span>Step {step.id + 1}{step.action ? ` — ${step.action}` : ''}</span>
              <span className="text-muted-foreground">{expandedSteps.has(step.id) ? '▲' : '▼'}</span>
            </button>
            {expandedSteps.has(step.id) && (
              <div className="px-3 pb-3 text-xs space-y-1.5 border-t border-border pt-2">
                {step.thought && (
                  <div>
                    <span className="font-semibold text-muted-foreground">Thought:</span>
                    <p className="mt-0.5 text-foreground/80">{step.thought}</p>
                  </div>
                )}
                {step.action && (
                  <div>
                    <span className="font-semibold text-muted-foreground">Tool:</span>
                    <code className="ml-1 bg-muted px-1 rounded">{step.action}</code>
                  </div>
                )}
                {step.actionInput && (
                  <div>
                    <span className="font-semibold text-muted-foreground">Input:</span>
                    <pre className="mt-0.5 bg-muted p-2 rounded overflow-auto text-[10px]">
                      {typeof step.actionInput === 'string'
                        ? step.actionInput
                        : JSON.stringify(step.actionInput, null, 2)}
                    </pre>
                  </div>
                )}
                {step.observation && (
                  <div>
                    <span className="font-semibold text-muted-foreground">Result:</span>
                    <pre className="mt-0.5 bg-muted p-2 rounded overflow-auto text-[10px]">
                      {step.observation}
                    </pre>
                  </div>
                )}
              </div>
            )}
          </div>
        ))}
        {tokens && (
          <div className="rounded-lg border border-border bg-muted/30 px-3 py-2 text-xs font-mono text-foreground/80 whitespace-pre-wrap">
            {tokens}
            <span className="inline-block w-1 h-3 bg-foreground/60 animate-pulse ml-0.5" />
          </div>
        )}
        {!connected && steps.length === 0 && tokens === '' && (
          <p className="text-xs text-muted-foreground">No reasoning stream available.</p>
        )}
      </CardContent>
    </Card>
  )
}
