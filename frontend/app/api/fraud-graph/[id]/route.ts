import { NextRequest, NextResponse } from 'next/server'

const AGENT_URL = process.env.AGENT_SERVICE_URL || process.env.NEXT_PUBLIC_AGENT_SERVICE_URL || 'http://localhost:8000'

export async function GET(
  _request: NextRequest,
  { params }: { params: { id: string } },
) {
  try {
    const res = await fetch(`${AGENT_URL}/invoices/${params.id}/fraud-graph`)
    if (!res.ok) return NextResponse.json({ nodes: [], edges: [] })
    return NextResponse.json(await res.json())
  } catch {
    return NextResponse.json({ nodes: [], edges: [] })
  }
}
