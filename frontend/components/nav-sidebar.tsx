'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { WsStatus } from '@/components/ws-status'
import { useWebSocket } from '@/lib/ws'
import { useInvoiceStore, LiveEvent } from '@/store/invoice-store'
import { useEffect } from 'react'
import { cn } from '@/lib/utils'
import {
  LayoutDashboard,
  FileText,
  Building2,
  Settings2,
  ShieldCheck,
} from 'lucide-react'

const WS_URL =
  typeof window !== 'undefined'
    ? `${process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080'}/ws?token=${
        localStorage.getItem('apex_token') || ''
      }`
    : 'ws://localhost:8080/ws'

const navItems = [
  { href: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { href: '/invoices',  label: 'Invoices',  icon: FileText },
  { href: '/vendors',   label: 'Vendors',   icon: Building2 },
  { href: '/settings',  label: 'Settings',  icon: Settings2 },
]

function WsConnector() {
  const { connected, messages } = useWebSocket(WS_URL)
  const addLiveEvent = useInvoiceStore((s) => s.addLiveEvent)

  useEffect(() => {
    if (messages.length > 0) {
      const latest = messages[0] as LiveEvent
      if (latest?.type === 'invoice.decision') {
        addLiveEvent(latest)
      }
    }
  }, [messages, addLiveEvent])

  return <WsStatus connected={connected} />
}

export function NavSidebar() {
  const pathname = usePathname()

  return (
    <nav className="fixed inset-y-0 left-0 z-30 flex w-56 flex-col border-r border-border bg-[var(--background)]">
      {/* Logo */}
      <div className="flex h-14 items-center gap-3 px-4 border-b border-border">
        {/* Diamond mark */}
        <div className="relative flex-shrink-0">
          <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
            <polygon
              points="9,1 17,9 9,17 1,9"
              fill="var(--apex-amber)"
              opacity="0.9"
            />
            <polygon
              points="9,4 14,9 9,14 4,9"
              fill="var(--background)"
              opacity="0.6"
            />
          </svg>
        </div>
        <div className="flex flex-col leading-none">
          <span
            className="text-base font-bold tracking-tight text-foreground"
            style={{ fontFamily: 'var(--font-syne, sans-serif)', letterSpacing: '-0.02em' }}
          >
            APEX
          </span>
          <span
            className="text-[9px] font-mono tracking-[0.18em] uppercase"
            style={{ color: 'var(--apex-amber)', opacity: 0.8 }}
          >
            Intelligence
          </span>
        </div>
      </div>

      {/* Nav items */}
      <div className="flex-1 overflow-y-auto py-3">
        <div className="px-2 mb-1">
          <p className="px-3 pb-1 text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground/60">
            Navigation
          </p>
        </div>
        <ul className="space-y-0.5 px-2">
          {navItems.map((item) => {
            const active =
              pathname === item.href || pathname.startsWith(item.href + '/')
            const Icon = item.icon
            return (
              <li key={item.href}>
                <Link
                  href={item.href}
                  className={cn(
                    'group flex items-center gap-2.5 rounded-sm px-3 py-2 text-sm font-medium transition-all duration-150',
                    active
                      ? 'border-l-2 border-[var(--apex-amber)] pl-[10px] bg-[var(--apex-amber-glow)] text-[var(--apex-amber-bright)]'
                      : 'border-l-2 border-transparent text-muted-foreground hover:bg-muted/60 hover:text-foreground'
                  )}
                >
                  <Icon
                    className={cn(
                      'h-3.5 w-3.5 flex-shrink-0 transition-colors',
                      active ? 'text-[var(--apex-amber)]' : 'text-muted-foreground group-hover:text-foreground'
                    )}
                    strokeWidth={active ? 2.5 : 2}
                  />
                  {item.label}
                </Link>
              </li>
            )
          })}
        </ul>

        {/* Divider */}
        <div className="my-3 mx-2 border-t border-border/60" />

        {/* Audit link */}
        <ul className="px-2">
          <li>
            <Link
              href="/audit"
              className={cn(
                'group flex items-center gap-2.5 rounded-sm px-3 py-2 text-sm font-medium transition-all duration-150',
                pathname.startsWith('/audit')
                  ? 'border-l-2 border-[var(--apex-amber)] pl-[10px] bg-[var(--apex-amber-glow)] text-[var(--apex-amber-bright)]'
                  : 'border-l-2 border-transparent text-muted-foreground hover:bg-muted/60 hover:text-foreground'
              )}
            >
              <ShieldCheck
                className={cn(
                  'h-3.5 w-3.5 flex-shrink-0',
                  pathname.startsWith('/audit') ? 'text-[var(--apex-amber)]' : 'text-muted-foreground group-hover:text-foreground'
                )}
                strokeWidth={2}
              />
              Audit
            </Link>
          </li>
        </ul>
      </div>

      {/* Footer: WS status */}
      <div className="border-t border-border px-4 py-3">
        <WsConnector />
        <p className="mt-1.5 text-[9px] font-mono tracking-wider text-muted-foreground/40 uppercase">
          apex v0.1.0
        </p>
      </div>
    </nav>
  )
}
