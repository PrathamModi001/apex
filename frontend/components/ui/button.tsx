import { Button as ButtonPrimitive } from "@base-ui/react/button"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "@/lib/utils"

const buttonVariants = cva(
  "group/button inline-flex shrink-0 items-center justify-center gap-1.5 rounded-sm border text-sm font-medium whitespace-nowrap transition-all duration-150 outline-none select-none cursor-pointer focus-visible:ring-2 focus-visible:ring-ring/50 active:not-aria-[haspopup]:translate-y-px disabled:pointer-events-none disabled:opacity-40 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-3.5",
  {
    variants: {
      variant: {
        default:
          "border-[var(--apex-amber)]/60 bg-[var(--apex-amber-glow)] text-[var(--apex-amber-bright)] hover:bg-[var(--apex-amber)]/20 hover:border-[var(--apex-amber)]/80",
        outline:
          "border-border bg-transparent text-muted-foreground hover:bg-muted/60 hover:text-foreground hover:border-border/80",
        secondary:
          "border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80",
        ghost:
          "border-transparent text-muted-foreground hover:bg-muted/60 hover:text-foreground",
        destructive:
          "border-[var(--apex-red)]/40 bg-[var(--apex-red)]/10 text-[var(--apex-red)] hover:bg-[var(--apex-red)]/20",
        success:
          "border-[var(--apex-green)]/40 bg-[var(--apex-green)]/10 text-[var(--apex-green)] hover:bg-[var(--apex-green)]/20",
        link: "border-transparent text-primary underline-offset-4 hover:underline",
      },
      size: {
        default: "h-8 px-3",
        xs:      "h-6 px-2 text-xs",
        sm:      "h-7 px-2.5 text-xs",
        lg:      "h-9 px-4",
        icon:    "h-8 w-8",
        "icon-sm": "h-7 w-7",
        "icon-xs": "h-6 w-6",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

function Button({
  className,
  variant = "default",
  size = "default",
  ...props
}: ButtonPrimitive.Props & VariantProps<typeof buttonVariants>) {
  return (
    <ButtonPrimitive
      data-slot="button"
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  )
}

export { Button, buttonVariants }
