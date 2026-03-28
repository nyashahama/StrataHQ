'use client'
import { useState } from 'react'
import { useMockAuth } from '@/lib/mock-auth'
import { mockMembers } from '@/lib/mock/members'
import { useToast } from '@/lib/toast'
import Modal from '@/components/Modal'

export default function ResidentProfilePage() {
  const { user } = useMockAuth()
  const { addToast } = useToast()
  const [showPasswordModal, setShowPasswordModal] = useState(false)
  const [pwForm, setPwForm] = useState({ current: '', next: '', confirm: '' })

  const myMember = mockMembers.find(m => m.unit_identifier === user?.unitIdentifier)

  function handlePasswordSave() {
    if (!pwForm.next || pwForm.next !== pwForm.confirm) return
    setShowPasswordModal(false)
    setPwForm({ current: '', next: '', confirm: '' })
    addToast('Password updated', 'success')
  }

  return (
    <div className="px-8 py-8 max-w-[700px]">
      <p className="text-[12px] text-muted mb-4">My Profile</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">My Profile</h1>
      <p className="text-[14px] text-muted mb-8">Your contact details and unit information.</p>

      {/* Unit info */}
      <div className="bg-surface border border-border rounded-lg px-5 py-4 mb-6 flex items-center gap-4">
        <div className="w-12 h-12 rounded-lg bg-accent-bg flex items-center justify-center flex-shrink-0">
          <span className="text-[16px] font-semibold text-accent">{user?.unitIdentifier}</span>
        </div>
        <div>
          <div className="text-[14px] font-semibold text-ink">Unit {user?.unitIdentifier}</div>
          <div className="text-[12px] text-muted">{user?.schemeName}</div>
        </div>
        <span className="ml-auto text-[11px] font-semibold px-2 py-[2px] rounded-full bg-green-bg text-green">
          Owner
        </span>
      </div>

      {/* Contact details */}
      <div className="bg-surface border border-border rounded-lg overflow-hidden mb-6">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Contact details</span>
        </div>
        <div className="px-5 py-4 flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Full name</label>
            <input
              type="text"
              defaultValue={myMember?.name ?? ''}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Email</label>
            <input
              type="email"
              defaultValue={myMember?.email ?? ''}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Phone</label>
            <input
              type="tel"
              defaultValue={myMember?.phone ?? ''}
              placeholder="+27 82 000 0000"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <button
              onClick={() => addToast('Profile updated', 'success')}
              className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors"
            >
              Save changes
            </button>
          </div>
        </div>
      </div>

      {/* Account */}
      <div className="bg-surface border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Account</span>
        </div>
        <div className="px-5 py-4 flex flex-col gap-3">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Password</label>
            <button
              onClick={() => setShowPasswordModal(true)}
              className="text-[12px] text-accent font-medium hover:underline"
            >
              Change password →
            </button>
          </div>
          <div className="text-[12px] text-muted pt-2 border-t border-border">
            Account managed by your body corporate managing agent.
          </div>
        </div>
      </div>

      <Modal open={showPasswordModal} onClose={() => setShowPasswordModal(false)} title="Change password">
        <div className="flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Current password</label>
            <input type="password" value={pwForm.current} onChange={e => setPwForm(f => ({ ...f, current: e.target.value }))} placeholder="••••••••" className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent" />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">New password</label>
            <input type="password" value={pwForm.next} onChange={e => setPwForm(f => ({ ...f, next: e.target.value }))} placeholder="••••••••" className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent" />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Confirm new password</label>
            <input type="password" value={pwForm.confirm} onChange={e => setPwForm(f => ({ ...f, confirm: e.target.value }))} placeholder="••••••••" className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent" />
            {pwForm.confirm && pwForm.next !== pwForm.confirm && (
              <p className="text-[11px] text-red mt-1">Passwords do not match</p>
            )}
          </div>
          <div className="flex justify-end gap-3 pt-1">
            <button onClick={() => setShowPasswordModal(false)} className="text-[12px] text-muted hover:text-ink px-3 py-2">Cancel</button>
            <button onClick={handlePasswordSave} disabled={!pwForm.current || !pwForm.next || pwForm.next !== pwForm.confirm} className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed">Update password</button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
