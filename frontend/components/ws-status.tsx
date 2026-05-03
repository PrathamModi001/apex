'use client'

interface WsStatusProps {
  connected: boolean
}

export function WsStatus({ connected }: WsStatusProps) {
  return (
    <span className="flex items-center gap-1.5 text-xs font-medium">
      <span className="relative flex h-2 w-2">
        {connected && (
          <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-400 opacity-75" />
        )}
        <span
          className={`relative inline-flex h-2 w-2 rounded-full ${
            connected ? 'bg-green-500' : 'bg-red-500'
          }`}
        />
      </span>
      <span className={connected ? 'text-green-400' : 'text-red-400'}>
        {connected ? 'Live' : 'Offline'}
      </span>
    </span>
  )
}
