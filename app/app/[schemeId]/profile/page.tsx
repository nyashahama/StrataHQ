'use client'

import { useEffect, useState } from 'react'

import Modal from '@/components/Modal'
import { changePassword, updateProfile } from '@/lib/account-api'
import { setSessionCookie, useAuth } from '@/lib/auth'
import { useToast } from '@/lib/toast'

export default function ResidentProfilePage() {
  const { user, setUser } = useAuth()
  const { addToast } = useToast()
  const [profileForm, setProfileForm] = useState({
    full_name: '',
    email: '',
    phone: '',
  })
  const [showPasswordModal, setShowPasswordModal] = useState(false)
  const [pwForm, setPwForm] = useState({ current: '', next: '', confirm: '' })
  const [savingProfile, setSavingProfile] = useState(false)
  const [savingPassword, setSavingPassword] = useState(false)

  useEffect(() => {
    setProfileForm({
      full_name: user?.full_name ?? '',
      email: user?.email ?? '',
      phone: user?.phone ?? '',
    })
  }, [user])

  const primaryMembership = user?.scheme_memberships?.[0] ?? null

  async function handleProfileSave() {
    if (!profileForm.full_name.trim() || !profileForm.email.trim()) {
      addToast('Full name and email are required', 'error')
      return
    }

    setSavingProfile(true)
    try {
      const updated = await updateProfile({
        full_name: profileForm.full_name.trim(),
        email: profileForm.email.trim(),
        phone: profileForm.phone.trim(),
      })
      setUser(updated)
      setSessionCookie(updated)
      setProfileForm({
        full_name: updated.full_name,
        email: updated.email,
        phone: updated.phone ?? '',
      })
      addToast('Profile updated', 'success')
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to update profile',
        'error',
      )
    } finally {
      setSavingProfile(false)
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
      <p className="text-[12px] text-muted mb-4">My Profile</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">My Profile</h1>
      <p className="text-[14px] text-muted mb-8">Your contact details and unit information.</p>

      <div className="bg-surface border border-border rounded-lg px-5 py-4 mb-6 flex items-center gap-4">
        <div className="w-12 h-12 rounded-lg bg-accent-bg flex items-center justify-center flex-shrink-0">
          <span className="text-[16px] font-semibold text-accent">
            {primaryMembership?.unit_identifier?.slice(0, 2).toUpperCase() ?? 'ME'}
          </span>
        </div>
        <div>
          <div className="text-[14px] font-semibold text-ink">
            {primaryMembership?.unit_identifier ? `Unit ${primaryMembership.unit_identifier}` : 'Unit assignment pending'}
          </div>
          <div className="text-[12px] text-muted">{primaryMembership?.scheme_name ?? 'No scheme linked yet'}</div>
        </div>
        <span className="ml-auto flex-shrink-0 text-[11px] font-semibold px-2 py-[2px] rounded-full bg-green-bg text-green">
          {primaryMembership?.role ?? 'resident'}
        </span>
      </div>

      <div className="bg-surface border border-border rounded-lg overflow-hidden mb-6">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Contact details</span>
        </div>
        <div className="px-5 py-4 flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Full name</label>
            <input
              type="text"
              value={profileForm.full_name}
              onChange={e => setProfileForm(current => ({ ...current, full_name: e.target.value }))}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Email</label>
            <input
              type="email"
              value={profileForm.email}
              onChange={e => setProfileForm(current => ({ ...current, email: e.target.value }))}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Phone</label>
            <input
              type="tel"
              value={profileForm.phone}
              onChange={e => setProfileForm(current => ({ ...current, phone: e.target.value }))}
              placeholder="+27 82 000 0000"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <button
              onClick={handleProfileSave}
              disabled={savingProfile}
              className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {savingProfile ? 'Saving…' : 'Save changes'}
            </button>
          </div>
        </div>
      </div>

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
            <button onClick={handlePasswordSave} disabled={savingPassword || !pwForm.current || !pwForm.next || pwForm.next !== pwForm.confirm} className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed">{savingPassword ? 'Updating…' : 'Update password'}</button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
