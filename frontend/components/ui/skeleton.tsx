import * as React from 'react'
import { cn } from '@/lib/utils'

function Skeleton({ className, ...props }: React.ComponentProps<'div'>) {
  return (
    <div
      data-slot="skeleton"
      className={cn(
        'animate-pulse rounded-sm',
        'bg-gradient-to-r from-muted via-muted/60 to-muted',
        className
      )}
      {...props}
    />
  )
}

export { Skeleton }
