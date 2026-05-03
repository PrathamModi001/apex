'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { RiskBadge } from '@/components/risk-badge'
import { fetchVendors, fetchVendor } from '@/lib/api'

interface Vendor {
  id: string
  name: string
  risk_score: number
  invoice_count: number
  bank_accounts?: string[]
  correction_count?: number
  [key: string]: unknown
}

interface VendorDetail {
  id: string
  name: string
  risk_score: number
  invoice_count: number
  bank_accounts?: string[]
  correction_count?: number
  email?: string
  phone?: string
  address?: string
  [key: string]: unknown
}

function VendorPanel({
  vendor,
  onClose,
}: {
  vendor: VendorDetail
  onClose: () => void
}) {
  return (
    <div className="fixed inset-y-0 right-0 z-50 w-80 border-l border-border bg-card shadow-xl flex flex-col">
      <div className="flex items-center justify-between p-4 border-b border-border">
        <h2 className="text-sm font-semibold">{vendor.name}</h2>
        <button
          className="text-muted-foreground hover:text-foreground transition-colors text-lg leading-none"
          onClick={onClose}
        >
          ✕
        </button>
      </div>
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">Risk Score</span>
          <RiskBadge score={vendor.risk_score ?? 0} />
        </div>

        <div className="space-y-2 text-xs">
          <Row label="Invoice Count" value={vendor.invoice_count} />
          <Row label="Correction Count" value={vendor.correction_count ?? 0} />
          <Row label="Email" value={vendor.email} />
          <Row label="Phone" value={vendor.phone} />
          <Row label="Address" value={vendor.address} />
        </div>

        {vendor.bank_accounts && vendor.bank_accounts.length > 0 && (
          <div>
            <p className="text-xs text-muted-foreground mb-1">
              Bank Accounts ({vendor.bank_accounts.length})
            </p>
            <div className="space-y-1">
              {vendor.bank_accounts.map((acct, i) => (
                <span
                  key={i}
                  className="block font-mono text-[10px] bg-muted rounded px-2 py-1 text-foreground/80"
                >
                  {acct}
                </span>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function Row({ label, value }: { label: string; value?: string | number | null }) {
  return (
    <div className="flex items-center justify-between border-b border-border/40 pb-1">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium">{value ?? '—'}</span>
    </div>
  )
}

export default function VendorsPage() {
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const { data: vendors, isLoading } = useQuery<Vendor[]>({
    queryKey: ['vendors'],
    queryFn: fetchVendors,
    select: (data) =>
      Array.isArray(data)
        ? [...data].sort((a, b) => (b.risk_score ?? 0) - (a.risk_score ?? 0))
        : [],
    refetchInterval: 30_000,
  })

  const { data: selectedVendor, isLoading: detailLoading } = useQuery<VendorDetail | null>({
    queryKey: ['vendor', selectedId],
    queryFn: () => (selectedId ? fetchVendor(selectedId) : Promise.resolve(null)),
    enabled: !!selectedId,
  })

  const vendorList = vendors ?? []

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Vendors</h1>
        <p className="text-sm text-muted-foreground">All known vendors, sorted by risk</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center justify-between">
            <span>Vendor List</span>
            {!isLoading && (
              <span className="text-xs text-muted-foreground font-normal">
                {vendorList.length} vendor{vendorList.length !== 1 ? 's' : ''}
              </span>
            )}
          </CardTitle>
          <CardDescription>Click a row to see vendor details</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border text-xs text-muted-foreground">
                  <th className="py-2 px-3 text-left font-medium">Vendor Name</th>
                  <th className="py-2 px-3 text-left font-medium">Risk Score</th>
                  <th className="py-2 px-3 text-right font-medium">Invoices</th>
                  <th className="py-2 px-3 text-right font-medium">Bank Accounts</th>
                  <th className="py-2 px-3 text-right font-medium">Corrections</th>
                </tr>
              </thead>
              <tbody>
                {isLoading ? (
                  Array.from({ length: 5 }).map((_, i) => (
                    <tr key={i} className="border-b border-border/50">
                      {Array.from({ length: 5 }).map((_, j) => (
                        <td key={j} className="py-2.5 px-3">
                          <Skeleton className="h-4 w-full" />
                        </td>
                      ))}
                    </tr>
                  ))
                ) : vendorList.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="py-12 text-center text-muted-foreground text-sm">
                      No vendors found.
                    </td>
                  </tr>
                ) : (
                  vendorList.map((v) => (
                    <tr
                      key={v.id}
                      className={`border-b border-border/50 hover:bg-muted/30 cursor-pointer transition-colors ${
                        selectedId === v.id ? 'bg-primary/5' : ''
                      }`}
                      onClick={() => setSelectedId(v.id === selectedId ? null : v.id)}
                    >
                      <td className="py-2.5 px-3 font-medium">{v.name}</td>
                      <td className="py-2.5 px-3">
                        <RiskBadge score={v.risk_score ?? 0} />
                      </td>
                      <td className="py-2.5 px-3 text-right tabular-nums text-muted-foreground">
                        {v.invoice_count ?? 0}
                      </td>
                      <td className="py-2.5 px-3 text-right tabular-nums text-muted-foreground">
                        {v.bank_accounts?.length ?? 0}
                      </td>
                      <td className="py-2.5 px-3 text-right tabular-nums text-muted-foreground">
                        {v.correction_count ?? 0}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      {/* Vendor Detail Side Panel */}
      {selectedId && (
        <>
          {/* Backdrop */}
          <div
            className="fixed inset-0 z-40 bg-background/60 backdrop-blur-sm"
            onClick={() => setSelectedId(null)}
          />
          {detailLoading || !selectedVendor ? (
            <div className="fixed inset-y-0 right-0 z-50 w-80 border-l border-border bg-card shadow-xl p-4 space-y-3">
              <Skeleton className="h-6 w-40" />
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-4 w-full" />
              ))}
            </div>
          ) : (
            <VendorPanel vendor={selectedVendor} onClose={() => setSelectedId(null)} />
          )}
        </>
      )}
    </div>
  )
}
