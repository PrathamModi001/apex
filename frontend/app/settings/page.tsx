'use client'

import { useState } from 'react'
import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Select } from '@/components/ui/select'
import { fetchPolicies, createPolicy, togglePolicy, fetchUsers, updateUserRole } from '@/lib/api'
import { Mail, MessageSquare, ToggleLeft, ToggleRight, Trash2, Plus, Users, Shield, Plug } from 'lucide-react'

interface Policy {
  id: string
  raw_text: string
  compiled_rule?: string
  active: boolean
  last_triggered?: string
}

interface User {
  id: string
  name: string
  email: string
  role: string
}

type Tab = 'integrations' | 'policies' | 'team'

// ---- Integrations Tab ----
function IntegrationsTab() {
  return (
    <div className="space-y-3">
      {/* Gmail */}
      <div className="rounded-sm border border-border bg-card overflow-hidden">
        <div className="flex items-start gap-3 px-5 py-4 border-b border-border/60">
          <div
            className="flex-shrink-0 w-7 h-7 rounded-sm flex items-center justify-center"
            style={{ background: 'rgba(72, 136, 248, 0.12)', border: '1px solid rgba(72, 136, 248, 0.25)' }}
          >
            <Mail className="h-3.5 w-3.5" style={{ color: 'var(--apex-blue)' }} strokeWidth={1.5} />
          </div>
          <div>
            <p className="text-xs font-semibold text-foreground">Gmail Integration</p>
            <p className="text-[10px] text-muted-foreground mt-0.5">
              Connect Gmail account to ingest invoices from AP Inbox label.
            </p>
          </div>
        </div>
        <div className="flex items-center justify-between px-5 py-3">
          <div>
            <p className="text-[10px] font-mono text-muted-foreground/60 uppercase tracking-wider">OAuth Status</p>
            <p className="text-[11px] text-muted-foreground mt-0.5">Not connected</p>
          </div>
          <a href="/auth/google">
            <Button size="sm" variant="outline" className="text-[11px]">
              Connect Gmail
            </Button>
          </a>
        </div>
      </div>

      {/* Telegram */}
      <div className="rounded-sm border border-border bg-card overflow-hidden">
        <div className="flex items-start gap-3 px-5 py-4 border-b border-border/60">
          <div
            className="flex-shrink-0 w-7 h-7 rounded-sm flex items-center justify-center"
            style={{ background: 'var(--apex-amber-glow)', border: '1px solid var(--apex-amber-line)' }}
          >
            <MessageSquare className="h-3.5 w-3.5" style={{ color: 'var(--apex-amber)' }} strokeWidth={1.5} />
          </div>
          <div>
            <p className="text-xs font-semibold text-foreground">Telegram Bot</p>
            <p className="text-[10px] text-muted-foreground mt-0.5">
              Receive invoices from Telegram bot messages.
            </p>
          </div>
        </div>
        <div className="flex items-center justify-between px-5 py-3 border-b border-border/60">
          <div>
            <p className="text-[10px] font-mono text-muted-foreground/60 uppercase tracking-wider">Bot Status</p>
            <p className="text-[11px] text-muted-foreground mt-0.5">Awaiting configuration</p>
          </div>
          <Badge variant="outline">Not Connected</Badge>
        </div>
        <div className="px-5 py-4">
          <p className="text-[9px] font-mono tracking-[0.15em] uppercase text-muted-foreground/60 mb-2">
            Setup Instructions
          </p>
          <ol className="space-y-1.5 text-[11px] text-muted-foreground list-none">
            {[
              'Create bot via @BotFather on Telegram',
              'Copy the bot token',
              <>Set <code className="font-mono text-[10px] px-1 py-0.5 rounded-[2px]" style={{ background: 'var(--background)', border: '1px solid var(--border)', color: 'var(--apex-amber)' }}>TELEGRAM_BOT_TOKEN</code> in environment</>,
              'Restart ingestor service',
            ].map((step, i) => (
              <li key={i} className="flex items-start gap-2">
                <span
                  className="flex-shrink-0 w-4 h-4 rounded-[2px] flex items-center justify-center text-[8px] font-mono font-bold mt-0.5"
                  style={{ background: 'var(--apex-amber-glow)', color: 'var(--apex-amber)', border: '1px solid var(--apex-amber-line)' }}
                >
                  {i + 1}
                </span>
                <span>{step}</span>
              </li>
            ))}
          </ol>
        </div>
      </div>
    </div>
  )
}

