'use client'

import { useEffect, useState } from 'react'

import Modal from '@/components/Modal'
import { changePassword, updateOrgSettings } from '@/lib/account-api'
import { setSessionCookie, useAuth } from '@/lib/auth'
import { useToast } from '@/lib/toast'

export default function AgentSettingsPage() {
  const { user, setUser } = useAuth()
  const { addToast } = useToast()
  const [orgForm, setOrgForm] = useState({
    name: '',
    contact_email: '',
    contact_phone: '',
  })
  const [showPasswordModal, setShowPasswordModal] = useState(false)
  const [pwForm, setPwForm] = useState({ current: '', next: '', confirm: '' })
  const [savingOrg, setSavingOrg] = useState(false)
  const [savingPassword, setSavingPassword] = useState(false)

  useEffect(() => {
    setOrgForm({
      name: user?.org?.name ?? '',
      contact_email: user?.org?.contact_email ?? '',
      contact_phone: user?.org?.contact_phone ?? '',
    })
  }, [user])

  async function handleOrgSave() {
    if (!user) return
    if (!orgForm.name.trim()) {
      addToast('Organisation name is required', 'error')
      return
    }

    setSavingOrg(true)
    try {
      const org = await updateOrgSettings({
        name: orgForm.name.trim(),
        contact_email: orgForm.contact_email.trim(),
        contact_phone: orgForm.contact_phone.trim(),
      })
      const nextUser = { ...user, org }
      setUser(nextUser)
      setSessionCookie(nextUser)
      setOrgForm({
        name: org.name,
        contact_email: org.contact_email ?? '',
        contact_phone: org.contact_phone ?? '',
      })
      addToast('Organisation settings saved', 'success')
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to save settings',
        'error',
      )
    } finally {
      setSavingOrg(false)
    }
  }

  async function handlePasswordSave() {
    if (!pwForm.current || !pwForm.next) return
    if (pwForm.next !== pwForm.confirm) return

    setSavingPassword(true)
    try {
      await changePassword({
        current_password: pwForm.current,
        new_password: pwForm.next,
      })
      setShowPasswordModal(false)
      setPwForm({ current: '', next: '', confirm: '' })
      addToast('Password updated', 'success')
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to update password',
        'error',
      )
    } finally {
      setSavingPassword(false)
    }
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[700px]">
      <p className="text-[12px] text-muted mb-4">Settings</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Settings</h1>
      <p className="text-[14px] text-muted mb-8">Organisation and account settings.</p>

      <div className="bg-surface border border-border rounded-lg overflow-hidden mb-6">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Organisation</span>
        </div>
        <div className="px-5 py-4 flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Organisation name</label>
            <input
              type="text"
              value={orgForm.name}
              onChange={e => setOrgForm(current => ({ ...current, name: e.target.value }))}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Contact email</label>
            <input
              type="email"
              value={orgForm.contact_email}
              onChange={e => setOrgForm(current => ({ ...current, contact_email: e.target.value }))}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Phone</label>
            <input
              type="tel"
              value={orgForm.contact_phone}
              onChange={e => setOrgForm(current => ({ ...current, contact_phone: e.target.value }))}
              placeholder="+27 21 555 0100"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <button
              onClick={handleOrgSave}
              disabled={savingOrg || !user}
              className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {savingOrg ? 'Saving…' : 'Save changes'}
            </button>
          </div>
        </div>
      </div>

      <div className="bg-surface border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Account</span>
        </div>
        <div className="px-5 py-4 flex flex-col gap-4">
          <div>
            <p className="text-[12px] font-semibold text-ink mb-1">Signed in as</p>
            <p className="text-[13px] text-ink">{user?.full_name ?? 'Unknown user'}</p>
            <p className="text-[12px] text-muted">{user?.email ?? 'No email address'}</p>
          </div>
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
            <button onClick={handlePasswordSave} disabled={savingPassword || !pwForm.current || !pwForm.next || pwForm.next !== pwForm.confirm} className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed">{savingPassword ? 'Updating…' : 'Update password'}</button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
