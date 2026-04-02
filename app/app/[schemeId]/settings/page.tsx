'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'

import Modal from '@/components/Modal'
import { changePassword } from '@/lib/account-api'
import {
  createSchemeUnit,
  getScheme,
  updateScheme,
  updateSchemeUnit,
  type SchemeDetail,
  type UnitInfo,
} from '@/lib/scheme-api'
import { useAuth } from '@/lib/auth'
import { useToast } from '@/lib/toast'

interface UnitFormState {
  identifier: string
  owner_name: string
  floor: string
  section_value_pct: string
}

const EMPTY_UNIT_FORM: UnitFormState = {
  identifier: '',
  owner_name: '',
  floor: '0',
  section_value_pct: '',
}

function toUnitForm(unit?: UnitInfo | null): UnitFormState {
  if (!unit) return EMPTY_UNIT_FORM
  return {
    identifier: unit.identifier,
    owner_name: unit.owner_name,
    floor: String(unit.floor),
    section_value_pct: unit.section_value_pct.toFixed(2),
  }
}

export default function SchemeSettingsPage() {
  const { user } = useAuth()
  const { addToast } = useToast()
  const params = useParams()
  const schemeId = params.schemeId as string

  const [scheme, setScheme] = useState<SchemeDetail | null>(null)
  const [schemeForm, setSchemeForm] = useState({
    name: '',
    address: '',
    unit_count: '',
  })
  const [loading, setLoading] = useState(true)
  const [savingScheme, setSavingScheme] = useState(false)
  const [showPasswordModal, setShowPasswordModal] = useState(false)
  const [savingPassword, setSavingPassword] = useState(false)
  const [pwForm, setPwForm] = useState({ current: '', next: '', confirm: '' })
  const [showUnitModal, setShowUnitModal] = useState(false)
  const [savingUnit, setSavingUnit] = useState(false)
  const [editingUnit, setEditingUnit] = useState<UnitInfo | null>(null)
  const [unitForm, setUnitForm] = useState<UnitFormState>(EMPTY_UNIT_FORM)

  const canEdit = user?.role === 'admin'

  useEffect(() => {
    async function load() {
      try {
        const detail = await getScheme(schemeId)
        setScheme(detail)
        setSchemeForm({
          name: detail.name,
          address: detail.address,
          unit_count: String(detail.unit_count),
        })
      } catch (error) {
        addToast(
          error instanceof Error ? error.message : 'Failed to load scheme',
          'error',
        )
      } finally {
        setLoading(false)
      }
    }

    load()
  }, [addToast, schemeId])

  async function handleSchemeSave() {
    if (!scheme) return
    const unitCount = Number(schemeForm.unit_count)
    if (!schemeForm.name.trim() || !schemeForm.address.trim() || Number.isNaN(unitCount) || unitCount <= 0) {
      addToast('Name, address, and unit count are required', 'error')
      return
    }

    setSavingScheme(true)
    try {
      const updated = await updateScheme(scheme.id, {
        name: schemeForm.name.trim(),
        address: schemeForm.address.trim(),
        unit_count: unitCount,
      })
      setScheme(current => current ? {
        ...current,
        ...updated,
      } : current)
      setSchemeForm({
        name: updated.name,
        address: updated.address,
        unit_count: String(updated.unit_count),
      })
      addToast('Scheme settings saved', 'success')
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to save scheme settings',
        'error',
      )
    } finally {
      setSavingScheme(false)
    }
  }

  function openCreateUnit() {
    setEditingUnit(null)
    setUnitForm(EMPTY_UNIT_FORM)
    setShowUnitModal(true)
  }

  function openEditUnit(unit: UnitInfo) {
    setEditingUnit(unit)
    setUnitForm(toUnitForm(unit))
    setShowUnitModal(true)
  }

  async function handleUnitSave() {
    if (!scheme) return
    const floor = Number(unitForm.floor)
    const sectionValuePct = Number(unitForm.section_value_pct)
    if (
      !unitForm.identifier.trim() ||
      !unitForm.owner_name.trim() ||
      Number.isNaN(floor) ||
      Number.isNaN(sectionValuePct) ||
      sectionValuePct <= 0
    ) {
      addToast('Unit identifier, owner name, floor, and section value are required', 'error')
      return
    }

    const payload = {
      identifier: unitForm.identifier.trim(),
      owner_name: unitForm.owner_name.trim(),
      floor,
      section_value_bps: Math.round(sectionValuePct * 100),
    }

    setSavingUnit(true)
    try {
      const savedUnit = editingUnit
        ? await updateSchemeUnit(scheme.id, editingUnit.id, payload)
        : await createSchemeUnit(scheme.id, payload)

      setScheme(current => {
        if (!current) return current
        const units = editingUnit
          ? current.units.map(unit => unit.id === savedUnit.id ? savedUnit : unit)
          : [...current.units, savedUnit].sort((a, b) => a.identifier.localeCompare(b.identifier))
        return {
          ...current,
          units,
        }
      })
      setShowUnitModal(false)
      setEditingUnit(null)
      setUnitForm(EMPTY_UNIT_FORM)
      addToast(editingUnit ? 'Unit updated' : 'Unit created', 'success')
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to save unit',
        'error',
      )
    } finally {
      setSavingUnit(false)
    }
  }

  async function handlePasswordSave() {
    if (!pwForm.current || !pwForm.next || pwForm.next !== pwForm.confirm) return

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

  if (loading) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Loading scheme settings…
        </div>
      </div>
    )
  }

  if (!scheme) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <div className="bg-hover-subtle border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Scheme settings could not be loaded.
        </div>
      </div>
    )
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Settings</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Scheme Settings</h1>
      <p className="text-[14px] text-muted mb-8">Manage scheme details, unit data, and your account password.</p>

      <div className="bg-surface border border-border rounded-lg overflow-hidden mb-6">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Scheme details</span>
        </div>
        <div className="px-5 py-4 flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Scheme name</label>
            <input
              type="text"
              value={schemeForm.name}
              onChange={e => setSchemeForm(current => ({ ...current, name: e.target.value }))}
              disabled={!canEdit}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent disabled:bg-page disabled:text-muted"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Physical address</label>
            <input
              type="text"
              value={schemeForm.address}
              onChange={e => setSchemeForm(current => ({ ...current, address: e.target.value }))}
              disabled={!canEdit}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent disabled:bg-page disabled:text-muted"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Total units</label>
            <input
              type="number"
              value={schemeForm.unit_count}
              onChange={e => setSchemeForm(current => ({ ...current, unit_count: e.target.value }))}
              disabled={!canEdit}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent disabled:bg-page disabled:text-muted"
            />
          </div>
          {canEdit && (
            <div>
              <button
                onClick={handleSchemeSave}
                disabled={savingScheme}
                className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {savingScheme ? 'Saving…' : 'Save changes'}
              </button>
            </div>
          )}
        </div>
      </div>

      <div className="bg-surface border border-border rounded-lg overflow-hidden mb-6">
        <div className="px-5 py-3 border-b border-border flex items-center justify-between gap-3">
          <span className="text-[13px] font-semibold text-ink">Unit register</span>
          {canEdit && (
            <button
              onClick={openCreateUnit}
              className="text-[12px] font-semibold bg-accent text-white px-3 py-1.5 rounded hover:opacity-90 transition-colors"
            >
              + Add unit
            </button>
          )}
        </div>
        <div className="px-5 py-4">
          <div className="overflow-x-auto">
            <div className="min-w-[620px]">
              <div className="grid grid-cols-[70px_1fr_80px_110px_72px] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
                <span>Unit</span>
                <span>Owner</span>
                <span>Floor</span>
                <span>Section value</span>
                <span></span>
              </div>
              {scheme.units.length === 0 ? (
                <div className="py-8 text-center text-[13px] text-muted">No units captured yet.</div>
              ) : (
                scheme.units.map((unit, index) => (
                  <div key={unit.id} className={`grid grid-cols-[70px_1fr_80px_110px_72px] gap-4 items-center py-3 text-[13px] ${index < scheme.units.length - 1 ? 'border-b border-border' : ''}`}>
                    <span className="font-semibold text-ink">{unit.identifier}</span>
                    <span className="text-ink">{unit.owner_name}</span>
                    <span className="text-muted">{unit.floor}</span>
                    <span className="text-muted">{unit.section_value_pct.toFixed(2)}%</span>
                    <div className="text-right">
                      {canEdit ? (
                        <button onClick={() => openEditUnit(unit)} className="text-[12px] text-accent font-medium hover:underline">
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
        </div>
      </div>

      <Modal open={showUnitModal} onClose={() => setShowUnitModal(false)} title={editingUnit ? 'Edit unit' : 'Add unit'}>
        <div className="flex flex-col gap-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Unit identifier</label>
              <input
                type="text"
                value={unitForm.identifier}
                onChange={e => setUnitForm(current => ({ ...current, identifier: e.target.value }))}
                placeholder="e.g. 1A"
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
              />
            </div>
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Floor</label>
              <input
                type="number"
                value={unitForm.floor}
                onChange={e => setUnitForm(current => ({ ...current, floor: e.target.value }))}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
              />
            </div>
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Owner name</label>
            <input
              type="text"
              value={unitForm.owner_name}
              onChange={e => setUnitForm(current => ({ ...current, owner_name: e.target.value }))}
              placeholder="e.g. N. Nkosi"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Section value (%)</label>
            <input
              type="number"
              step="0.01"
              value={unitForm.section_value_pct}
              onChange={e => setUnitForm(current => ({ ...current, section_value_pct: e.target.value }))}
              placeholder="e.g. 4.17"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div className="flex justify-end gap-3 pt-1">
            <button onClick={() => setShowUnitModal(false)} className="text-[12px] text-muted hover:text-ink px-3 py-2">Cancel</button>
            <button onClick={handleUnitSave} disabled={savingUnit} className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed">
              {savingUnit ? 'Saving…' : editingUnit ? 'Update unit' : 'Create unit'}
            </button>
          </div>
        </div>
      </Modal>

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
