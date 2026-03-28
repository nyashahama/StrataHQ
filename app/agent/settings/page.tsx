'use client'
import { useState } from 'react'
import { useMockAuth } from '@/lib/mock-auth'
import { useToast } from '@/lib/toast'
import Modal from '@/components/Modal'

export default function AgentSettingsPage() {
  const { user } = useMockAuth()
  const { addToast } = useToast()
  const [showPasswordModal, setShowPasswordModal] = useState(false)
  const [pwForm, setPwForm] = useState({ current: '', next: '', confirm: '' })

  function handlePasswordSave() {
    if (!pwForm.next || pwForm.next !== pwForm.confirm) return
    setShowPasswordModal(false)
    setPwForm({ current: '', next: '', confirm: '' })
    addToast('Password updated', 'success')
  }

  return (
    <div className="px-8 py-8 max-w-[700px]">
      <p className="text-[12px] text-muted mb-4">Settings</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Settings</h1>
      <p className="text-[14px] text-muted mb-8">Organisation and account settings.</p>

      {/* Organisation */}
      <div className="bg-surface border border-border rounded-lg overflow-hidden mb-6">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Organisation</span>
        </div>
        <div className="px-5 py-4 flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Organisation name</label>
            <input
              type="text"
              defaultValue={user?.orgName}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Contact email</label>
            <input
              type="email"
              defaultValue="admin@acme.co.za"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Phone</label>
            <input
              type="tel"
              defaultValue="+27 21 555 0100"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <button
              onClick={() => addToast('Settings saved', 'success')}
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
        <div className="px-5 py-4 flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Password</label>
            <button
              onClick={() => setShowPasswordModal(true)}
              className="text-[12px] text-accent font-medium hover:underline"
            >
              Change password →
            </button>
          </div>
          <div className="pt-2 border-t border-border">
            <p className="text-[12px] text-muted mb-2">Danger zone</p>
            <button className="text-[12px] font-medium text-red hover:underline">
              Delete account
            </button>
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
