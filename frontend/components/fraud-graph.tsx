'use client'

import { useEffect, useState, useCallback } from 'react'
import { Network } from 'lucide-react'

interface GraphNode {
  id: string
  type: 'vendor' | 'bank_account' | string
  label: string
  [key: string]: unknown
}

interface GraphEdge {
  id: string
  source: string
  target: string
  label?: string
}

interface FraudGraphProps {
  invoiceId: string
}

export function FraudGraph({ invoiceId }: FraudGraphProps) {
  const [graphData, setGraphData] = useState<{ nodes: GraphNode[]; edges: GraphEdge[] } | null>(null)
  const [FlowComponent, setFlowComponent] = useState<React.ComponentType<{
    nodes: unknown[]
    edges: unknown[]
    fitView: boolean
    colorMode: string
  }> | null>(null)
  const [flowAvailable, setFlowAvailable] = useState<boolean | null>(null)

  useEffect(() => {
    import('@xyflow/react')
      .then((mod) => {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        setFlowComponent(mod.ReactFlow as any)
        setFlowAvailable(true)
      })
      .catch(() => setFlowAvailable(false))
  }, [])

  const fetchGraph = useCallback(async () => {
    try {
      const res = await fetch(`/api/fraud-graph/${invoiceId}`)
      if (!res.ok) throw new Error('failed')
      const data = await res.json()
      setGraphData(data)
    } catch {
      setGraphData({ nodes: [], edges: [] })
    }
  }, [invoiceId])

  useEffect(() => { fetchGraph() }, [fetchGraph])

  const mapNodes = (nodes: GraphNode[]) =>
    nodes.map((n, i) => ({
      id: n.id,
      data: { label: n.label },
      position: { x: (i % 3) * 160 + 40, y: Math.floor(i / 3) * 100 + 40 },
      style: {
        background: n.type === 'vendor' ? '#1A1C2A' : '#0F1218',
        color: n.type === 'vendor' ? '#E8A820' : '#4888F8',
        border: `1.5px solid ${n.type === 'vendor' ? '#C88A0A40' : '#4888F840'}`,
        borderRadius: 2,
        padding: '6px 10px',
        fontSize: 10,
        fontFamily: 'var(--font-jetbrains, monospace)',
      },
    }))

  const mapEdges = (edges: GraphEdge[]) =>
    edges.map((e) => ({
      id: e.id,
      source: e.source,
      target: e.target,
      label: e.label,
      style: { stroke: '#1A1E2C', strokeWidth: 1.5 },
      labelStyle: { fontSize: 9, fontFamily: 'var(--font-jetbrains, monospace)', fill: '#65708A' },
    }))

  if (!graphData) {
    return (
      <div className="rounded-sm border border-border bg-card overflow-hidden">
        <div className="px-5 pt-4 pb-3 border-b border-border/60">
          <p className="text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground">
            Fraud Network Graph
          </p>
        </div>
        <div className="h-48 animate-pulse bg-muted/40 m-4 rounded-sm" />
      </div>
    )
  }

  if (graphData.nodes.length === 0) {
    return (
      <div className="rounded-sm border border-border bg-card overflow-hidden">
        <div className="px-5 pt-4 pb-3 border-b border-border/60">
          <p className="text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground">
            Fraud Network Graph
          </p>
        </div>
        <div className="px-5 py-8 text-center">
          <Network className="h-6 w-6 text-muted-foreground/20 mx-auto mb-2" strokeWidth={1} />
          <p className="text-[11px] font-mono text-muted-foreground/50">No graph connections found.</p>
        </div>
      </div>
    )
  }

  return (
    <div className="rounded-sm border border-border bg-card overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-5 pt-4 pb-3 border-b border-border/60">
        <p className="text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground">
          Fraud Network Graph
        </p>
        <div className="flex items-center gap-3 text-[9px] font-mono text-muted-foreground/60">
          <span className="flex items-center gap-1">
            <span className="inline-block w-2 h-2 rounded-[2px]" style={{ background: 'var(--apex-amber)', opacity: 0.7 }} />
            Vendor
          </span>
          <span className="flex items-center gap-1">
            <span className="inline-block w-2 h-2 rounded-[2px]" style={{ background: 'var(--apex-blue)', opacity: 0.7 }} />
            Account
          </span>
        </div>
      </div>

      <div className="p-4">
        {flowAvailable && FlowComponent ? (
          <div
            className="h-64 w-full rounded-sm overflow-hidden"
            style={{ border: '1px solid var(--border)' }}
          >
            <FlowComponent
              nodes={mapNodes(graphData.nodes)}
              edges={mapEdges(graphData.edges)}
              fitView
              colorMode="dark"
            />
          </div>
        ) : (
          <div className="space-y-3">
            {flowAvailable === null && (
              <p className="text-[10px] font-mono text-muted-foreground/50">Loading renderer…</p>
            )}
            {flowAvailable === false && (
              <p className="text-[10px] font-mono text-muted-foreground/50">
                Graph renderer unavailable — showing node list.
              </p>
            )}
            <div className="flex flex-wrap gap-2">
              {graphData.nodes.map((n) => (
                <span
                  key={n.id}
                  className="rounded-[3px] px-2 py-1 text-[10px] font-mono font-medium"
                  style={{
                    background: n.type === 'vendor' ? 'var(--apex-amber-glow)' : 'rgba(72,136,248,0.1)',
                    color: n.type === 'vendor' ? 'var(--apex-amber-bright)' : 'var(--apex-blue)',
                    border: `1px solid ${n.type === 'vendor' ? 'var(--apex-amber-line)' : 'rgba(72,136,248,0.25)'}`,
                  }}
                >
                  {n.label}
                </span>
              ))}
            </div>
            {graphData.edges.length > 0 && (
              <div className="space-y-1 mt-2">
                {graphData.edges.map((e) => (
                  <p key={e.id} className="text-[9px] font-mono text-muted-foreground/50">
                    {e.source} {'->'} {e.target}
                    {e.label ? ` (${e.label})` : ''}
                  </p>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
