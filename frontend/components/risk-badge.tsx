'use client'

export function RiskBadge({ score }: { score: number }) {
  const pct = Math.min(100, Math.max(0, score || 0))
  const color =
    pct < 30
      ? 'var(--apex-green)'
      : pct < 60
      ? 'var(--apex-amber)'
      : 'var(--apex-red)'

  return (
    <span
      className="inline-flex items-center gap-1.5 font-mono text-[11px] font-medium tabular-nums"
      style={{ color }}
    >
      <span
        className="inline-block w-8 h-[3px] rounded-full overflow-hidden"
        style={{ background: 'var(--border)' }}
      >
        <span
          className="block h-full rounded-full transition-all duration-700"
          style={{ width: `${pct}%`, background: color }}
        />
      </span>
      {pct.toFixed(0)}
    </span>
  )
}

export function RiskBadgeLarge({ score }: { score: number }) {
  const pct = Math.min(100, Math.max(0, score || 0))
  const color =
    pct < 30
      ? 'var(--apex-green)'
      : pct < 60
      ? 'var(--apex-amber)'
      : 'var(--apex-red)'
  const label = pct < 30 ? 'LOW' : pct < 60 ? 'MEDIUM' : 'HIGH'

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-end gap-2">
        <span
          className="text-3xl font-bold font-mono tabular-nums leading-none"
          style={{ color }}
        >
          {pct.toFixed(0)}
        </span>
        <span className="text-xs font-mono text-muted-foreground mb-1">/ 100</span>
      </div>
      <div className="h-1.5 rounded-full overflow-hidden" style={{ background: 'var(--border)' }}>
        <div
          className="h-full rounded-full transition-all duration-700"
          style={{ width: `${pct}%`, background: color }}
        />
      </div>
      <span
        className="text-[10px] font-mono tracking-[0.15em] uppercase"
        style={{ color }}
      >
        {label} RISK
      </span>
    </div>
  )
}
