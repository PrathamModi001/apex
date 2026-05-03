'use client'

import { useEffect, useState, useCallback } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

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

// Dynamic import wrapper for @xyflow/react to handle SSR and availability
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
    // Dynamically import to handle potential unavailability
    import('@xyflow/react')
      .then((mod) => {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        setFlowComponent(mod.ReactFlow as any)
        setFlowAvailable(true)
      })
      .catch(() => {
        setFlowAvailable(false)
      })
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

  useEffect(() => {
    fetchGraph()
  }, [fetchGraph])

  const mapNodes = (nodes: GraphNode[]) =>
    nodes.map((n) => ({
      id: n.id,
      data: { label: n.label },
      position: { x: Math.random() * 400, y: Math.random() * 300 },
      style: {
        background: n.type === 'vendor' ? '#06b6d4' : '#f97316',
        color: '#fff',
        borderRadius: 8,
        padding: '8px 12px',
        fontSize: 11,
        border: 'none',
      },
    }))

  const mapEdges = (edges: GraphEdge[]) =>
    edges.map((e) => ({
      id: e.id,
      source: e.source,
      target: e.target,
      label: e.label,
      style: { stroke: '#6b7280' },
    }))

  if (!graphData) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Fraud Network Graph</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="animate-pulse bg-muted h-48 rounded-md" />
        </CardContent>
      </Card>
    )
  }

  if (graphData.nodes.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Fraud Network Graph</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-xs text-muted-foreground">No graph data available.</p>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm flex items-center gap-3">
          Fraud Network Graph
          <span className="flex items-center gap-2 text-xs font-normal text-muted-foreground">
            <span className="inline-block h-2 w-2 rounded-full bg-cyan-500" /> Vendor
            <span className="inline-block h-2 w-2 rounded-full bg-orange-500" /> Bank Account
          </span>
        </CardTitle>
      </CardHeader>
      <CardContent>
        {flowAvailable && FlowComponent ? (
          <div className="h-64 w-full rounded-lg overflow-hidden border border-border">
            <FlowComponent
              nodes={mapNodes(graphData.nodes)}
              edges={mapEdges(graphData.edges)}
              fitView
              colorMode="dark"
            />
          </div>
        ) : (
          // Fallback: simple node list
          <div className="space-y-2">
            <p className="text-xs text-muted-foreground mb-2">
              {flowAvailable === false ? 'Graph renderer unavailable — showing node list.' : 'Loading…'}
            </p>
            <div className="flex flex-wrap gap-2">
              {graphData.nodes.map((n) => (
                <span
                  key={n.id}
                  className={`rounded px-2 py-1 text-xs font-medium text-white ${
                    n.type === 'vendor' ? 'bg-cyan-600' : 'bg-orange-600'
                  }`}
                >
                  {n.label}
                </span>
              ))}
            </div>
            <div className="mt-2 space-y-1">
              {graphData.edges.map((e) => (
                <p key={e.id} className="text-[10px] text-muted-foreground">
                  {e.source} → {e.target}
                  {e.label ? ` (${e.label})` : ''}
                </p>
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
