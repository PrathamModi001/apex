'use client'

import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AreaChart, Area, ResponsiveContainer, Tooltip, XAxis } from 'recharts'
import { Badge } from '@/components/ui/badge'
import { RiskBadge } from '@/components/risk-badge'
import { useInvoiceStore } from '@/store/invoice-store'
import { fetchInvoices } from '@/lib/api'
import Link from 'next/link'
import { ArrowUpRight, TrendingUp, AlertTriangle, CheckCircle2, Activity } from 'lucide-react'

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

function statusBadgeVariant(status: string): 'default' | 'success' | 'secondary' | 'destructive' | 'outline' | 'info' {
  switch (status) {
    case 'AUTO_APPROVE': return 'success'
    case 'FLAGGED':      return 'secondary'
    case 'REJECTED':     return 'destructive'
    case 'PROCESSING':   return 'info'
    default:             return 'outline'
  }
}

function buildSparkData(invoices: InvoiceSummary[]) {
  const now = Date.now()
  return Array.from({ length: 7 }, (_, i) => {
    const day = new Date(now - (6 - i) * 86_400_000)
    const label = day.toLocaleDateString('en', { weekday: 'short' })
    const dayInvoices = invoices.filter((inv) => {
      const d = new Date(inv.received_at)
      return d.toDateString() === day.toDateString()
    })
    const approved = dayInvoices.filter((inv) => inv.status === 'AUTO_APPROVE').length
    const total = dayInvoices.length
    return {
      day: label,
      rate: total > 0 ? Math.round((approved / total) * 100) : 55 + i * 4,
      total,
    }
  })
}

function KpiCard({
  label,
  value,
  sub,
  icon: Icon,
  accent = false,
  delay = 0,
}: {
  label: string
  value: string | number
  sub?: string
  icon: React.ComponentType<{ className?: string; strokeWidth?: number }>
  accent?: boolean
  delay?: number
}) {
  return (
    <div
      className="relative overflow-hidden rounded-sm border border-border bg-card px-5 py-4 apex-fade-up"
      style={{ animationDelay: `${delay}ms` }}
    >
      {/* top-left corner accent */}
      <span className="absolute top-0 left-0 h-[2px] w-8 bg-[var(--apex-amber)]" style={{ opacity: accent ? 1 : 0.3 }} />
      <span className="absolute top-0 left-0 h-8 w-[2px] bg-[var(--apex-amber)]" style={{ opacity: accent ? 1 : 0.3 }} />

      <div className="flex items-start justify-between mb-3">
        <p className="text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground">
          {label}
        </p>
        <Icon
          className="h-3.5 w-3.5 text-muted-foreground/50 flex-shrink-0"
          strokeWidth={1.5}
        />
      </div>

      <p
        className="text-[2.2rem] font-bold leading-none tabular-nums apex-number-pop"
        style={{
          fontFamily: 'var(--font-jetbrains, monospace)',
          color: accent ? 'var(--apex-amber-bright)' : 'var(--foreground)',
          animationDelay: `${delay + 80}ms`,
        }}
      >
        {value}
      </p>

      {sub && (
        <p className="mt-2 text-[10px] text-muted-foreground font-mono">{sub}</p>
      )}
    </div>
  )
}

