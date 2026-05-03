'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { RiskBadge } from '@/components/risk-badge'
import { fetchInvoices } from '@/lib/api'

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

function statusVariant(status: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (status) {
    case 'AUTO_APPROVE': return 'default'
    case 'FLAGGED': return 'secondary'
    case 'REJECTED': return 'destructive'
    default: return 'outline'
  }
}

function sourceIcon(source: string): string {
  switch (source) {
    case 'gmail': return '📧'
    case 'telegram': return '📱'
    default: return '🔧'
  }
}

const PAGE_SIZE = 20

export default function InvoicesPage() {
  const router = useRouter()
  const [status, setStatus] = useState('')
  const [source, setSource] = useState('')
  const [minRisk, setMinRisk] = useState('')
  const [maxRisk, setMaxRisk] = useState('')
  const [page, setPage] = useState(1)

  const filters: Record<string, string> = {}
  if (status) filters.status = status
  if (source) filters.source = source
  if (minRisk) filters.minRisk = minRisk
  if (maxRisk) filters.maxRisk = maxRisk
  filters.page = String(page)
  filters.pageSize = String(PAGE_SIZE)

  const { data, isLoading } = useQuery<InvoicesResponse>({
    queryKey: ['invoices', filters],
    queryFn: () => fetchInvoices(filters),
    refetchInterval: 10_000,
  })

  const invoices = data?.invoices ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const resetPage = () => setPage(1)

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Invoices</h1>
        <p className="text-sm text-muted-foreground">Browse and filter all processed invoices</p>
      </div>

      {/* Filter bar */}
      <Card>
        <CardContent className="pt-4">
          <div className="flex flex-wrap gap-3 items-end">
            <div className="flex flex-col gap-1">
              <label className="text-xs text-muted-foreground">Status</label>
              <Select
                value={status}
                onChange={(e) => { setStatus(e.target.value); resetPage() }}
                className="w-40"
              >
                <option value="">All</option>
                <option value="PROCESSING">PROCESSING</option>
                <option value="AUTO_APPROVE">AUTO_APPROVE</option>
                <option value="FLAGGED">FLAGGED</option>
                <option value="REJECTED">REJECTED</option>
              </Select>
            </div>

            <div className="flex flex-col gap-1">
              <label className="text-xs text-muted-foreground">Source</label>
              <Select
                value={source}
                onChange={(e) => { setSource(e.target.value); resetPage() }}
                className="w-36"
              >
                <option value="">All</option>
                <option value="gmail">Gmail</option>
                <option value="telegram">Telegram</option>
                <option value="test">Test</option>
              </Select>
            </div>

            <div className="flex flex-col gap-1">
              <label className="text-xs text-muted-foreground">Min Risk</label>
              <Input
                type="number"
                placeholder="0"
                value={minRisk}
                onChange={(e) => { setMinRisk(e.target.value); resetPage() }}
                className="w-20"
                min={0}
                max={100}
              />
            </div>

            <div className="flex flex-col gap-1">
              <label className="text-xs text-muted-foreground">Max Risk</label>
              <Input
                type="number"
                placeholder="100"
                value={maxRisk}
                onChange={(e) => { setMaxRisk(e.target.value); resetPage() }}
                className="w-20"
                min={0}
                max={100}
              />
            </div>

            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setStatus('')
                setSource('')
                setMinRisk('')
                setMaxRisk('')
                setPage(1)
              }}
            >
              Clear
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Table */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center justify-between">
            <span>Results</span>
            {!isLoading && (
              <span className="text-xs text-muted-foreground font-normal">
                {total} invoice{total !== 1 ? 's' : ''}
              </span>
            )}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border text-xs text-muted-foreground">
                  <th className="py-2 px-3 text-left font-medium">ID</th>
                  <th className="py-2 px-3 text-left font-medium">Vendor</th>
                  <th className="py-2 px-3 text-right font-medium">Amount</th>
                  <th className="py-2 px-3 text-left font-medium">Status</th>
                  <th className="py-2 px-3 text-left font-medium">Risk</th>
                  <th className="py-2 px-3 text-left font-medium">Source</th>
                  <th className="py-2 px-3 text-left font-medium">Date</th>
                </tr>
              </thead>
              <tbody>
                {isLoading ? (
                  Array.from({ length: 5 }).map((_, i) => (
                    <tr key={i} className="border-b border-border/50">
                      {Array.from({ length: 7 }).map((_, j) => (
                        <td key={j} className="py-2.5 px-3">
                          <Skeleton className="h-4 w-full" />
                        </td>
                      ))}
                    </tr>
                  ))
                ) : invoices.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="py-12 text-center text-muted-foreground text-sm">
                      No invoices match your filters.
                    </td>
                  </tr>
                ) : (
                  invoices.map((inv) => (
                    <tr
                      key={inv.id}
                      className="border-b border-border/50 hover:bg-muted/30 cursor-pointer transition-colors"
                      onClick={() => router.push(`/invoices/${inv.id}`)}
                    >
                      <td className="py-2.5 px-3 font-mono text-xs text-muted-foreground">
                        {inv.id.slice(0, 8)}…
                      </td>
                      <td className="py-2.5 px-3 font-medium max-w-[160px] truncate">
                        {inv.vendor_name}
                      </td>
                      <td className="py-2.5 px-3 text-right tabular-nums">
                        <span className="text-xs text-muted-foreground mr-1">{inv.currency}</span>
                        {inv.amount?.toLocaleString()}
                      </td>
                      <td className="py-2.5 px-3">
                        <Badge variant={statusVariant(inv.status)} className="text-[10px]">
                          {inv.status}
                        </Badge>
                      </td>
                      <td className="py-2.5 px-3">
                        <RiskBadge score={inv.risk_score ?? 0} />
                      </td>
                      <td className="py-2.5 px-3 text-base" title={inv.source}>
                        {sourceIcon(inv.source)}
                      </td>
                      <td className="py-2.5 px-3 text-xs text-muted-foreground whitespace-nowrap">
                        {inv.received_at
                          ? new Date(inv.received_at).toLocaleDateString('en', {
                              month: 'short',
                              day: 'numeric',
                              hour: '2-digit',
                              minute: '2-digit',
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
            <div className="flex items-center justify-between mt-4 pt-4 border-t border-border">
              <p className="text-xs text-muted-foreground">
                Page {page} of {totalPages}
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage((p) => p - 1)}
                >
                  Prev
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
