const BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
const AGENT_URL = process.env.NEXT_PUBLIC_AGENT_SERVICE_URL || 'http://localhost:8000'

export { AGENT_URL }

function authHeaders(): Record<string, string> {
  const token = typeof window !== 'undefined' ? localStorage.getItem('apex_token') : null
  return token ? { Authorization: `Bearer ${token}` } : {}
}

export async function fetchInvoices(filters?: Record<string, string>) {
  try {
    const params = new URLSearchParams(filters)
    const res = await fetch(`${BASE_URL}/invoices?${params}`, { headers: authHeaders() })
    if (!res.ok) throw new Error('Failed to fetch invoices')
    return res.json()
  } catch {
    return { invoices: [], total: 0 }
  }
}

export async function fetchInvoice(id: string) {
  try {
    const res = await fetch(`${BASE_URL}/invoices/${id}`, { headers: authHeaders() })
    if (!res.ok) throw new Error('Failed to fetch invoice')
    return res.json()
  } catch {
    return null
  }
}

export async function fetchDecision(id: string) {
  try {
    const res = await fetch(`${BASE_URL}/invoices/${id}/decision`, { headers: authHeaders() })
    if (!res.ok) throw new Error('Failed to fetch decision')
    return res.json()
  } catch {
    return null
  }
}

export async function fetchAuditChain(id: string) {
  try {
    const res = await fetch(`${BASE_URL}/audit/${id}`, { headers: authHeaders() })
    if (!res.ok) throw new Error('Failed to fetch audit chain')
    return res.json()
  } catch {
    return []
  }
}

export async function approveInvoice(id: string) {
  const res = await fetch(`${BASE_URL}/invoices/${id}/approve`, {
    method: 'POST',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error('Failed to approve invoice')
  return res.json()
}

export async function rejectInvoice(id: string, reason: string) {
  const res = await fetch(`${BASE_URL}/invoices/${id}/reject`, {
    method: 'POST',
    headers: { ...authHeaders(), 'Content-Type': 'application/json' },
    body: JSON.stringify({ reason }),
  })
  if (!res.ok) throw new Error('Failed to reject invoice')
  return res.json()
}

export async function fetchVendors() {
  try {
    const res = await fetch(`${BASE_URL}/vendors`, { headers: authHeaders() })
    if (!res.ok) throw new Error('Failed to fetch vendors')
    return res.json()
  } catch {
    return []
  }
}

export async function fetchVendor(id: string) {
  try {
    const res = await fetch(`${BASE_URL}/vendors/${id}`, { headers: authHeaders() })
    if (!res.ok) throw new Error('Failed to fetch vendor')
    return res.json()
  } catch {
    return null
  }
}

export async function fetchPolicies() {
  try {
    const res = await fetch(`${BASE_URL}/policies`, { headers: authHeaders() })
    if (!res.ok) throw new Error('Failed to fetch policies')
    return res.json()
  } catch {
    return []
  }
}

export async function createPolicy(rawText: string) {
  const res = await fetch(`${BASE_URL}/policies`, {
    method: 'POST',
    headers: { ...authHeaders(), 'Content-Type': 'application/json' },
    body: JSON.stringify({ raw_text: rawText }),
  })
  if (!res.ok) throw new Error('Failed to create policy')
  return res.json()
}

export async function togglePolicy(id: string, active: boolean) {
  const res = await fetch(`${BASE_URL}/policies/${id}`, {
    method: 'PATCH',
    headers: { ...authHeaders(), 'Content-Type': 'application/json' },
    body: JSON.stringify({ active }),
  })
  if (!res.ok) throw new Error('Failed to toggle policy')
  return res.json()
}

export async function fetchUsers() {
  try {
    const res = await fetch(`${BASE_URL}/users`, { headers: authHeaders() })
    if (!res.ok) throw new Error('Failed to fetch users')
    return res.json()
  } catch {
    return []
  }
}

export async function updateUserRole(userId: string, role: string) {
  const res = await fetch(`${BASE_URL}/admin/users/${userId}/role`, {
    method: 'POST',
    headers: { ...authHeaders(), 'Content-Type': 'application/json' },
    body: JSON.stringify({ role }),
  })
  if (!res.ok) throw new Error('Failed to update user role')
  return res.json()
}

export async function fetchFraudGraph(id: string) {
  try {
    const res = await fetch(`/api/fraud-graph/${id}`, { headers: authHeaders() })
    if (!res.ok) throw new Error('Failed to fetch fraud graph')
    return res.json()
  } catch {
    return { nodes: [], edges: [] }
  }
}
