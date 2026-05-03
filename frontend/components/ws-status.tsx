'use client'

interface WsStatusProps {
  connected: boolean
}

export function WsStatus({ connected }: WsStatusProps) {
  return (
    <span className="flex items-center gap-2 text-[10px] font-mono tracking-wider uppercase">
      <span className="relative flex h-[6px] w-[6px]">
        {connected && (
          <span
            className="absolute inline-flex h-full w-full rounded-full opacity-60 animate-ping"
            style={{ background: 'var(--apex-green)' }}
          />
        )}
        <span
          className="relative inline-flex h-[6px] w-[6px] rounded-full"
          style={{
            background: connected ? 'var(--apex-green)' : 'var(--apex-red)',
            boxShadow: connected
              ? '0 0 6px var(--apex-green)'
              : '0 0 4px var(--apex-red)',
          }}
        />
      </span>
      <span style={{ color: connected ? 'var(--apex-green)' : 'var(--apex-red)' }}>
        {connected ? 'Live' : 'Offline'}
      </span>
    </span>
  )
}
