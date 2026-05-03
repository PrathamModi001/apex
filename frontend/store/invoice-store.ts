import { create } from 'zustand'

export interface LiveEvent {
  type: string
  invoice_id: string
  vendor_name: string
  risk_score: number
  decision: string
  [key: string]: unknown
}

interface InvoiceStore {
  liveEvents: LiveEvent[]
  addLiveEvent: (event: LiveEvent) => void
  clearEvents: () => void
}

export const useInvoiceStore = create<InvoiceStore>((set) => ({
  liveEvents: [],
  addLiveEvent: (event) =>
    set((s) => ({ liveEvents: [event, ...s.liveEvents].slice(0, 100) })),
  clearEvents: () => set({ liveEvents: [] }),
}))
