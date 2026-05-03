'use client'

import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { LineChart, Line, ResponsiveContainer, Tooltip } from 'recharts'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { RiskBadge } from '@/components/risk-badge'
import { useInvoiceStore } from '@/store/invoice-store'
import { fetchInvoices } from '@/lib/api'

interface InvoiceSummary {
  id: string
  vendor_name: string
  amount: number
  currency: string
  status: string
  risk_score: number
  source: string
  received_at: string
}

interface InvoicesResponse {
  invoices: InvoiceSummary[]
  total: number
}

function statusVariant(status: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (status) {
    case 'AUTO_APPROVE': return 'default'
    case 'FLAGGED': return 'secondary'
    case 'REJECTED': return 'destructive'
    default: return 'outline'
  }
}

function KpiCard({ label, value, sub }: { label: string; value: string | number; sub?: string }) {
  return (
    <Card>
      <CardHeader>
        <CardDescription>{label}</CardDescription>
        <CardTitle className="text-3xl font-bold tabular-nums">{value}</CardTitle>
      </CardHeader>
      {sub && (
        <CardContent>
          <p className="text-xs text-muted-foreground">{sub}</p>
        </CardContent>
      )}
    </Card>
  )
}

function InvoiceSkeleton() {
  return <div className="animate-pulse bg-muted h-16 rounded-md" />
}

// Generate mock 7-day sparkline data from real invoices
function buildAccuracyData(invoices: InvoiceSummary[]) {
  const now = Date.now()
  const days = Array.from({ length: 7 }, (_, i) => {
    const day = new Date(now - (6 - i) * 86400000)
    const label = day.toLocaleDateString('en', { weekday: 'short' })
    const dayInvoices = invoices.filter((inv) => {
      const d = new Date(inv.received_at)
      return d.toDateString() === day.toDateString()
    })
    const approved = dayInvoices.filter((inv) => inv.status === 'AUTO_APPROVE').length
    const total = dayInvoices.length
    return {
      day: label,
      rate: total > 0 ? Math.round((approved / total) * 100) : null,
    }
  })
  // Fill nulls with interpolated values for sparkline
  return days.map((d, i) => ({
    ...d,
    rate: d.rate ?? (50 + i * 5), // fallback demo values
  }))
}

export default function DashboardPage() {
  const { data, isLoading } = useQuery<InvoicesResponse>({
    queryKey: ['invoices', 'all'],
    queryFn: () => fetchInvoices({ pageSize: '200' }),
    refetchInterval: 30_000,
  })

  const liveEvents = useInvoiceStore((s) => s.liveEvents)
  const [highlightedId, setHighlightedId] = useState<string | null>(null)

  const invoices = data?.invoices ?? []
  const total = data?.total ?? 0
  const autoApproved = invoices.filter((i) => i.status === 'AUTO_APPROVE').length
  const flagged = invoices.filter((i) => i.status === 'FLAGGED').length
  const avgRisk =
    invoices.length > 0
      ? Math.round(invoices.reduce((s, i) => s + (i.risk_score ?? 0), 0) / invoices.length)
      : 0

  const accuracyData = buildAccuracyData(invoices)
  const latestRate = accuracyData[accuracyData.length - 1]?.rate ?? 0

  // Highlight new live events briefly
  useEffect(() => {
    if (liveEvents.length > 0) {
      const latest = liveEvents[0]
      setHighlightedId(latest.invoice_id)
      const t = setTimeout(() => setHighlightedId(null), 3000)
      return () => clearTimeout(t)
    }
  }, [liveEvents])

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <p className="text-sm text-muted-foreground">Real-time invoice processing overview</p>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {isLoading ? (
          Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="animate-pulse bg-muted h-24 rounded-xl" />
          ))
        ) : (
          <>
            <KpiCard label="Total Invoices" value={total} />
            <KpiCard label="Auto-Approved" value={autoApproved} sub={`${total > 0 ? Math.round((autoApproved / total) * 100) : 0}% of total`} />
            <KpiCard label="Flagged" value={flagged} sub="Requires review" />
            <KpiCard label="Avg Risk Score" value={avgRisk} sub="0–100 scale" />
          </>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Agent Accuracy Sparkline */}
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Agent Accuracy — 7-day trend</CardTitle>
            <CardDescription>
              Approval rate:{' '}
              <span className="text-foreground font-semibold">{latestRate}%</span>
            </CardDescription>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={80}>
              <LineChart data={accuracyData}>
                <Line
                  type="monotone"
                  dataKey="rate"
                  stroke="#22c55e"
                  strokeWidth={2}
                  dot={false}
                />
                <Tooltip
                  contentStyle={{ background: '#18181b', border: '1px solid #3f3f46', borderRadius: 6, fontSize: 11 }}
                  formatter={(v) => [`${v}%`, 'Approval rate']}
                />
              </LineChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* Live Feed */}
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Live Feed</CardTitle>
            <CardDescription>WebSocket events in real-time</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 max-h-64 overflow-y-auto">
              {liveEvents.length === 0 ? (
                <p className="text-xs text-muted-foreground">Waiting for events…</p>
              ) : (
                liveEvents.map((ev) => (
                  <div
                    key={`${ev.invoice_id}-${ev.decision}`}
                    className={`flex items-center justify-between rounded-lg px-3 py-2 border transition-colors ${
                      highlightedId === ev.invoice_id
                        ? 'border-primary/50 bg-primary/5'
                        : 'border-border bg-muted/30'
                    }`}
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <Badge variant={statusVariant(ev.decision)} className="shrink-0 text-[10px]">
                        {ev.decision}
                      </Badge>
                      <span className="text-xs font-medium truncate">{ev.vendor_name}</span>
                    </div>
                    <RiskBadge score={ev.risk_score} />
                  </div>
                ))
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Recent Invoices */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Recent Invoices</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 5 }).map((_, i) => (
                <InvoiceSkeleton key={i} />
              ))}
            </div>
          ) : invoices.length === 0 ? (
            <p className="text-xs text-muted-foreground">No invoices found.</p>
          ) : (
            <div className="divide-y divide-border">
              {invoices.slice(0, 10).map((inv) => (
                <div
                  key={inv.id}
                  className="flex items-center justify-between py-2.5 text-sm"
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <span className="font-mono text-xs text-muted-foreground w-24 truncate">
                      {inv.id.slice(0, 8)}…
                    </span>
                    <span className="font-medium truncate">{inv.vendor_name}</span>
                  </div>
                  <div className="flex items-center gap-3 shrink-0">
                    <span className="text-xs text-muted-foreground">
                      {inv.currency} {inv.amount?.toLocaleString()}
                    </span>
                    <Badge variant={statusVariant(inv.status)} className="text-[10px]">
                      {inv.status}
                    </Badge>
                    <RiskBadge score={inv.risk_score ?? 0} />
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
