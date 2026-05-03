'use client'

import { useState } from 'react'
import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Select } from '@/components/ui/select'
import { fetchPolicies, createPolicy, togglePolicy, fetchUsers, updateUserRole } from '@/lib/api'

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
    <div className="space-y-6">
      {/* Gmail */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            📧 Gmail Integration
          </CardTitle>
          <CardDescription>Connect your Gmail account to ingest invoices automatically.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">OAuth Status</p>
              <p className="text-xs text-muted-foreground">Not connected</p>
            </div>
            <a href="/auth/google">
              <Button size="sm" variant="outline">
                Connect Gmail
              </Button>
            </a>
          </div>
        </CardContent>
      </Card>

      {/* Telegram */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            📱 Telegram Bot
          </CardTitle>
          <CardDescription>Receive invoices via Telegram bot messages.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Bot Status</p>
              <p className="text-xs text-muted-foreground">Awaiting configuration</p>
            </div>
            <Badge variant="outline">Not Connected</Badge>
          </div>
          <div className="bg-muted rounded-lg p-3 text-xs text-muted-foreground space-y-1">
            <p className="font-medium text-foreground">Setup Instructions</p>
            <ol className="list-decimal list-inside space-y-0.5">
              <li>Create a bot via @BotFather on Telegram</li>
              <li>Copy the bot token</li>
              <li>Set <code className="bg-background px-1 rounded">TELEGRAM_BOT_TOKEN</code> in your environment</li>
              <li>Restart the ingestor service</li>
            </ol>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

// ---- Policies Tab ----
function PoliciesTab() {
  const queryClient = useQueryClient()
  const [newPolicyText, setNewPolicyText] = useState('')
  const [addError, setAddError] = useState<string | null>(null)

  const { data: policies, isLoading } = useQuery<Policy[]>({
    queryKey: ['policies'],
    queryFn: fetchPolicies,
  })

  const createMutation = useMutation({
    mutationFn: (text: string) => createPolicy(text),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] })
      setNewPolicyText('')
      setAddError(null)
    },
    onError: (err: Error) => {
      setAddError(err.message)
    },
  })

  const toggleMutation = useMutation({
    mutationFn: ({ id, active }: { id: string; active: boolean }) => togglePolicy(id, active),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      const BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const token = typeof window !== 'undefined' ? localStorage.getItem('apex_token') : null
      const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {}
      const res = await fetch(`${BASE_URL}/policies/${id}`, { method: 'DELETE', headers })
      if (!res.ok) throw new Error('Failed to delete policy')
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] })
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!newPolicyText.trim()) return
    createMutation.mutate(newPolicyText.trim())
  }

  const policyList = Array.isArray(policies) ? policies : []

  return (
    <div className="space-y-6">
      {/* Add policy form */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Add Policy</CardTitle>
          <CardDescription>Write a plain-English rule. The agent will compile it automatically.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-3">
            <textarea
              className="w-full min-h-[80px] rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground outline-none resize-y focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/50 placeholder:text-muted-foreground"
              placeholder="e.g. Flag any invoice from a new vendor with amount > $10,000"
              value={newPolicyText}
              onChange={(e) => setNewPolicyText(e.target.value)}
            />
            {addError && <p className="text-xs text-destructive">{addError}</p>}
            <Button
              type="submit"
              size="sm"
              disabled={createMutation.isPending || !newPolicyText.trim()}
            >
              {createMutation.isPending ? 'Adding…' : 'Add Policy'}
            </Button>
          </form>
        </CardContent>
      </Card>

      {/* Policy list */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Active Policies</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-16 w-full" />
              ))}
            </div>
          ) : policyList.length === 0 ? (
            <p className="text-sm text-muted-foreground py-4 text-center">No policies configured.</p>
          ) : (
            <div className="space-y-3">
              {policyList.map((policy) => (
                <div
                  key={policy.id}
                  className="rounded-lg border border-border bg-muted/20 p-3 space-y-2"
                >
                  <div className="flex items-start justify-between gap-3">
                    <p className="text-xs font-medium flex-1">{policy.raw_text}</p>
                    <div className="flex items-center gap-1.5 shrink-0">
                      <Button
                        size="sm"
                        variant={policy.active ? 'default' : 'outline'}
                        onClick={() =>
                          toggleMutation.mutate({ id: policy.id, active: !policy.active })
                        }
                        disabled={toggleMutation.isPending}
                      >
                        {policy.active ? 'Enabled' : 'Disabled'}
                      </Button>
                      <Button
                        size="sm"
                        variant="destructive"
                        onClick={() => deleteMutation.mutate(policy.id)}
                        disabled={deleteMutation.isPending}
                      >
                        Delete
                      </Button>
                    </div>
                  </div>
                  {policy.compiled_rule && (
                    <p className="text-[10px] text-muted-foreground font-mono bg-muted rounded px-2 py-1">
                      {policy.compiled_rule}
                    </p>
                  )}
                  {policy.last_triggered && (
                    <p className="text-[10px] text-muted-foreground">
                      Last triggered: {new Date(policy.last_triggered).toLocaleString()}
                    </p>
                  )}
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

// ---- Team Tab ----
const ROLE_OPTIONS = ['admin', 'reviewer', 'viewer']

function roleBadgeVariant(role: string): 'default' | 'secondary' | 'outline' {
  if (role === 'admin') return 'default'
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
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
  })

  const userList = Array.isArray(users) ? users : []

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Team Members</CardTitle>
        <CardDescription>Manage user roles and access</CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        ) : userList.length === 0 ? (
          <p className="text-sm text-muted-foreground py-4 text-center">No team members found.</p>
        ) : (
          <div className="divide-y divide-border">
            {userList.map((user) => (
              <div key={user.id} className="flex items-center justify-between py-3 gap-3">
                <div className="flex flex-col min-w-0">
                  <span className="text-sm font-medium truncate">{user.name}</span>
                  <span className="text-xs text-muted-foreground truncate">{user.email}</span>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <Badge variant={roleBadgeVariant(user.role)} className="text-[10px]">
                    {user.role}
                  </Badge>
                  <Select
                    value={user.role}
                    onChange={(e) =>
                      updateRoleMutation.mutate({ userId: user.id, role: e.target.value })
                    }
                    className="w-28"
                    disabled={updateRoleMutation.isPending}
                  >
                    {ROLE_OPTIONS.map((r) => (
                      <option key={r} value={r}>
                        {r}
                      </option>
                    ))}
                  </Select>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// ---- Main Settings Page ----
const TABS: { id: Tab; label: string }[] = [
  { id: 'integrations', label: 'Integrations' },
  { id: 'policies', label: 'Policies' },
  { id: 'team', label: 'Team' },
]

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState<Tab>('integrations')

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Settings</h1>
        <p className="text-sm text-muted-foreground">Manage integrations, policies, and team</p>
      </div>

      {/* Tab bar */}
      <div className="flex border-b border-border">
        {TABS.map((tab) => (
          <button
            key={tab.id}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === tab.id
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div>
        {activeTab === 'integrations' && <IntegrationsTab />}
        {activeTab === 'policies' && <PoliciesTab />}
        {activeTab === 'team' && <TeamTab />}
      </div>
    </div>
  )
}