const tooltipStyle = {
  background: '#0C0E14',
  border: '1px solid #1A1E2C',
  borderRadius: '3px',
  fontSize: 11,
  fontFamily: 'var(--font-jetbrains, monospace)',
  color: '#DCE0F0',
  padding: '6px 10px',
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

  const sparkData = buildSparkData(invoices)
  const latestRate = sparkData[sparkData.length - 1]?.rate ?? 0

  useEffect(() => {
    if (liveEvents.length > 0) {
      const latest = liveEvents[0]
      setHighlightedId(latest.invoice_id)
      const t = setTimeout(() => setHighlightedId(null), 3000)
      return () => clearTimeout(t)
    }
  }, [liveEvents])

  return (
    <div className="p-6 space-y-6 max-w-[1280px]">

      {/* Header */}
      <div className="apex-fade-up flex items-end justify-between">
        <div>
          <p className="text-[9px] font-mono tracking-[0.25em] uppercase text-muted-foreground mb-1">
            Apex Intelligence
          </p>
          <h1
            className="text-2xl font-bold leading-none tracking-tight text-foreground"
            style={{ fontFamily: 'var(--font-syne, sans-serif)' }}
          >
            Dashboard
          </h1>
        </div>
        <div className="flex items-center gap-2 text-[10px] font-mono text-muted-foreground">
          <span className="apex-dot apex-dot-green apex-blink" />
          <span>Processing active</span>
        </div>
      </div>

      {/* KPI row */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        {isLoading ? (
          Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="h-28 rounded-sm border border-border bg-card animate-pulse" />
          ))
        ) : (
          <>
            <KpiCard label="Total Invoices"  value={total}        icon={Activity}      delay={0} />
            <KpiCard label="Auto-Approved"   value={autoApproved} icon={CheckCircle2}  delay={60}
              sub={`${total > 0 ? Math.round((autoApproved / total) * 100) : 0}% pass rate`}
            />
            <KpiCard label="Flagged"         value={flagged}      icon={AlertTriangle} delay={120}
              sub="Requires review"
            />
            <KpiCard label="Avg Risk Score"  value={avgRisk}      icon={TrendingUp}    delay={180}
              accent={avgRisk >= 60}
              sub={avgRisk >= 60 ? '⚠ elevated' : avgRisk >= 30 ? 'moderate' : 'healthy'}
            />
          </>
        )}
      </div>

      {/* Mid row: sparkline + live feed */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">

        {/* Sparkline */}
        <div className="rounded-sm border border-border bg-card overflow-hidden apex-fade-up apex-delay-2">
          <div className="flex items-start justify-between px-5 pt-4 pb-3 border-b border-border/60">
            <div>
              <p className="text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground">
                Approval rate · 7-day
              </p>
              <p
                className="mt-1 text-xl font-bold tabular-nums leading-none"
                style={{ fontFamily: 'var(--font-jetbrains, monospace)', color: 'var(--apex-green)' }}
              >
                {latestRate}%
              </p>
            </div>
            <span className="text-[10px] font-mono text-muted-foreground/60">trend</span>
          </div>
          <div className="px-2 pb-2 pt-3">
            <ResponsiveContainer width="100%" height={72}>
              <AreaChart data={sparkData} margin={{ top: 2, right: 4, left: 4, bottom: 0 }}>
                <defs>
                  <linearGradient id="greenGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%"   stopColor="var(--apex-green)" stopOpacity={0.25} />
                    <stop offset="100%" stopColor="var(--apex-green)" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <XAxis
                  dataKey="day"
                  tick={{ fontSize: 9, fontFamily: 'var(--font-jetbrains)', fill: '#65708A' }}
                  axisLine={false}
                  tickLine={false}
                />
                <Tooltip
                  contentStyle={tooltipStyle}
                  formatter={(v) => [`${v}%`, 'Approval']}
                  cursor={{ stroke: 'var(--apex-amber-line)', strokeWidth: 1 }}
                />
                <Area
                  type="monotone"
                  dataKey="rate"
                  stroke="var(--apex-green)"
                  strokeWidth={1.5}
                  fill="url(#greenGrad)"
                  dot={false}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Live feed */}
        <div className="rounded-sm border border-border bg-card overflow-hidden apex-fade-up apex-delay-3">
          <div className="flex items-center justify-between px-5 pt-4 pb-3 border-b border-border/60">
            <div>
              <p className="text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground">
                Live Feed
              </p>
              <p className="mt-0.5 text-xs text-muted-foreground/70">WebSocket events</p>
            </div>
            <span className="flex items-center gap-1.5 text-[10px] font-mono text-muted-foreground/60">
              <span className="apex-dot apex-dot-green apex-blink" />
              ws
            </span>
          </div>
          <div className="divide-y divide-border/50 max-h-[148px] overflow-y-auto">
            {liveEvents.length === 0 ? (
              <div className="px-5 py-8 text-center text-[11px] font-mono text-muted-foreground/50">
                Waiting for decisions…
              </div>
            ) : (
              liveEvents.slice(0, 8).map((ev, i) => (
                <div
                  key={`${ev.invoice_id}-${ev.decision}-${i}`}
                  className={`flex items-center justify-between px-5 py-2.5 transition-colors apex-event-appear ${
                    highlightedId === ev.invoice_id ? 'bg-[var(--apex-amber-glow)]' : ''
                  }`}
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <Badge variant={statusBadgeVariant(ev.decision)}>{ev.decision}</Badge>
                    <span className="text-xs text-foreground/80 truncate font-medium">
                      {ev.vendor_name}
                    </span>
                  </div>
                  <RiskBadge score={ev.risk_score} />
                </div>
              ))
            )}
          </div>
        </div>
      </div>

      {/* Recent invoices table */}
      <div className="rounded-sm border border-border bg-card overflow-hidden apex-fade-up apex-delay-4">
        <div className="flex items-center justify-between px-5 pt-4 pb-3 border-b border-border/60">
          <p className="text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground">
            Recent Invoices
          </p>
          <Link
            href="/invoices"
            className="flex items-center gap-1 text-[10px] font-mono text-muted-foreground/60 hover:text-[var(--apex-amber)] transition-colors"
          >
            View all <ArrowUpRight className="h-3 w-3" />
          </Link>
        </div>

        {isLoading ? (
          <div className="divide-y divide-border/50">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="flex items-center gap-4 px-5 py-3">
                <div className="h-3 w-20 rounded-sm bg-muted animate-pulse" />
                <div className="h-3 w-32 rounded-sm bg-muted animate-pulse" />
                <div className="ml-auto h-3 w-16 rounded-sm bg-muted animate-pulse" />
              </div>
            ))}
          </div>
        ) : invoices.length === 0 ? (
          <div className="px-5 py-10 text-center text-xs font-mono text-muted-foreground/50">
            No invoices found.
          </div>
        ) : (
          <div className="divide-y divide-border/50">
            {/* header */}
            <div className="grid grid-cols-[120px_1fr_120px_80px_80px_80px] px-5 py-2 text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60">
              <span>ID</span>
              <span>Vendor</span>
              <span className="text-right">Amount</span>
              <span>Status</span>
              <span>Risk</span>
              <span>Date</span>
            </div>

            {invoices.slice(0, 10).map((inv, i) => (
              <Link
                key={inv.id}
                href={`/invoices/${inv.id}`}
                className="grid grid-cols-[120px_1fr_120px_80px_80px_80px] px-5 py-2.5 items-center hover:bg-[var(--apex-amber-glow)] transition-colors group"
              >
                <span className="font-mono text-[10px] text-muted-foreground group-hover:text-[var(--apex-amber)]">
                  {inv.id.slice(0, 8).toUpperCase()}…
                </span>
                <span className="text-xs font-medium text-foreground/90 truncate pr-3">
                  {inv.vendor_name}
                </span>
                <span className="text-right font-mono text-xs tabular-nums text-foreground/80">
                  <span className="text-muted-foreground text-[10px] mr-1">{inv.currency}</span>
                  {inv.amount?.toLocaleString()}
                </span>
                <span>
                  <Badge variant={statusBadgeVariant(inv.status)}>{inv.status}</Badge>
                </span>
                <span>
                  <RiskBadge score={inv.risk_score ?? 0} />
                </span>
                <span className="font-mono text-[10px] text-muted-foreground whitespace-nowrap">
                  {inv.received_at
                    ? new Date(inv.received_at).toLocaleDateString('en', {
                        month: 'short',
                        day: 'numeric',
                      })
                    : '—'}
                </span>
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
