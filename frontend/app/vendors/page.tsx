'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Skeleton } from '@/components/ui/skeleton'
import { RiskBadge } from '@/components/risk-badge'
import { fetchVendors, fetchVendor } from '@/lib/api'
import { X, Building2, CreditCard, FileText, AlertTriangle } from 'lucide-react'

interface Vendor {
  id: string
  name: string
  risk_score: number
  invoice_count: number
  bank_accounts?: string[]
  correction_count?: number
  [key: string]: unknown
}

interface VendorDetail extends Vendor {
  email?: string
  phone?: string
  address?: string
}

function PanelRow({ label, value }: { label: string; value?: string | number | null }) {
  return (
    <div className="flex items-center justify-between py-2 border-b border-border/50 last:border-0">
      <span className="text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60">
        {label}
      </span>
      <span className="text-[11px] font-mono text-foreground/80 max-w-[140px] truncate text-right">
        {value ?? <span className="text-muted-foreground/30">—</span>}
      </span>
    </div>
  )
}

function VendorPanel({ vendor, onClose }: { vendor: VendorDetail; onClose: () => void }) {
  const riskColor =
    vendor.risk_score < 30
      ? 'var(--apex-green)'
      : vendor.risk_score < 60
      ? 'var(--apex-amber)'
      : 'var(--apex-red)'

  return (
    <div
      className="fixed inset-y-0 right-0 w-80 flex flex-col border-l border-border bg-card"
      style={{ zIndex: 50 }}
    >
      {/* Panel header */}
      <div className="flex items-start justify-between px-5 pt-5 pb-4 border-b border-border/60">
        <div>
          <p className="text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground/60 mb-1">
            Vendor Detail
          </p>
          <h2
            className="text-base font-bold leading-tight text-foreground"
            style={{ fontFamily: 'var(--font-syne, sans-serif)' }}
          >
            {vendor.name}
          </h2>
        </div>
        <button
          className="text-muted-foreground/50 hover:text-foreground transition-colors mt-1"
          onClick={onClose}
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Risk score bar */}
      <div className="px-5 py-4 border-b border-border/60">
        <p className="text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 mb-2">
          Risk Score
        </p>
        <div className="flex items-center gap-3">
          <span
            className="text-2xl font-bold font-mono tabular-nums leading-none"
            style={{ color: riskColor }}
          >
            {(vendor.risk_score ?? 0).toFixed(0)}
          </span>
          <div className="flex-1">
            <div className="h-1.5 rounded-full overflow-hidden" style={{ background: 'var(--border)' }}>
              <div
                className="h-full rounded-full transition-all duration-700"
                style={{ width: `${vendor.risk_score ?? 0}%`, background: riskColor }}
              />
            </div>
          </div>
        </div>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-3 border-b border-border/60">
        <div className="px-4 py-3 text-center border-r border-border/60">
          <p className="text-[9px] font-mono uppercase text-muted-foreground/50 mb-0.5">Invoices</p>
          <p className="text-base font-mono font-bold text-foreground">{vendor.invoice_count ?? 0}</p>
        </div>
        <div className="px-4 py-3 text-center border-r border-border/60">
          <p className="text-[9px] font-mono uppercase text-muted-foreground/50 mb-0.5">Accounts</p>
          <p className="text-base font-mono font-bold text-foreground">
            {vendor.bank_accounts?.length ?? 0}
          </p>
        </div>
        <div className="px-4 py-3 text-center">
          <p className="text-[9px] font-mono uppercase text-muted-foreground/50 mb-0.5">Corrections</p>
          <p className="text-base font-mono font-bold" style={{ color: (vendor.correction_count ?? 0) > 0 ? 'var(--apex-orange)' : 'var(--foreground)' }}>
            {vendor.correction_count ?? 0}
          </p>
        </div>
      </div>

      {/* Fields */}
      <div className="flex-1 overflow-y-auto px-5 py-4 space-y-0">
        <PanelRow label="Email"   value={vendor.email} />
        <PanelRow label="Phone"   value={vendor.phone} />
        <PanelRow label="Address" value={vendor.address} />
      </div>

      {/* Bank accounts */}
      {vendor.bank_accounts && vendor.bank_accounts.length > 0 && (
        <div className="px-5 pb-5 border-t border-border/60 pt-4">
          <p className="text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 mb-2 flex items-center gap-1">
            <CreditCard className="h-2.5 w-2.5" />
            Bank Accounts
          </p>
          <div className="space-y-1">
            {vendor.bank_accounts.map((acct, i) => (
              <span
                key={i}
                className="block font-mono text-[10px] px-2 py-1.5 rounded-[3px] text-foreground/70"
                style={{ background: 'var(--background)', border: '1px solid var(--border)' }}
              >
                {acct}
              </span>
            ))}
          </div>
        </div>
      )}
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
            Vendors
          </h1>
        </div>
        {!isLoading && (
          <span className="text-[10px] font-mono text-muted-foreground">
            {vendorList.length} vendor{vendorList.length !== 1 ? 's' : ''}
          </span>
        )}
      </div>

      {/* Table */}
      <div className="rounded-sm border border-border bg-card overflow-hidden apex-fade-up apex-delay-1">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border/70 bg-muted/30">
                <th className="px-4 py-2.5 text-left text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 font-normal">
                  <span className="flex items-center gap-1.5">
                    <Building2 className="h-2.5 w-2.5" /> Vendor
                  </span>
                </th>
                <th className="px-4 py-2.5 text-left text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 font-normal">
                  Risk
                </th>
                <th className="px-4 py-2.5 text-right text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 font-normal">
                  <span className="flex items-center gap-1 justify-end">
                    <FileText className="h-2.5 w-2.5" /> Invoices
                  </span>
                </th>
                <th className="px-4 py-2.5 text-right text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 font-normal">
                  <span className="flex items-center gap-1 justify-end">
                    <CreditCard className="h-2.5 w-2.5" /> Accounts
                  </span>
                </th>
                <th className="px-4 py-2.5 text-right text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 font-normal">
                  <span className="flex items-center gap-1 justify-end">
                    <AlertTriangle className="h-2.5 w-2.5" /> Corrections
                  </span>
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/50">
              {isLoading ? (
                Array.from({ length: 6 }).map((_, i) => (
                  <tr key={i} className="animate-pulse">
                    {Array.from({ length: 5 }).map((_, j) => (
                      <td key={j} className="px-4 py-3">
                        <div className="h-3 rounded-sm bg-muted w-full" />
                      </td>
                    ))}
                  </tr>
                ))
              ) : vendorList.length === 0 ? (
                <tr>
                  <td colSpan={5} className="py-16 text-center text-[11px] font-mono text-muted-foreground/50">
                    No vendors found.
                  </td>
                </tr>
              ) : (
                vendorList.map((v) => {
                  const isSelected = selectedId === v.id
                  return (
                    <tr
                      key={v.id}
                      className="group cursor-pointer transition-colors"
                      style={{
                        background: isSelected ? 'var(--apex-amber-glow)' : undefined,
                      }}
                      onMouseEnter={(e) => {
                        if (!isSelected) (e.currentTarget as HTMLElement).style.background = 'var(--apex-amber-glow)'
                      }}
                      onMouseLeave={(e) => {
                        if (!isSelected) (e.currentTarget as HTMLElement).style.background = ''
                      }}
                      onClick={() => setSelectedId(v.id === selectedId ? null : v.id)}
                    >
                      <td className="px-4 py-3 font-medium text-xs text-foreground/90">
                        <span
                          className="flex items-center gap-2"
                          style={{ color: isSelected ? 'var(--apex-amber-bright)' : undefined }}
                        >
                          {isSelected && (
                            <span
                              className="inline-block w-1 h-3 rounded-full flex-shrink-0"
                              style={{ background: 'var(--apex-amber)' }}
                            />
                          )}
                          {v.name}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <RiskBadge score={v.risk_score ?? 0} />
                      </td>
                      <td className="px-4 py-3 text-right font-mono text-[11px] tabular-nums text-muted-foreground">
                        {v.invoice_count ?? 0}
                      </td>
                      <td className="px-4 py-3 text-right font-mono text-[11px] tabular-nums text-muted-foreground">
                        {v.bank_accounts?.length ?? 0}
                      </td>
                      <td className="px-4 py-3 text-right font-mono text-[11px] tabular-nums"
                        style={{ color: (v.correction_count ?? 0) > 0 ? 'var(--apex-orange)' : 'var(--muted-foreground)' }}
                      >
                        {v.correction_count ?? 0}
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Side panel */}
      {selectedId && (
        <>
          <div
            className="fixed inset-0 bg-background/50 backdrop-blur-[2px]"
            style={{ zIndex: 40 }}
            onClick={() => setSelectedId(null)}
          />
          {detailLoading || !selectedVendor ? (
            <div className="fixed inset-y-0 right-0 w-80 border-l border-border bg-card p-5 space-y-3" style={{ zIndex: 50 }}>
              <Skeleton className="h-6 w-40" />
              <Skeleton className="h-4 w-24" />
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
