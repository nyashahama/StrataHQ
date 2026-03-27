'use client'
import { useState } from 'react'
import { useMockAuth } from '@/lib/mock-auth'
import { mockMaintenanceRequests, type MaintenanceRequest } from '@/lib/mock/maintenance'
import { useToast } from '@/lib/toast'
import Modal from '@/components/Modal'

const STATUS_STYLES: Record<string, string> = {
  open:             'bg-red-bg text-red',
  in_progress:      'bg-yellowbg text-[#92400e]',
  pending_approval: 'bg-accent-bg text-accent',
  resolved:         'bg-green-bg text-green',
}

const STATUS_LABELS: Record<string, string> = {
  open:             'Open',
  in_progress:      'In progress',
  pending_approval: 'Pending',
  resolved:         'Resolved',
}

const CATEGORY_ICONS: Record<string, string> = {
  plumbing:   '🚿',
  electrical: '💡',
  structural: '🏗️',
  garden:     '🌿',
  pool:       '🏊',
  other:      '🔧',
}

function isSlaBreached(req: MaintenanceRequest): boolean {
  if (req.status === 'resolved') return false
  const created = new Date(req.created_at).getTime()
  const now = new Date().getTime()
  const hoursElapsed = (now - created) / (1000 * 60 * 60)
  return hoursElapsed > req.sla_hours
}

