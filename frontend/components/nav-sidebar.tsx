'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { WsStatus } from '@/components/ws-status'
import { useWebSocket } from '@/lib/ws'
import { useInvoiceStore, LiveEvent } from '@/store/invoice-store'
import { useEffect } from 'react'
import { cn } from '@/lib/utils'

const WS_URL =
  typeof window !== 'undefined'
    ? `${process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080'}/ws?token=${
        localStorage.getItem('apex_token') || ''
      }`
    : 'ws://localhost:8080/ws'

const navItems = [
  { href: '/dashboard', label: 'Dashboard' },
  { href: '/invoices', label: 'Invoices' },
  { href: '/vendors', label: 'Vendors' },
  { href: '/settings', label: 'Settings' },
]

function WsConnector() {
  const { connected, messages } = useWebSocket(WS_URL)
  const addLiveEvent = useInvoiceStore((s) => s.addLiveEvent)

  useEffect(() => {
    if (messages.length > 0) {
      const latest = messages[0] as LiveEvent
      if (latest && latest.type === 'invoice.decision') {
        addLiveEvent(latest)
      }
    }
  }, [messages, addLiveEvent])

  return <WsStatus connected={connected} />
}

export function NavSidebar() {
  const pathname = usePathname()

  return (
    <nav className="fixed inset-y-0 left-0 z-30 flex w-56 flex-col border-r border-border bg-card">
      <div className="flex h-14 items-center px-4 border-b border-border">
        <span className="text-base font-bold tracking-tight text-foreground">APEX</span>
        <span className="ml-1 text-xs text-muted-foreground">AP Agent</span>
      </div>

      <div className="flex-1 overflow-y-auto py-4">
        <ul className="space-y-0.5 px-2">
          {navItems.map((item) => (
            <li key={item.href}>
              <Link
                href={item.href}
                className={cn(
                  'flex items-center rounded-md px-3 py-2 text-sm font-medium transition-colors',
                  pathname === item.href || pathname.startsWith(item.href + '/')
                    ? 'bg-primary/10 text-primary'
                    : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                )}
              >
                {item.label}
              </Link>
            </li>
          ))}
        </ul>
      </div>

      <div className="border-t border-border p-4">
        <WsConnector />
      </div>
    </nav>
  )
}
