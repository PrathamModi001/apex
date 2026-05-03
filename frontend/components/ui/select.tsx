import * as React from 'react'
import { cn } from '@/lib/utils'

function Select({ className, children, ...props }: React.ComponentProps<'select'>) {
  return (
    <select
      data-slot="select"
      className={cn(
        'flex h-8 w-full rounded-sm border border-input bg-[var(--input)] px-2 py-1',
        'text-xs font-mono text-foreground',
        'outline-none transition-colors appearance-none cursor-pointer',
        'focus-visible:border-[var(--apex-amber)]/60 focus-visible:ring-1 focus-visible:ring-[var(--apex-amber)]/20',
        'disabled:cursor-not-allowed disabled:opacity-40',
        className
      )}
      {...props}
    >
      {children}
    </select>
  )
}

export { Select }
