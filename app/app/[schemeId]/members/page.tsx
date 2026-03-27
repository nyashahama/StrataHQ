'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { mockMembers } from '@/lib/mock/members'

const ROLE_STYLES: Record<string, string> = {
  trustee:  'bg-accent-bg text-accent',
  owner:    'bg-[#f0efe9] text-muted',
  resident: 'bg-green-bg text-green',
}

export default function MembersPage() {
  const { user } = useMockAuth()

  // Resident: trustees only
  if (user?.role === 'resident') {
    const trustees = mockMembers.filter(m => m.is_trustee_committee)
    return (
      <div className="px-8 py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › Members</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Trustee Committee</h1>
        <p className="text-[14px] text-muted mb-8">Contact the trustees for scheme-related matters.</p>
        <div className="flex flex-col gap-3">
          {trustees.map(m => (
            <div key={m.id} className="bg-white border border-border rounded-lg px-5 py-4 flex items-center justify-between">
              <div>
                <div className="text-[14px] font-semibold text-ink">{m.name}</div>
                <div className="text-[12px] text-muted mt-0.5">Unit {m.unit_identifier} · {m.email}</div>
              </div>
              {m.phone && <span className="text-[12px] text-muted">{m.phone}</span>}
            </div>
          ))}
        </div>
      </div>
    )
  }

  // Agent / Trustee: full roster
  const canEdit = user?.role === 'agent'
  const trustees = mockMembers.filter(m => m.is_trustee_committee)
  const owners = mockMembers.filter(m => !m.is_trustee_committee)

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Members</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Members</h1>
      <p className="text-[14px] text-muted mb-8">Owners, trustees, and contact information.</p>

      {/* Stat cards */}
      <div className="grid grid-cols-3 gap-4 mb-8">
        {[
          { label: 'Total members', value: String(mockMembers.length) },
          { label: 'Trustees', value: String(trustees.length) },
          { label: 'Owners', value: String(owners.length) },
        ].map(({ label, value }) => (
          <div key={label} className="bg-white border border-border rounded-lg px-5 py-4">
            <div className="text-[24px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      {/* Members table */}
      <div className="bg-white border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border flex items-center justify-between">
          <span className="text-[13px] font-semibold text-ink">All members</span>
          {canEdit && (
            <button className="text-[12px] font-semibold bg-accent text-white px-3 py-1.5 rounded hover:bg-[#245a96] transition-colors">
              + Invite member
            </button>
          )}
        </div>
        <div className="px-5">
          <div className="grid grid-cols-[auto_1fr_auto_auto] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
            <span>Unit</span><span>Name</span><span>Contact</span><span>Role</span>
          </div>
          {mockMembers.map((m, i) => (
            <div key={m.id} className={`grid grid-cols-[auto_1fr_auto_auto] gap-4 items-center py-3 text-[13px] ${i < mockMembers.length - 1 ? 'border-b border-border' : ''}`}>
              <span className="font-semibold text-ink w-8">{m.unit_identifier}</span>
              <span className="text-ink">{m.name}</span>
              <span className="text-[12px] text-muted">{m.phone ?? '—'}</span>
              <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${ROLE_STYLES[m.role]}`}>
                {m.is_trustee_committee ? 'Trustee' : m.role.charAt(0).toUpperCase() + m.role.slice(1)}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
