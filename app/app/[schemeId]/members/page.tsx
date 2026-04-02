'use client'

import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'next/navigation'

import Modal from '@/components/Modal'
import { apiFetch } from '@/lib/api'
import { readApiError } from '@/lib/api-contract'
import { useAuth } from '@/lib/auth'
import {
  listSchemeMembers,
  listSchemeUnits,
  updateSchemeMember,
  type MemberInfo,
  type UnitInfo,
} from '@/lib/scheme-api'
import { useToast } from '@/lib/toast'

const ROLE_STYLES: Record<string, string> = {
  trustee: 'bg-accent-bg text-accent',
  resident: 'bg-green-bg text-green',
}

type InviteRole = 'trustee' | 'resident'

interface InviteFormState {
  full_name: string
  email: string
  role: InviteRole
  unit_id: string
}

interface EditFormState {
  role: InviteRole
  unit_id: string
}

const EMPTY_INVITE_FORM: InviteFormState = {
  full_name: '',
  email: '',
  role: 'resident',
  unit_id: '',
}

const EMPTY_EDIT_FORM: EditFormState = {
  role: 'resident',
  unit_id: '',
}

export default function MembersPage() {
  const { user } = useAuth()
  const { addToast } = useToast()
  const params = useParams()
  const schemeId = params.schemeId as string

  const [members, setMembers] = useState<MemberInfo[]>([])
  const [units, setUnits] = useState<UnitInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [showInviteModal, setShowInviteModal] = useState(false)
  const [showEditModal, setShowEditModal] = useState(false)
  const [inviteForm, setInviteForm] = useState<InviteFormState>(EMPTY_INVITE_FORM)
  const [editForm, setEditForm] = useState<EditFormState>(EMPTY_EDIT_FORM)
  const [selectedMember, setSelectedMember] = useState<MemberInfo | null>(null)
  const [savingInvite, setSavingInvite] = useState(false)
  const [savingMember, setSavingMember] = useState(false)

  const canEdit = user?.role === 'admin'

  useEffect(() => {
    async function load() {
      try {
        const [loadedMembers, loadedUnits] = await Promise.all([
          listSchemeMembers(schemeId),
          listSchemeUnits(schemeId),
        ])
        setMembers(loadedMembers)
        setUnits(loadedUnits)
      } catch (error) {
        addToast(
          error instanceof Error ? error.message : 'Failed to load members',
          'error',
        )
      } finally {
        setLoading(false)
      }
    }

    load()
  }, [addToast, schemeId])

  const trustees = useMemo(
    () => members.filter(member => member.role === 'trustee'),
    [members],
  )
  const residents = useMemo(
    () => members.filter(member => member.role === 'resident'),
    [members],
  )

  const filteredMembers = useMemo(() => {
    const query = search.trim().toLowerCase()
    if (!query) return members
    return members.filter(member =>
      member.full_name.toLowerCase().includes(query) ||
      member.email.toLowerCase().includes(query) ||
      (member.unit_identifier ?? '').toLowerCase().includes(query),
    )
  }, [members, search])

  function openEdit(member: MemberInfo) {
    setSelectedMember(member)
    setEditForm({
      role: member.role,
      unit_id: member.unit_id ?? '',
    })
    setShowEditModal(true)
  }

  async function handleInvite() {
    if (!inviteForm.full_name.trim() || !inviteForm.email.trim()) return
    if (inviteForm.role === 'resident' && !inviteForm.unit_id) {
      addToast('Residents require a unit assignment', 'error')
      return
    }

    setSavingInvite(true)
    try {
      const response = await apiFetch('/api/v1/invitations', {
        method: 'POST',
        body: JSON.stringify({
          full_name: inviteForm.full_name.trim(),
          email: inviteForm.email.trim(),
          role: inviteForm.role,
          scheme_id: schemeId,
          unit_id: inviteForm.role === 'resident' ? inviteForm.unit_id : '',
        }),
      })

      if (!response.ok) {
        throw new Error(await readApiError(response, 'Failed to send invitation'))
      }

      setShowInviteModal(false)
      setInviteForm(EMPTY_INVITE_FORM)
      addToast(`Invite sent to ${inviteForm.email.trim()}`, 'success')
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to send invitation',
        'error',
      )
    } finally {
      setSavingInvite(false)
    }
  }

  async function handleMemberSave() {
    if (!selectedMember) return
    if (editForm.role === 'resident' && !editForm.unit_id) {
      addToast('Residents require a unit assignment', 'error')
      return
    }

    setSavingMember(true)
    try {
      const updated = await updateSchemeMember(schemeId, selectedMember.user_id, {
        role: editForm.role,
        unit_id: editForm.role === 'resident' ? editForm.unit_id : null,
      })
      setMembers(current =>
        current.map(member => member.user_id === updated.user_id ? updated : member),
      )
      setShowEditModal(false)
      setSelectedMember(null)
      setEditForm(EMPTY_EDIT_FORM)
      addToast('Member updated', 'success')
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to update member',
        'error',
      )
    } finally {
      setSavingMember(false)
    }
  }

  if (loading) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Loading members…
        </div>
      </div>
    )
  }

  if (user?.role === 'resident') {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › Members</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Trustee Committee</h1>
        <p className="text-[14px] text-muted mb-8">Contact the trustees for scheme-related matters.</p>
        <div className="flex flex-col gap-3">
          {trustees.map(member => (
            <div key={member.user_id} className="bg-surface border border-border rounded-lg px-5 py-4 flex items-center justify-between gap-3">
              <div>
                <div className="text-[14px] font-semibold text-ink">{member.full_name}</div>
                <div className="text-[12px] text-muted mt-0.5">
                  {member.unit_identifier ? `Unit ${member.unit_identifier} · ` : ''}{member.email}
                </div>
              </div>
              {member.phone && <span className="text-[12px] text-muted">{member.phone}</span>}
            </div>
          ))}
          {trustees.length === 0 && (
            <div className="bg-hover-subtle border border-border rounded-lg px-6 py-12 text-center text-[14px] text-muted">
              No trustee committee members are linked yet.
            </div>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Members</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Members</h1>
      <p className="text-[14px] text-muted mb-8">Trustees, residents, and their current unit assignments.</p>

      <div className="grid grid-cols-3 gap-3 sm:gap-4 mb-8">
        {[
          { label: 'Total members', value: String(members.length) },
          { label: 'Trustees', value: String(trustees.length) },
          { label: 'Residents', value: String(residents.length) },
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
              onClick={() => setShowInviteModal(true)}
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
              placeholder="Search by name, email, or unit…"
              value={search}
              onChange={e => setSearch(e.target.value)}
              className="w-full sm:max-w-xs border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div className="overflow-x-auto">
            <div className="min-w-[620px]">
              <div className="grid grid-cols-[90px_1fr_160px_auto_72px] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
                <span>Unit</span><span>Name</span><span>Contact</span><span>Role</span><span></span>
              </div>
              {filteredMembers.length === 0 ? (
                <div className="text-[13px] text-muted text-center py-8">No members match your search.</div>
              ) : (
                filteredMembers.map((member, index) => (
                  <div key={member.user_id} className={`grid grid-cols-[90px_1fr_160px_auto_72px] gap-4 items-center py-3 text-[13px] ${index < filteredMembers.length - 1 ? 'border-b border-border' : ''}`}>
                    <span className="font-semibold text-ink">{member.unit_identifier ?? '—'}</span>
                    <div>
                      <span className="text-ink">{member.full_name}</span>
                      <div className="text-[12px] text-muted mt-0.5">{member.email}</div>
                    </div>
                    <span className="text-[12px] text-muted">{member.phone ?? '—'}</span>
                    <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${ROLE_STYLES[member.role]}`}>
                      {member.role.charAt(0).toUpperCase() + member.role.slice(1)}
                    </span>
                    <div className="text-right">
                      {canEdit ? (
                        <button onClick={() => openEdit(member)} className="text-[12px] text-accent font-medium hover:underline">
                          Edit
                        </button>
                      ) : (
                        <span className="text-[12px] text-muted">Read only</span>
                      )}
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      </div>

      <Modal open={showInviteModal} onClose={() => setShowInviteModal(false)} title="Invite member">
        <div className="flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Full name *</label>
            <input
              type="text"
              value={inviteForm.full_name}
              onChange={e => setInviteForm(current => ({ ...current, full_name: e.target.value }))}
              placeholder="e.g. Nkosi, A."
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Email *</label>
            <input
              type="email"
              value={inviteForm.email}
              onChange={e => setInviteForm(current => ({ ...current, email: e.target.value }))}
              placeholder="e.g. nkosi@email.co.za"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Role</label>
              <select
                value={inviteForm.role}
                onChange={e => setInviteForm(current => ({ ...current, role: e.target.value as InviteRole, unit_id: e.target.value === 'trustee' ? '' : current.unit_id }))}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
              >
                <option value="resident">Resident</option>
                <option value="trustee">Trustee</option>
              </select>
            </div>
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Unit</label>
              <select
                value={inviteForm.unit_id}
                onChange={e => setInviteForm(current => ({ ...current, unit_id: e.target.value }))}
                disabled={inviteForm.role !== 'resident'}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent disabled:bg-page disabled:text-muted"
              >
                <option value="">Select unit</option>
                {units.map(unit => (
                  <option key={unit.id} value={unit.id}>
                    {unit.identifier} · {unit.owner_name}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <div className="flex gap-2 pt-1">
            <button
              onClick={handleInvite}
              disabled={savingInvite || !inviteForm.full_name.trim() || !inviteForm.email.trim()}
              className="flex-1 bg-accent text-white text-[13px] font-semibold py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {savingInvite ? 'Sending…' : 'Send invite'}
            </button>
            <button
              onClick={() => setShowInviteModal(false)}
              className="px-4 text-[13px] font-medium text-muted hover:text-ink border border-border rounded py-2 transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      </Modal>

      <Modal open={showEditModal} onClose={() => setShowEditModal(false)} title="Edit member">
        <div className="flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Role</label>
            <select
              value={editForm.role}
              onChange={e => setEditForm(current => ({ ...current, role: e.target.value as InviteRole, unit_id: e.target.value === 'trustee' ? '' : current.unit_id }))}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            >
              <option value="resident">Resident</option>
              <option value="trustee">Trustee</option>
            </select>
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Unit</label>
            <select
              value={editForm.unit_id}
              onChange={e => setEditForm(current => ({ ...current, unit_id: e.target.value }))}
              disabled={editForm.role !== 'resident'}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent disabled:bg-page disabled:text-muted"
            >
              <option value="">Select unit</option>
              {units.map(unit => (
                <option key={unit.id} value={unit.id}>
                  {unit.identifier} · {unit.owner_name}
                </option>
              ))}
            </select>
          </div>
          <div className="flex gap-2 pt-1">
            <button
              onClick={handleMemberSave}
              disabled={savingMember}
              className="flex-1 bg-accent text-white text-[13px] font-semibold py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {savingMember ? 'Saving…' : 'Save changes'}
            </button>
            <button
              onClick={() => setShowEditModal(false)}
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
