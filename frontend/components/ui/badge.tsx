import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "@/lib/utils"
import * as React from "react"

const badgeVariants = cva(
  "inline-flex h-[18px] w-fit shrink-0 items-center justify-center gap-1 overflow-hidden rounded-[3px] border px-1.5 text-[10px] font-mono font-medium tracking-wide whitespace-nowrap transition-colors",
  {
    variants: {
      variant: {
        default:
          "border-[var(--apex-amber)]/40 bg-[var(--apex-amber-glow)] text-[var(--apex-amber-bright)]",
        secondary:
          "border-[var(--apex-orange)]/40 bg-[var(--apex-orange)]/10 text-[var(--apex-orange)]",
        destructive:
          "border-[var(--apex-red)]/40 bg-[var(--apex-red)]/10 text-[var(--apex-red)]",
        success:
          "border-[var(--apex-green)]/40 bg-[var(--apex-green)]/10 text-[var(--apex-green)]",
        info:
          "border-[var(--apex-blue)]/40 bg-[var(--apex-blue)]/10 text-[var(--apex-blue)]",
        outline:
          "border-border text-muted-foreground",
        ghost:
          "border-transparent bg-muted text-muted-foreground",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)

function Badge({
  className,
  variant = "default",
  ...props
}: React.ComponentProps<"span"> & VariantProps<typeof badgeVariants>) {
  return (
    <span
      className={cn(badgeVariants({ variant }), className)}
      {...props}
    />
  )
}

export { Badge, badgeVariants }
