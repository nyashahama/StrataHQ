'use client'

import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'next/navigation'

import Modal from '@/components/Modal'
import { useAuth } from '@/lib/auth'
import {
  assignMaintenanceRequest,
  createMaintenanceRequest,
  getMaintenanceDashboard,
  resolveMaintenanceRequest,
} from '@/lib/maintenance-api'
import type { MaintenanceDashboard, MaintenanceRequestInfo } from '@/lib/maintenance'
import { useToast } from '@/lib/toast'

const STATUS_STYLES: Record<string, string> = {
  open: 'bg-red-bg text-red',
  in_progress: 'bg-yellowbg text-amber',
  pending_approval: 'bg-accent-bg text-accent',
  resolved: 'bg-green-bg text-green',
}

const STATUS_LABELS: Record<string, string> = {
  open: 'Open',
  in_progress: 'In progress',
  pending_approval: 'Pending',
  resolved: 'Resolved',
}

const CATEGORY_ICONS: Record<string, string> = {
  plumbing: 'P',
  electrical: 'E',
  structural: 'S',
  garden: 'G',
  pool: 'L',
  other: 'W',
}

const EMPTY_JOB_FORM = { title: '', category: 'other', description: '' }
const EMPTY_ASSIGN_FORM = { name: '', phone: '' }

