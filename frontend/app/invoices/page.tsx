'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { useQuery } from '@tanstack/react-query'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import { RiskBadge } from '@/components/risk-badge'
import { fetchInvoices } from '@/lib/api'
import { Mail, MessageSquare, Wrench, ChevronLeft, ChevronRight, X } from 'lucide-react'

interface Invoice {
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
  invoices: Invoice[]
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

function SourceIcon({ source }: { source: string }) {
  if (source === 'gmail')    return <Mail className="h-3 w-3 text-[var(--apex-blue)]" />
  if (source === 'telegram') return <MessageSquare className="h-3 w-3 text-[var(--apex-amber)]" />
  return <Wrench className="h-3 w-3 text-muted-foreground" />
}

const PAGE_SIZE = 20

export default function InvoicesPage() {
  const router = useRouter()
  const [status, setStatus]   = useState('')
  const [source, setSource]   = useState('')
  const [minRisk, setMinRisk] = useState('')
  const [maxRisk, setMaxRisk] = useState('')
  const [page, setPage]       = useState(1)

  const filters: Record<string, string> = {}
  if (status)  filters.status  = status
  if (source)  filters.source  = source
  if (minRisk) filters.minRisk = minRisk
  if (maxRisk) filters.maxRisk = maxRisk
  filters.page     = String(page)
  filters.pageSize = String(PAGE_SIZE)

  const { data, isLoading } = useQuery<InvoicesResponse>({
    queryKey: ['invoices', filters],
    queryFn:  () => fetchInvoices(filters),
    refetchInterval: 10_000,
  })

  const invoices   = data?.invoices ?? []
  const total      = data?.total    ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const resetPage  = () => setPage(1)

  const hasFilters = !!(status || source || minRisk || maxRisk)

  return (
    <div className="p-6 space-y-4 max-w-[1280px]">

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
            Invoices
          </h1>
        </div>
        {!isLoading && (
          <span className="text-[10px] font-mono text-muted-foreground">
            {total} record{total !== 1 ? 's' : ''}
          </span>
        )}
      </div>

      {/* Filter bar */}
      <div className="rounded-sm border border-border bg-card px-4 py-3 apex-fade-up apex-delay-1">
        <div className="flex flex-wrap gap-3 items-end">
          <div className="flex flex-col gap-1">
            <label className="text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/70">
              Status
            </label>
            <Select
              value={status}
              onChange={(e) => { setStatus(e.target.value); resetPage() }}
              className="w-40 h-7 text-xs font-mono rounded-sm border-border bg-[var(--input)] text-foreground"
            >
              <option value="">All</option>
              <option value="PROCESSING">PROCESSING</option>
              <option value="AUTO_APPROVE">AUTO_APPROVE</option>
              <option value="FLAGGED">FLAGGED</option>
              <option value="REJECTED">REJECTED</option>
            </Select>
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/70">
              Source
            </label>
            <Select
              value={source}
              onChange={(e) => { setSource(e.target.value); resetPage() }}
              className="w-32 h-7 text-xs font-mono rounded-sm border-border bg-[var(--input)] text-foreground"
            >
              <option value="">All</option>
              <option value="gmail">Gmail</option>
              <option value="telegram">Telegram</option>
              <option value="test">Test</option>
            </Select>
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/70">
              Min Risk
            </label>
            <Input
              type="number"
              placeholder="0"
              value={minRisk}
              onChange={(e) => { setMinRisk(e.target.value); resetPage() }}
              className="w-20 h-7 text-xs font-mono rounded-sm"
              min={0} max={100}
            />
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/70">
              Max Risk
            </label>
            <Input
              type="number"
              placeholder="100"
              value={maxRisk}
              onChange={(e) => { setMaxRisk(e.target.value); resetPage() }}
              className="w-20 h-7 text-xs font-mono rounded-sm"
              min={0} max={100}
            />
          </div>

          {hasFilters && (
            <Button
              variant="ghost"
              size="sm"
              className="h-7 gap-1"
              onClick={() => { setStatus(''); setSource(''); setMinRisk(''); setMaxRisk(''); setPage(1) }}
            >
              <X className="h-3 w-3" /> Clear
            </Button>
          )}
        </div>
      </div>

      {/* Table */}
      <div className="rounded-sm border border-border bg-card overflow-hidden apex-fade-up apex-delay-2">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border/70 bg-muted/30">
                <th className="px-4 py-2.5 text-left text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 font-normal">ID</th>
                <th className="px-4 py-2.5 text-left text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 font-normal">Vendor</th>
                <th className="px-4 py-2.5 text-right text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 font-normal">Amount</th>
                <th className="px-4 py-2.5 text-left text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 font-normal">Status</th>
                <th className="px-4 py-2.5 text-left text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 font-normal">Risk</th>
                <th className="px-4 py-2.5 text-left text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 font-normal">Via</th>
                <th className="px-4 py-2.5 text-left text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 font-normal">Received</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/50">
              {isLoading ? (
                Array.from({ length: 8 }).map((_, i) => (
                  <tr key={i} className="animate-pulse">
                    {Array.from({ length: 7 }).map((_, j) => (
                      <td key={j} className="px-4 py-3">
                        <div className="h-3 rounded-sm bg-muted w-full" />
                      </td>
                    ))}
                  </tr>
                ))
              ) : invoices.length === 0 ? (
                <tr>
                  <td colSpan={7} className="py-16 text-center text-[11px] font-mono text-muted-foreground/50">
                    No invoices match your filters.
                  </td>
                </tr>
              ) : (
                invoices.map((inv) => (
                  <tr
                    key={inv.id}
                    className="group hover:bg-[var(--apex-amber-glow)] cursor-pointer transition-colors"
                    onClick={() => router.push(`/invoices/${inv.id}`)}
                  >
                    <td className="px-4 py-2.5 font-mono text-[10px] text-muted-foreground group-hover:text-[var(--apex-amber)]">
                      {inv.id.slice(0, 8).toUpperCase()}…
                    </td>
                    <td className="px-4 py-2.5 font-medium text-xs max-w-[180px] truncate text-foreground/90">
                      {inv.vendor_name}
                    </td>
                    <td className="px-4 py-2.5 text-right font-mono text-xs tabular-nums text-foreground/80">
                      <span className="text-muted-foreground text-[10px] mr-1">{inv.currency}</span>
                      {inv.amount?.toLocaleString()}
                    </td>
                    <td className="px-4 py-2.5">
                      <Badge variant={statusBadgeVariant(inv.status)}>{inv.status}</Badge>
                    </td>
                    <td className="px-4 py-2.5">
                      <RiskBadge score={inv.risk_score ?? 0} />
                    </td>
                    <td className="px-4 py-2.5">
                      <SourceIcon source={inv.source} />
                    </td>
                    <td className="px-4 py-2.5 font-mono text-[10px] text-muted-foreground whitespace-nowrap">
                      {inv.received_at
                        ? new Date(inv.received_at).toLocaleDateString('en', {
                            month: 'short', day: 'numeric',
                            hour: '2-digit', minute: '2-digit',
                          })
                        : '—'}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {!isLoading && total > PAGE_SIZE && (
          <div className="flex items-center justify-between px-4 py-3 border-t border-border/60">
            <p className="text-[10px] font-mono text-muted-foreground/60">
              Page {page} / {totalPages} · {total} total
            </p>
            <div className="flex gap-1">
              <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
                <ChevronLeft className="h-3 w-3" />
              </Button>
              <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>
                <ChevronRight className="h-3 w-3" />
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
