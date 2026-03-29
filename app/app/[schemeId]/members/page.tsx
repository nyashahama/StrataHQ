'use client'
import { useState } from 'react'
import { useAuth } from '@/lib/auth'
import { mockMembers, type Member } from '@/lib/mock/members'
import { useToast } from '@/lib/toast'
import Modal from '@/components/Modal'

const ROLE_STYLES: Record<string, string> = {
  trustee:  'bg-accent-bg text-accent',
  owner:    'bg-[#f0efe9] text-muted',
  resident: 'bg-green-bg text-green',
}

export default function MembersPage() {
  const { user } = useAuth()
  const { addToast } = useToast()

  const [members, setMembers] = useState<Member[]>([...mockMembers])
  const [search, setSearch] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [form, setForm] = useState({ name: '', email: '', unit: '', role: 'owner' as 'owner' | 'resident' })

  function handleInvite() {
    if (!form.name.trim() || !form.email.trim() || !form.unit.trim()) return
    const newMember: Member = {
      id: `member-${Date.now()}`,
      scheme_id: 'scheme-001',
      unit_id: `unit-${form.unit.toLowerCase()}`,
      unit_identifier: form.unit.toUpperCase(),
      name: form.name.trim(),
      role: form.role,
      email: form.email.trim(),
      phone: null,
      is_trustee_committee: false,
    }
    setMembers(prev => [...prev, newMember])
    setShowModal(false)
    setForm({ name: '', email: '', unit: '', role: 'owner' })
    addToast(`Invite sent to ${form.email.trim()}`, 'success')
  }

  if (user?.role === 'resident') {
    const trustees = members.filter(m => m.is_trustee_committee)
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › Members</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Trustee Committee</h1>
        <p className="text-[14px] text-muted mb-8">Contact the trustees for scheme-related matters.</p>
        <div className="flex flex-col gap-3">
          {trustees.map(m => (
            <div key={m.id} className="bg-surface border border-border rounded-lg px-5 py-4 flex items-center justify-between">
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

  const canEdit = user?.role === 'admin'
  const trustees = members.filter(m => m.is_trustee_committee)
  const owners = members.filter(m => !m.is_trustee_committee)

  const filteredMembers = members.filter(m =>
    search === '' ||
    m.name.toLowerCase().includes(search.toLowerCase()) ||
    m.unit_identifier.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Members</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Members</h1>
      <p className="text-[14px] text-muted mb-8">Owners, trustees, and contact information.</p>

      <div className="grid grid-cols-3 gap-3 sm:gap-4 mb-8">
        {[
          { label: 'Total members', value: String(members.length) },
          { label: 'Trustees',      value: String(trustees.length) },
          { label: 'Owners',        value: String(owners.length) },
        ].map(({ label, value }) => (
          <div key={label} className="bg-surface border border-border rounded-lg px-5 py-4">
            <div className="text-[24px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      <div className="bg-surface border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border flex flex-wrap items-center justify-between gap-3">
          <span className="text-[13px] font-semibold text-ink">All members</span>
          {canEdit && (
            <button
              onClick={() => setShowModal(true)}
              className="text-[12px] font-semibold bg-accent text-white px-3 py-1.5 rounded hover:opacity-90 transition-colors"
            >
              + Invite member
            </button>
          )}
        </div>
        <div className="px-5">
          <div className="mb-4 pt-4">
            <input
              type="text"
              placeholder="Search by name or unit…"
              value={search}
              onChange={e => setSearch(e.target.value)}
              className="w-full sm:max-w-xs border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div className="overflow-x-auto">
            <div className="min-w-[480px]">
              <div className="grid grid-cols-[auto_1fr_auto_auto] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
                <span>Unit</span><span>Name</span><span>Contact</span><span>Role</span>
              </div>
              {filteredMembers.length === 0 ? (
                <div className="text-[13px] text-muted text-center py-8">No members match your search.</div>
              ) : (
                filteredMembers.map((m, i) => (
                  <div key={m.id} className={`grid grid-cols-[auto_1fr_auto_auto] gap-4 items-center py-3 text-[13px] ${i < filteredMembers.length - 1 ? 'border-b border-border' : ''}`}>
                    <span className="font-semibold text-ink w-8">{m.unit_identifier}</span>
                    <span className="text-ink">{m.name}</span>
                    <span className="text-[12px] text-muted">{m.phone ?? '—'}</span>
                    <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${ROLE_STYLES[m.role]}`}>
                      {m.is_trustee_committee ? 'Trustee' : m.role.charAt(0).toUpperCase() + m.role.slice(1)}
                    </span>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      </div>

      <Modal open={showModal} onClose={() => setShowModal(false)} title="Invite member">
        <div className="flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Full name *</label>
            <input
              type="text"
              value={form.name}
              onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
              placeholder="e.g. Nkosi, A."
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Email *</label>
            <input
              type="email"
              value={form.email}
              onChange={e => setForm(f => ({ ...f, email: e.target.value }))}
              placeholder="e.g. nkosi@email.co.za"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Unit *</label>
              <input
                type="text"
                value={form.unit}
                onChange={e => setForm(f => ({ ...f, unit: e.target.value }))}
                placeholder="e.g. 9A"
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
              />
            </div>
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Role</label>
              <select
                value={form.role}
                onChange={e => setForm(f => ({ ...f, role: e.target.value as 'owner' | 'resident' }))}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
              >
                <option value="owner">Owner</option>
                <option value="resident">Resident</option>
              </select>
            </div>
          </div>
          <div className="flex gap-2 pt-1">
            <button
              onClick={handleInvite}
              disabled={!form.name.trim() || !form.email.trim() || !form.unit.trim()}
              className="flex-1 bg-accent text-white text-[13px] font-semibold py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Send invite
            </button>
            <button
              onClick={() => setShowModal(false)}
              className="px-4 text-[13px] font-medium text-muted hover:text-ink border border-border rounded py-2 transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
