'use client'

export function RiskBadge({ score }: { score: number }) {
  const color =
    score < 30
      ? 'bg-green-500/20 text-green-400'
      : score < 60
      ? 'bg-yellow-500/20 text-yellow-400'
      : 'bg-red-500/20 text-red-400'
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${color}`}
    >
      {score.toFixed(0)}
    </span>
  )
}