// ---- Policies Tab ----
function PoliciesTab() {
  const queryClient   = useQueryClient()
  const [text, setText]     = useState('')
  const [addError, setAddError] = useState<string | null>(null)

  const { data: policies, isLoading } = useQuery<Policy[]>({
    queryKey: ['policies'],
    queryFn: fetchPolicies,
  })

  const createMutation = useMutation({
    mutationFn: (t: string) => createPolicy(t),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] })
      setText('')
      setAddError(null)
    },
    onError: (err: Error) => setAddError(err.message),
  })

  const toggleMutation = useMutation({
    mutationFn: ({ id, active }: { id: string; active: boolean }) => togglePolicy(id, active),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['policies'] }),
  })

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      const BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const token = typeof window !== 'undefined' ? localStorage.getItem('apex_token') : null
      const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {}
      const res = await fetch(`${BASE_URL}/policies/${id}`, { method: 'DELETE', headers })
      if (!res.ok) throw new Error('Failed to delete policy')
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['policies'] }),
  })

  const policyList = Array.isArray(policies) ? policies : []

  return (
    <div className="space-y-3">
      {/* Add form */}
      <div className="rounded-sm border border-border bg-card overflow-hidden">
        <div className="px-5 pt-4 pb-3 border-b border-border/60">
          <p className="text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground">
            Add Policy
          </p>
          <p className="text-[10px] text-muted-foreground mt-0.5">
            Write a plain-English rule — agent compiles it automatically.
          </p>
        </div>
        <div className="px-5 py-4">
          <form
            onSubmit={(e) => {
              e.preventDefault()
              if (!text.trim()) return
              createMutation.mutate(text.trim())
            }}
            className="space-y-3"
          >
            <textarea
              className="w-full min-h-[80px] rounded-sm border border-input bg-[var(--input)] px-3 py-2 text-xs font-mono text-foreground outline-none resize-y placeholder:text-muted-foreground/50 focus-visible:border-[var(--apex-amber)]/60 focus-visible:ring-1 focus-visible:ring-[var(--apex-amber)]/20"
              placeholder="e.g. Flag any invoice from a new vendor with amount > $10,000"
              value={text}
              onChange={(e) => setText(e.target.value)}
            />
            {addError && (
              <p className="text-[10px] font-mono" style={{ color: 'var(--apex-red)' }}>{addError}</p>
            )}
            <Button
              type="submit"
              size="sm"
              disabled={createMutation.isPending || !text.trim()}
              className="gap-1.5"
            >
              <Plus className="h-3 w-3" />
              {createMutation.isPending ? 'Adding…' : 'Add Policy'}
            </Button>
          </form>
        </div>
      </div>

      {/* Policy list */}
      <div className="rounded-sm border border-border bg-card overflow-hidden">
        <div className="px-5 pt-4 pb-3 border-b border-border/60">
          <p className="text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground">
            Active Policies
          </p>
        </div>
        {isLoading ? (
          <div className="divide-y divide-border/50">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="px-5 py-4">
                <Skeleton className="h-4 w-full mb-2" />
                <Skeleton className="h-3 w-2/3" />
              </div>
            ))}
          </div>
        ) : policyList.length === 0 ? (
          <div className="px-5 py-10 text-center">
            <p className="text-[11px] font-mono text-muted-foreground/50">No policies configured.</p>
          </div>
        ) : (
          <div className="divide-y divide-border/50">
            {policyList.map((policy) => (
              <div key={policy.id} className="px-5 py-4 space-y-2">
                <div className="flex items-start justify-between gap-3">
                  <p className="text-xs text-foreground/90 flex-1 leading-relaxed">{policy.raw_text}</p>
                  <div className="flex items-center gap-1 shrink-0">
                    <Button
                      size="sm"
                      variant={policy.active ? 'default' : 'ghost'}
                      className="gap-1 h-7 text-[10px]"
                      onClick={() => toggleMutation.mutate({ id: policy.id, active: !policy.active })}
                      disabled={toggleMutation.isPending}
                    >
                      {policy.active
                        ? <ToggleRight className="h-3 w-3" />
                        : <ToggleLeft className="h-3 w-3 text-muted-foreground" />}
                      {policy.active ? 'On' : 'Off'}
                    </Button>
                    <Button
                      size="icon-sm"
                      variant="destructive"
                      className="h-7 w-7"
                      onClick={() => deleteMutation.mutate(policy.id)}
                      disabled={deleteMutation.isPending}
                    >
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  </div>
                </div>
                {policy.compiled_rule && (
                  <p
                    className="text-[9px] font-mono px-2 py-1 rounded-[2px]"
                    style={{ background: 'var(--background)', border: '1px solid var(--border)', color: 'var(--muted-foreground)' }}
                  >
                    {policy.compiled_rule}
                  </p>
                )}
                {policy.last_triggered && (
                  <p className="text-[9px] font-mono text-muted-foreground/50">
                    Last triggered: {new Date(policy.last_triggered).toLocaleString()}
                  </p>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

// ---- Team Tab ----
const ROLE_OPTIONS = ['admin', 'reviewer', 'viewer']

function roleBadgeVariant(role: string): 'default' | 'secondary' | 'info' | 'outline' {
  if (role === 'admin')    return 'default'
  if (role === 'reviewer') return 'secondary'
  return 'outline'
}

function TeamTab() {
  const queryClient = useQueryClient()

  const { data: users, isLoading } = useQuery<User[]>({
    queryKey: ['users'],
    queryFn: fetchUsers,
  })

  const updateRoleMutation = useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: string }) =>
      updateUserRole(userId, role),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['users'] }),
  })

  const userList = Array.isArray(users) ? users : []

  return (
    <div className="rounded-sm border border-border bg-card overflow-hidden">
      <div className="px-5 pt-4 pb-3 border-b border-border/60">
        <p className="text-[9px] font-mono tracking-[0.2em] uppercase text-muted-foreground">
          Team Members
        </p>
      </div>
      {isLoading ? (
        <div className="divide-y divide-border/50">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="flex items-center justify-between px-5 py-3">
              <div className="space-y-1.5">
                <Skeleton className="h-3 w-28" />
                <Skeleton className="h-2.5 w-40" />
              </div>
              <Skeleton className="h-7 w-24" />
            </div>
          ))}
        </div>
      ) : userList.length === 0 ? (
        <div className="px-5 py-10 text-center">
          <p className="text-[11px] font-mono text-muted-foreground/50">No team members found.</p>
        </div>
      ) : (
        <div className="divide-y divide-border/50">
          {userList.map((user) => (
            <div key={user.id} className="flex items-center justify-between px-5 py-3 gap-3">
              <div className="flex flex-col min-w-0">
                <span className="text-xs font-medium text-foreground truncate">{user.name || '—'}</span>
                <span className="text-[10px] font-mono text-muted-foreground truncate">{user.email}</span>
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <Badge variant={roleBadgeVariant(user.role)}>{user.role}</Badge>
                <Select
                  value={user.role}
                  onChange={(e) => updateRoleMutation.mutate({ userId: user.id, role: e.target.value })}
                  className="w-24 h-7"
                  disabled={updateRoleMutation.isPending}
                >
                  {ROLE_OPTIONS.map((r) => (
                    <option key={r} value={r}>{r}</option>
                  ))}
                </Select>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// ---- Main Page ----
const TABS: { id: Tab; label: string; icon: React.ComponentType<{ className?: string; strokeWidth?: number; style?: React.CSSProperties }> }[] = [
  { id: 'integrations', label: 'Integrations', icon: Plug },
  { id: 'policies',     label: 'Policies',     icon: Shield },
  { id: 'team',         label: 'Team',         icon: Users },
]

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState<Tab>('integrations')

  return (
    <div className="p-6 space-y-4 max-w-[900px]">

      {/* Header */}
      <div className="apex-fade-up">
        <p className="text-[9px] font-mono tracking-[0.25em] uppercase text-muted-foreground mb-1">
          Apex Intelligence
        </p>
        <h1
          className="text-2xl font-bold leading-none tracking-tight text-foreground"
          style={{ fontFamily: 'var(--font-syne, sans-serif)' }}
        >
          Settings
        </h1>
      </div>

      {/* Tab bar */}
      <div className="flex border-b border-border apex-fade-up apex-delay-1">
        {TABS.map(({ id, label, icon: Icon }) => {
          const active = activeTab === id
          return (
            <button
              key={id}
              className="flex items-center gap-1.5 px-4 py-2.5 text-xs font-medium border-b-2 transition-colors"
              style={{
                borderColor: active ? 'var(--apex-amber)' : 'transparent',
                color: active ? 'var(--apex-amber-bright)' : undefined,
              }}
              onClick={() => setActiveTab(id)}
            >
              <Icon
                className="h-3 w-3"
                strokeWidth={active ? 2.5 : 1.5}
                style={{ color: active ? 'var(--apex-amber)' : undefined }}
              />
              <span className={active ? '' : 'text-muted-foreground hover:text-foreground transition-colors'}>
                {label}
              </span>
            </button>
          )
        })}
      </div>

      {/* Content */}
      <div className="apex-fade-up apex-delay-2">
        {activeTab === 'integrations' && <IntegrationsTab />}
        {activeTab === 'policies'     && <PoliciesTab />}
        {activeTab === 'team'         && <TeamTab />}
      </div>
    </div>
  )
}
