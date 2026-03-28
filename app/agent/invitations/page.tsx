'use client'
import { useState } from 'react'
import { useMockAuth } from '@/lib/mock-auth'
import { useToast } from '@/lib/toast'

interface Invitation {
  id: string
  name: string
  email: string
  role: 'trustee' | 'resident'
  scheme_name: string
  unit_identifier: string
  invited_at: string
}

const mockInvitations: Invitation[] = [
  { id: 'inv-001', name: 'Nkosi, A.',      email: 'a.nkosi@gmail.com',        role: 'resident', scheme_name: 'Sunridge Heights', unit_identifier: '9A',  invited_at: '2025-10-14T09:00:00Z' },
  { id: 'inv-002', name: 'Botha, C.',      email: 'c.botha@outlook.com',       role: 'trustee',  scheme_name: 'Sunridge Heights', unit_identifier: '10B', invited_at: '2025-10-13T11:30:00Z' },
  { id: 'inv-003', name: 'Fredericks, P.', email: 'p.fredericks@email.co.za',  role: 'resident', scheme_name: 'Sunridge Heights', unit_identifier: '11C', invited_at: '2025-10-12T14:00:00Z' },
]

const ROLE_STYLES: Record<string, string> = {
  trustee:  'bg-accent-bg text-accent',
  resident: 'bg-green-bg text-green',
}

export default function InvitationsPage() {
  useMockAuth()
  const { addToast } = useToast()
  const [invitations, setInvitations] = useState<Invitation[]>(mockInvitations)

  function handleAction(id: string, action: 'resend' | 'revoke') {
    if (action === 'revoke') {
      setInvitations(prev => prev.filter(i => i.id !== id))
      addToast('Invitation revoked', 'info')
    } else {
      addToast('Invitation resent', 'success')
    }
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Portfolio › Invitations</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Invitations</h1>
      <p className="text-[14px] text-muted mb-8">Pending trustee and resident invitations.</p>

      {invitations.length === 0 ? (
        <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          No pending invitations
        </div>
      ) : (
        <div className="bg-surface border border-border rounded-lg overflow-hidden">
          <div className="px-5 py-3 border-b border-border flex items-center justify-between">
            <span className="text-[13px] font-semibold text-ink">Pending</span>
            <span className="text-[11px] font-semibold px-2 py-[2px] rounded-full bg-yellowbg text-amber">{invitations.length} pending</span>
          </div>
          <div className="overflow-x-auto -mx-5">
            <div className="px-5 min-w-[480px]">
            <div className="grid grid-cols-[1fr_auto_auto_auto] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
              <span>Invitee</span><span>Unit</span><span>Role</span><span>Actions</span>
            </div>
            {invitations.map((inv, i) => (
              <div key={inv.id} className={`grid grid-cols-[1fr_auto_auto_auto] gap-4 items-center py-3 text-[13px] ${i < invitations.length - 1 ? 'border-b border-border' : ''}`}>
                <div>
                  <div className="font-medium text-ink">{inv.name}</div>
                  <div className="text-[12px] text-muted">{inv.email}</div>
                </div>
                <span className="text-muted text-[12px]">{inv.unit_identifier}</span>
                <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${ROLE_STYLES[inv.role]}`}>
                  {inv.role.charAt(0).toUpperCase() + inv.role.slice(1)}
                </span>
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => handleAction(inv.id, 'resend')}
                    className="text-[11px] text-accent font-medium hover:underline"
                  >
                    Resend
                  </button>
                  <button
                    onClick={() => handleAction(inv.id, 'revoke')}
                    className="text-[11px] text-red font-medium hover:underline"
                  >
                    Revoke
                  </button>
                </div>
              </div>
            ))}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
