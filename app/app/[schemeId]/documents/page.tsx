'use client'
import { useState } from 'react'
import { useMockAuth } from '@/lib/mock-auth'
import { mockDocuments, type SchemeDocument } from '@/lib/mock/documents'
import { useToast } from '@/lib/toast'
import Modal from '@/components/Modal'

const CATEGORY_LABELS: Record<SchemeDocument['category'], string> = {
  rules:     'Conduct Rules',
  minutes:   'AGM Minutes',
  insurance: 'Insurance',
  financial: 'Financial',
  other:     'Other',
}

const FILE_TYPE_STYLES: Record<string, string> = {
  pdf:  'bg-red-bg text-red',
  xlsx: 'bg-green-bg text-green',
  docx: 'bg-accent-bg text-accent',
}

function formatBytes(bytes: number): string {
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function groupByCategory(docs: SchemeDocument[]): Record<string, SchemeDocument[]> {
  const order = Object.values(CATEGORY_LABELS)
  const grouped = docs.reduce((acc, doc) => {
    const key = CATEGORY_LABELS[doc.category]
    if (!acc[key]) acc[key] = []
    acc[key].push(doc)
    return acc
  }, {} as Record<string, SchemeDocument[]>)
  return Object.fromEntries(order.filter(k => grouped[k]).map(k => [k, grouped[k]]))
}

export default function DocumentsPage() {
  const { user } = useMockAuth()
  const { addToast } = useToast()

  const [documents, setDocuments] = useState<SchemeDocument[]>([...mockDocuments])
  const [categoryFilter, setCategoryFilter] = useState<string>('all')
  const [showModal, setShowModal] = useState(false)
  const [form, setForm] = useState({ name: '', category: 'other' as SchemeDocument['category'], file_type: 'pdf' as SchemeDocument['file_type'] })

  const canUpload = user?.role === 'agent'
  const allGrouped = groupByCategory(documents)
  const grouped = categoryFilter === 'all'
    ? allGrouped
    : Object.fromEntries(Object.entries(allGrouped).filter(([key]) => {
        const categoryKey = Object.entries(CATEGORY_LABELS).find(([, label]) => label === key)?.[0]
        return categoryKey === categoryFilter
      }))

  function handleUpload() {
    if (!form.name.trim()) return
    const newDoc: SchemeDocument = {
      id: `doc-${Date.now()}`,
      scheme_id: 'scheme-001',
      name: form.name.trim(),
      file_type: form.file_type,
      category: form.category,
      uploaded_at: new Date().toISOString(),
      uploaded_by_name: user?.orgName ?? 'Managing Agent',
      size_bytes: Math.floor(Math.random() * 500000) + 50000,
    }
    setDocuments(prev => [newDoc, ...prev])
    setShowModal(false)
    setForm({ name: '', category: 'other', file_type: 'pdf' })
    addToast(`"${newDoc.name}" uploaded successfully`, 'success')
  }

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Documents</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Documents</h1>
      <p className="text-[14px] text-muted mb-8">Scheme rules, minutes, and shared files.</p>

      <div className="flex items-center justify-between mb-6">
        <span className="text-[13px] text-muted">{documents.length} documents</span>
        {canUpload && (
          <button
            onClick={() => setShowModal(true)}
            className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors"
          >
            + Upload document
          </button>
        )}
      </div>

      <div className="mb-4 flex items-center gap-3">
        <label className="text-[12px] font-semibold text-muted">Category:</label>
        <select
          value={categoryFilter}
          onChange={e => setCategoryFilter(e.target.value)}
          className="border border-border rounded px-3 py-1.5 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
        >
          <option value="all">All categories</option>
          <option value="rules">Conduct Rules</option>
          <option value="minutes">AGM Minutes</option>
          <option value="insurance">Insurance</option>
          <option value="financial">Financial</option>
          <option value="other">Other</option>
        </select>
      </div>

      <div className="flex flex-col gap-6">
        {Object.entries(grouped).map(([category, docs]) => (
          <div key={category}>
            <h2 className="text-[11px] font-semibold text-muted uppercase tracking-[0.08em] mb-3">{category}</h2>
            <div className="bg-surface border border-border rounded-lg overflow-hidden">
              {docs.map((doc, i) => (
                <div key={doc.id} className={`flex items-center gap-4 px-5 py-3 text-[13px] ${i < docs.length - 1 ? 'border-b border-border' : ''}`}>
                  <span className={`text-[10px] font-bold px-[6px] py-[2px] rounded uppercase ${FILE_TYPE_STYLES[doc.file_type] ?? 'bg-page text-muted'}`}>
                    {doc.file_type}
                  </span>
                  <div className="flex-1 min-w-0">
                    <div className="font-medium text-ink truncate">{doc.name}</div>
                    <div className="text-[11px] text-muted mt-0.5">
                      {new Date(doc.uploaded_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' })}
                      {' · '}{formatBytes(doc.size_bytes)}
                    </div>
                  </div>
                  <button
                    onClick={() => addToast(`Downloading "${doc.name}"…`, 'info')}
                    className="text-[12px] text-accent font-medium hover:underline flex-shrink-0"
                  >
                    Download
                  </button>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>

      <Modal open={showModal} onClose={() => setShowModal(false)} title="Upload document">
        <div className="flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Document name *</label>
            <input
              type="text"
              value={form.name}
              onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
              placeholder="e.g. AGM Minutes November 2025"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Category</label>
              <select
                value={form.category}
                onChange={e => setForm(f => ({ ...f, category: e.target.value as SchemeDocument['category'] }))}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
              >
                {(Object.entries(CATEGORY_LABELS) as [SchemeDocument['category'], string][]).map(([val, label]) => (
                  <option key={val} value={val}>{label}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">File type</label>
              <select
                value={form.file_type}
                onChange={e => setForm(f => ({ ...f, file_type: e.target.value as SchemeDocument['file_type'] }))}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
              >
                {['pdf', 'docx', 'xlsx', 'jpg', 'png'].map(t => (
                  <option key={t} value={t}>{t.toUpperCase()}</option>
                ))}
              </select>
            </div>
          </div>
          <div className="border-2 border-dashed border-border rounded-lg px-6 py-8 text-center text-[13px] text-muted">
            File picker coming when backend is connected
          </div>
          <div className="flex gap-2 pt-1">
            <button
              onClick={handleUpload}
              disabled={!form.name.trim()}
              className="flex-1 bg-accent text-white text-[13px] font-semibold py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Upload
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