export default function MaintenancePage() {
  const { user } = useMockAuth()
  const { addToast } = useToast()

  const [jobs, setJobs] = useState<MaintenanceRequest[]>([...mockMaintenanceRequests])
  const [showModal, setShowModal] = useState(false)
  const [form, setForm] = useState({ title: '', category: 'other', description: '' })
  const [approveJobId, setApproveJobId] = useState<string | null>(null)
  const [contractorForm, setContractorForm] = useState({ name: '', phone: '' })

  function handleApproveConfirm() {
    if (!approveJobId || !contractorForm.name.trim()) return
    setJobs(prev => prev.map(j => j.id === approveJobId
      ? { ...j, status: 'in_progress' as const, contractor_name: contractorForm.name.trim(), contractor_phone: contractorForm.phone.trim() || null }
      : j
    ))
    setApproveJobId(null)
    setContractorForm({ name: '', phone: '' })
    addToast('Job approved and contractor assigned', 'success')
  }

  function handleSubmit() {
    if (!form.title.trim()) return
    const newJob: MaintenanceRequest = {
      id: `mr-${Date.now()}`,
      scheme_id: 'scheme-001',
      unit_id: user?.role === 'resident' ? 'unit-4b' : null,
      title: form.title.trim(),
      description: form.description.trim(),
      category: form.category as MaintenanceRequest['category'],
      status: user?.role === 'resident' ? 'pending_approval' : 'open',
      contractor_name: null,
      contractor_phone: null,
      sla_hours: 72,
      created_at: new Date().toISOString(),
      resolved_at: null,
      submitted_by_unit: user?.role === 'resident' ? (user.unitIdentifier ?? null) : null,
    }
    setJobs(prev => [newJob, ...prev])
    setShowModal(false)
    setForm({ title: '', category: 'other', description: '' })
    addToast(user?.role === 'resident' ? 'Request submitted for approval' : 'Job created', 'success')
  }

  const modalTitle = user?.role === 'resident' ? 'Submit maintenance request' : 'New maintenance job'

  // Resident: show only their unit's requests
  if (user?.role === 'resident') {
    const myRequests = jobs.filter(r => r.submitted_by_unit === user.unitIdentifier)
    return (
      <div className="px-8 py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › Maintenance</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Maintenance</h1>
        <p className="text-[14px] text-muted mb-8">Your maintenance requests for Unit {user.unitIdentifier}.</p>

        <div className="flex items-center justify-between mb-6">
          <span className="text-[13px] text-muted">{myRequests.length} request{myRequests.length !== 1 ? 's' : ''}</span>
          <button
            onClick={() => setShowModal(true)}
            className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:bg-[#245a96] transition-colors"
          >
            + Submit request
          </button>
        </div>

        {myRequests.length === 0 ? (
          <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
            No maintenance requests submitted yet.
          </div>
        ) : (
          <div className="flex flex-col gap-3">
            {myRequests.map(req => (
              <div key={req.id} className="bg-white border border-border rounded-lg px-5 py-4 flex gap-3">
                <div className="w-9 h-9 rounded bg-page border border-border flex-shrink-0 grid place-items-center text-[16px]">
                  {CATEGORY_ICONS[req.category]}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-[13px] font-semibold text-ink">{req.title}</span>
                    <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${STATUS_STYLES[req.status]}`}>
                      {STATUS_LABELS[req.status]}
                    </span>
                  </div>
                  <div className="text-[12px] text-muted">{req.contractor_name ?? 'No contractor assigned'}</div>
                  <div className="text-[11px] text-muted mt-1">
                    Submitted {new Date(req.created_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' })}
                    {req.resolved_at && ` · Resolved ${new Date(req.resolved_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' })}`}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        <Modal open={showModal} onClose={() => setShowModal(false)} title={modalTitle}>
          <JobForm form={form} setForm={setForm} onSubmit={handleSubmit} onCancel={() => setShowModal(false)} />
        </Modal>
      </div>
    )
  }

  // Agent / Trustee view
  const canEdit = user?.role === 'agent'
  const open = jobs.filter(r => r.status !== 'resolved')
  const breached = open.filter(isSlaBreached)
  const pendingApproval = open.filter(r => r.status === 'pending_approval')
  const resolvedCount = jobs.filter(r => r.status === 'resolved').length

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Maintenance</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Maintenance</h1>
      <p className="text-[14px] text-muted mb-8">Log and track maintenance requests.</p>

      {/* Stat cards */}
      <div className="grid grid-cols-4 gap-4 mb-8">
        {[
          { label: 'Open jobs',           value: String(open.length) },
          { label: 'SLA breaches',        value: String(breached.length) },
          { label: 'Pending approval',    value: String(pendingApproval.length) },
          { label: 'Resolved this month', value: String(resolvedCount) },
        ].map(({ label, value }) => (
          <div key={label} className="bg-white border border-border rounded-lg px-5 py-4">
            <div className="text-[24px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      {/* Work orders */}
      <div className="bg-white border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border flex items-center justify-between">
          <span className="text-[13px] font-semibold text-ink">Work Orders</span>
          {canEdit && (
            <button
              onClick={() => setShowModal(true)}
              className="text-[12px] font-semibold bg-accent text-white px-3 py-1.5 rounded hover:bg-[#245a96] transition-colors"
            >
              + New job
            </button>
          )}
        </div>
        <div className="px-5 py-3 flex flex-col gap-0">
          {jobs.map((req, i) => {
            const breachedSla = isSlaBreached(req)
            return (
              <div key={req.id} className={`flex gap-3 items-start py-4 ${i < jobs.length - 1 ? 'border-b border-border' : ''}`}>
                <div className="w-9 h-9 rounded bg-page border border-border flex-shrink-0 grid place-items-center text-[15px] mt-0.5">
                  {CATEGORY_ICONS[req.category]}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="text-[13px] font-semibold text-ink mb-[2px]">{req.title}</div>
                  <div className="text-[11px] text-muted">
                    {req.contractor_name ?? 'No contractor assigned'}
                    {req.submitted_by_unit && ` · Unit ${req.submitted_by_unit}`}
                  </div>
                  {breachedSla && (
                    <div className="text-[11px] text-red font-medium mt-[2px]">⚠ SLA breached</div>
                  )}
                  {!breachedSla && req.status !== 'resolved' && (
                    <div className="text-[11px] text-muted mt-[2px]">
                      SLA: {req.sla_hours}h · submitted {new Date(req.created_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' })}
                    </div>
                  )}
                  {req.status === 'resolved' && req.resolved_at && (
                    <div className="text-[11px] text-green mt-[2px]">
                      Resolved {new Date(req.resolved_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' })}
                    </div>
                  )}
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  <span className={`text-[11px] font-semibold px-[10px] py-[3px] rounded-full ${STATUS_STYLES[req.status]}`}>
                    {STATUS_LABELS[req.status]}
                  </span>
                  {canEdit && req.status === 'pending_approval' && (
                    <button
                      onClick={() => { setApproveJobId(req.id); setContractorForm({ name: '', phone: '' }) }}
                      className="text-[11px] text-accent font-medium hover:underline"
                    >
                      Approve
                    </button>
                  )}
                  {canEdit && req.status === 'in_progress' && (
                    <button
                      onClick={() => {
                        setJobs(prev => prev.map(j => j.id === req.id
                          ? { ...j, status: 'resolved' as const, resolved_at: new Date().toISOString() }
                          : j
                        ))
                        addToast('Job marked as resolved', 'success')
                      }}
                      className="text-[11px] text-green font-medium hover:underline"
                    >
                      Resolve
                    </button>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      </div>

      <Modal open={approveJobId !== null} onClose={() => setApproveJobId(null)} title="Approve job">
        <div className="flex flex-col gap-4">
          <p className="text-[13px] text-muted">Assign a contractor before approving this job.</p>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Contractor name *</label>
            <input
              type="text"
              value={contractorForm.name}
              onChange={e => setContractorForm(f => ({ ...f, name: e.target.value }))}
              placeholder="e.g. AquaFix Plumbing"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Contractor phone</label>
            <input
              type="tel"
              value={contractorForm.phone}
              onChange={e => setContractorForm(f => ({ ...f, phone: e.target.value }))}
              placeholder="+27 21 555 0199"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
            />
          </div>
          <div className="flex justify-end gap-3 pt-1">
            <button onClick={() => setApproveJobId(null)} className="text-[12px] text-muted hover:text-ink px-3 py-2">Cancel</button>
            <button
              onClick={handleApproveConfirm}
              disabled={!contractorForm.name.trim()}
              className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:bg-[#245a96] transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Approve &amp; assign
            </button>
          </div>
        </div>
      </Modal>

      <Modal open={showModal} onClose={() => setShowModal(false)} title={modalTitle}>
        <JobForm form={form} setForm={setForm} onSubmit={handleSubmit} onCancel={() => setShowModal(false)} />
      </Modal>
    </div>
  )
}

function JobForm({
  form,
  setForm,
  onSubmit,
  onCancel,
}: {
  form: { title: string; category: string; description: string }
  setForm: React.Dispatch<React.SetStateAction<{ title: string; category: string; description: string }>>
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
          onChange={e => setForm(f => ({ ...f, title: e.target.value }))}
          placeholder="e.g. Leaking tap in Unit 3A"
          className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
        />
      </div>
      <div>
        <label className="text-[12px] font-semibold text-ink block mb-1">Category</label>
        <select
          value={form.category}
          onChange={e => setForm(f => ({ ...f, category: e.target.value }))}
          className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
        >
          {['plumbing', 'electrical', 'structural', 'garden', 'pool', 'other'].map(c => (
            <option key={c} value={c}>{c.charAt(0).toUpperCase() + c.slice(1)}</option>
          ))}
        </select>
      </div>
      <div>
        <label className="text-[12px] font-semibold text-ink block mb-1">Description</label>
        <textarea
          value={form.description}
          onChange={e => setForm(f => ({ ...f, description: e.target.value }))}
          placeholder="Brief description of the issue"
          rows={3}
          className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent resize-none"
        />
      </div>
      <div className="flex gap-2 pt-1">
        <button
          onClick={onSubmit}
          disabled={!form.title.trim()}
          className="flex-1 bg-accent text-white text-[13px] font-semibold py-2 rounded hover:bg-[#245a96] transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          Submit
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