export default function MaintenancePage() {
  const { user } = useAuth()
  const { addToast } = useToast()
  const params = useParams()
  const schemeId = params.schemeId as string

  const [dashboard, setDashboard] = useState<MaintenanceDashboard | null>(null)
  const [loading, setLoading] = useState(true)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [jobForm, setJobForm] = useState(EMPTY_JOB_FORM)
  const [assignJobId, setAssignJobId] = useState<string | null>(null)
  const [assignForm, setAssignForm] = useState(EMPTY_ASSIGN_FORM)
  const [savingCreate, setSavingCreate] = useState(false)
  const [savingAssign, setSavingAssign] = useState(false)
  const [resolvingId, setResolvingId] = useState<string | null>(null)

  const isResident = user?.role === 'resident'
  const canManage = !isResident

  const unitLabel = useMemo(
    () =>
      user?.scheme_memberships.find(membership => membership.scheme_id === schemeId)
        ?.unit_identifier ?? '',
    [schemeId, user?.scheme_memberships],
  )

  useEffect(() => {
    async function load() {
      try {
        setLoading(true)
        setDashboard(await getMaintenanceDashboard(schemeId))
      } catch (error) {
        addToast(
          error instanceof Error ? error.message : 'Failed to load maintenance requests',
          'error',
        )
      } finally {
        setLoading(false)
      }
    }

    load()
  }, [addToast, schemeId])

  async function refreshDashboard() {
    setDashboard(await getMaintenanceDashboard(schemeId))
  }

  async function handleCreate() {
    if (!jobForm.title.trim() || !jobForm.description.trim()) return

    setSavingCreate(true)
    try {
      await createMaintenanceRequest(schemeId, {
        title: jobForm.title.trim(),
        description: jobForm.description.trim(),
        category: jobForm.category,
      })
      setShowCreateModal(false)
      setJobForm(EMPTY_JOB_FORM)
      await refreshDashboard()
      addToast(
        isResident ? 'Request submitted for approval' : 'Maintenance job created',
        'success',
      )
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to create maintenance request',
        'error',
      )
    } finally {
      setSavingCreate(false)
    }
  }

  async function handleAssign() {
    if (!assignJobId || !assignForm.name.trim()) return

    setSavingAssign(true)
    try {
      await assignMaintenanceRequest(schemeId, assignJobId, {
        contractor_name: assignForm.name.trim(),
        contractor_phone: assignForm.phone.trim() || null,
      })
      setAssignJobId(null)
      setAssignForm(EMPTY_ASSIGN_FORM)
      await refreshDashboard()
      addToast('Job approved and contractor assigned', 'success')
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to assign contractor',
        'error',
      )
    } finally {
      setSavingAssign(false)
    }
  }

  async function handleResolve(requestId: string) {
    setResolvingId(requestId)
    try {
      await resolveMaintenanceRequest(schemeId, requestId)
      await refreshDashboard()
      addToast('Job marked as resolved', 'success')
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to resolve maintenance request',
        'error',
      )
    } finally {
      setResolvingId(null)
    }
  }

  const requests = dashboard?.requests ?? []

  if (loading) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Loading maintenance…
        </div>
      </div>
    )
  }

  if (!dashboard) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <div className="bg-hover-subtle border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Maintenance requests could not be loaded.
        </div>
      </div>
    )
  }

  if (isResident) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › Maintenance</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Maintenance</h1>
        <p className="text-[14px] text-muted mb-8">
          Your maintenance requests for {unitLabel ? `Unit ${unitLabel}` : 'your unit'}.
        </p>

        <div className="flex items-center justify-between mb-6">
          <span className="text-[13px] text-muted">
            {requests.length} request{requests.length !== 1 ? 's' : ''}
          </span>
          <button
            onClick={() => setShowCreateModal(true)}
            className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors"
          >
            + Submit request
          </button>
        </div>

        {requests.length === 0 ? (
          <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
            No maintenance requests submitted yet.
          </div>
        ) : (
          <div className="flex flex-col gap-3">
            {requests.map(request => (
              <MaintenanceCard key={request.id} request={request} />
            ))}
          </div>
        )}

        <Modal open={showCreateModal} onClose={() => !savingCreate && setShowCreateModal(false)} title="Submit maintenance request">
          <JobForm
            form={jobForm}
            setForm={setJobForm}
            submitting={savingCreate}
            onSubmit={handleCreate}
            onCancel={() => setShowCreateModal(false)}
          />
        </Modal>
      </div>
    )
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Maintenance</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Maintenance</h1>
      <p className="text-[14px] text-muted mb-8">Log jobs, approve resident requests, assign contractors, and track SLA performance.</p>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
        {[
          { label: 'Open jobs', value: String(dashboard.open_count) },
          { label: 'SLA breaches', value: String(dashboard.sla_breached_count) },
          { label: 'Pending approval', value: String(dashboard.pending_approval_count) },
          { label: 'Resolved this month', value: String(dashboard.resolved_this_month) },
        ].map(({ label, value }) => (
          <div key={label} className="bg-surface border border-border rounded-lg px-5 py-4">
            <div className="text-[24px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      <div className="bg-surface border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border flex items-center justify-between">
          <span className="text-[13px] font-semibold text-ink">Work Orders</span>
          {canManage && (
            <button
              onClick={() => setShowCreateModal(true)}
              className="text-[12px] font-semibold bg-accent text-white px-3 py-1.5 rounded hover:opacity-90 transition-colors"
            >
              + New job
            </button>
          )}
        </div>
        <div className="px-5 py-3 flex flex-col gap-0">
          {requests.map((request, index) => (
            <div key={request.id} className={`flex gap-3 items-start py-4 ${index < requests.length - 1 ? 'border-b border-border' : ''}`}>
              <div className="w-9 h-9 rounded bg-page border border-border flex-shrink-0 grid place-items-center text-[13px] font-semibold text-ink mt-0.5">
                {CATEGORY_ICONS[request.category]}
              </div>
              <div className="flex-1 min-w-0">
                <div className="text-[13px] font-semibold text-ink mb-[2px] truncate">{request.title}</div>
                <div className="text-[11px] text-muted">
                  {request.contractor_name ?? 'No contractor assigned'}
                  {request.submitted_by_unit && ` · Unit ${request.submitted_by_unit}`}
                </div>
                {request.sla_breached ? (
                  <div className="text-[11px] text-red font-medium mt-[2px]">SLA breached</div>
                ) : request.status !== 'resolved' ? (
                  <div className="text-[11px] text-muted mt-[2px]">
                    SLA: {request.sla_hours}h · submitted {new Date(request.created_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' })}
                  </div>
                ) : (
                  request.resolved_at && (
                    <div className="text-[11px] text-green mt-[2px]">
                      Resolved {new Date(request.resolved_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' })}
                    </div>
                  )
                )}
              </div>
              <div className="flex items-center gap-2 flex-shrink-0">
                <span className={`text-[11px] font-semibold px-[10px] py-[3px] rounded-full ${STATUS_STYLES[request.status]}`}>
                  {STATUS_LABELS[request.status]}
                </span>
                {request.status === 'pending_approval' && (
                  <button
                    onClick={() => {
                      setAssignJobId(request.id)
                      setAssignForm(EMPTY_ASSIGN_FORM)
                    }}
                    className="text-[11px] text-accent font-medium hover:underline"
                  >
                    Approve
                  </button>
                )}
                {request.status === 'open' && (
                  <button
                    onClick={() => {
                      setAssignJobId(request.id)
                      setAssignForm(EMPTY_ASSIGN_FORM)
                    }}
                    className="text-[11px] text-accent font-medium hover:underline"
                  >
                    Assign
                  </button>
                )}
                {request.status === 'in_progress' && (
                  <button
                    onClick={() => handleResolve(request.id)}
                    disabled={resolvingId === request.id}
                    className="text-[11px] text-green font-medium hover:underline disabled:opacity-40"
                  >
                    {resolvingId === request.id ? 'Resolving…' : 'Resolve'}
                  </button>
                )}
              </div>
            </div>
          ))}
          {requests.length === 0 && (
            <div className="py-10 text-center text-[13px] text-muted">
              No maintenance jobs have been logged yet.
            </div>
          )}
        </div>
      </div>

      <Modal open={assignJobId !== null} onClose={() => !savingAssign && setAssignJobId(null)} title="Assign contractor">
        <div className="flex flex-col gap-4">
          <p className="text-[13px] text-muted">Assign a contractor to move this job into progress.</p>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Contractor name *</label>
            <input
              type="text"
              value={assignForm.name}
              onChange={event => setAssignForm(current => ({ ...current, name: event.target.value }))}
              placeholder="e.g. AquaFix Plumbing"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Contractor phone</label>
            <input
              type="tel"
              value={assignForm.phone}
              onChange={event => setAssignForm(current => ({ ...current, phone: event.target.value }))}
              placeholder="+27 21 555 0199"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div className="flex justify-end gap-3 pt-1">
            <button onClick={() => setAssignJobId(null)} className="text-[12px] text-muted hover:text-ink px-3 py-2">
              Cancel
            </button>
            <button
              onClick={handleAssign}
              disabled={!assignForm.name.trim() || savingAssign}
              className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {savingAssign ? 'Saving…' : 'Assign contractor'}
            </button>
          </div>
        </div>
      </Modal>

      <Modal open={showCreateModal} onClose={() => !savingCreate && setShowCreateModal(false)} title="New maintenance job">
        <JobForm
          form={jobForm}
          setForm={setJobForm}
          submitting={savingCreate}
          onSubmit={handleCreate}
          onCancel={() => setShowCreateModal(false)}
        />
      </Modal>
    </div>
  )
}

function MaintenanceCard({ request }: { request: MaintenanceRequestInfo }) {
  return (
    <div className="bg-surface border border-border rounded-lg px-5 py-4 flex gap-3">
      <div className="w-9 h-9 rounded bg-page border border-border flex-shrink-0 grid place-items-center text-[13px] font-semibold text-ink">
        {CATEGORY_ICONS[request.category]}
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <span className="text-[13px] font-semibold text-ink">{request.title}</span>
          <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${STATUS_STYLES[request.status]}`}>
            {STATUS_LABELS[request.status]}
          </span>
        </div>
        <div className="text-[12px] text-muted">
          {request.contractor_name ?? 'No contractor assigned'}
        </div>
        <div className="text-[11px] text-muted mt-1">
          Submitted {new Date(request.created_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' })}
          {request.resolved_at && ` · Resolved ${new Date(request.resolved_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' })}`}
        </div>
      </div>
    </div>
  )
}

function JobForm({
  form,
  setForm,
  submitting,
  onSubmit,
  onCancel,
}: {
  form: { title: string; category: string; description: string }
  setForm: React.Dispatch<React.SetStateAction<{ title: string; category: string; description: string }>>
  submitting: boolean
  onSubmit: () => void
  onCancel: () => void
}) {
  return (
    <div className="flex flex-col gap-4">
      <div>
        <label className="text-[12px] font-semibold text-ink block mb-1">Title *</label>
        <input
          type="text"
          value={form.title}
          onChange={event => setForm(current => ({ ...current, title: event.target.value }))}
          placeholder="e.g. Leaking tap in Unit 3A"
          className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
        />
      </div>
      <div>
        <label className="text-[12px] font-semibold text-ink block mb-1">Category</label>
        <select
          value={form.category}
          onChange={event => setForm(current => ({ ...current, category: event.target.value }))}
          className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
        >
          {['plumbing', 'electrical', 'structural', 'garden', 'pool', 'other'].map(category => (
            <option key={category} value={category}>
              {category.charAt(0).toUpperCase() + category.slice(1)}
            </option>
          ))}
        </select>
      </div>
      <div>
        <label className="text-[12px] font-semibold text-ink block mb-1">Description</label>
        <textarea
          value={form.description}
          onChange={event => setForm(current => ({ ...current, description: event.target.value }))}
          placeholder="Brief description of the issue"
          rows={3}
          className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent resize-none"
        />
      </div>
      <div className="flex gap-2 pt-1">
        <button
          onClick={onSubmit}
          disabled={!form.title.trim() || !form.description.trim() || submitting}
          className="flex-1 bg-accent text-white text-[13px] font-semibold py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {submitting ? 'Submitting…' : 'Submit'}
        </button>
        <button
          onClick={onCancel}
          className="px-4 text-[13px] font-medium text-muted hover:text-ink border border-border rounded py-2 transition-colors"
        >
          Cancel
        </button>
      </div>
    </div>
  )
}
