import * as React from 'react'
import { cn } from '@/lib/utils'

function Input({ className, type, ...props }: React.ComponentProps<'input'>) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        'flex h-8 w-full rounded-sm border border-input bg-[var(--input)] px-3 py-1',
        'text-xs font-mono text-foreground',
        'outline-none transition-colors',
        'placeholder:text-muted-foreground/50',
        'focus-visible:border-[var(--apex-amber)]/60 focus-visible:ring-1 focus-visible:ring-[var(--apex-amber)]/20',
        'disabled:cursor-not-allowed disabled:opacity-40',
        '[appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none',
        className
      )}
      {...props}
    />
  )
}

export { Input }
