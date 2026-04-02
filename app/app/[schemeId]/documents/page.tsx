'use client'

import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'next/navigation'

import Modal from '@/components/Modal'
import { createDocument, deleteDocument, getDocumentsDashboard } from '@/lib/documents-api'
import type { DocumentCategory, DocumentFileType, SchemeDocumentInfo } from '@/lib/documents'
import { useAuth } from '@/lib/auth'
import { useToast } from '@/lib/toast'

const CATEGORY_LABELS: Record<DocumentCategory, string> = {
  rules: 'Conduct Rules',
  minutes: 'AGM Minutes',
  insurance: 'Insurance',
  financial: 'Financial',
  other: 'Other',
}

const FILE_TYPE_STYLES: Record<string, string> = {
  pdf: 'bg-red-bg text-red',
  xlsx: 'bg-green-bg text-green',
  docx: 'bg-accent-bg text-accent',
  jpg: 'bg-page text-muted',
  png: 'bg-page text-muted',
}

function formatBytes(bytes: number): string {
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function groupByCategory(docs: SchemeDocumentInfo[]): Record<string, SchemeDocumentInfo[]> {
  const order = Object.values(CATEGORY_LABELS)
  const grouped = docs.reduce((acc, doc) => {
    const key = CATEGORY_LABELS[doc.category]
    if (!acc[key]) acc[key] = []
    acc[key].push(doc)
    return acc
  }, {} as Record<string, SchemeDocumentInfo[]>)
  return Object.fromEntries(order.filter(key => grouped[key]).map(key => [key, grouped[key]]))
}

const EMPTY_FORM = {
  name: '',
  category: 'other' as DocumentCategory,
}

export default function DocumentsPage() {
  const { user } = useAuth()
  const { addToast } = useToast()
  const params = useParams()
  const schemeId = params.schemeId as string

  const [documents, setDocuments] = useState<SchemeDocumentInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [categoryFilter, setCategoryFilter] = useState<'all' | DocumentCategory>('all')
  const [showModal, setShowModal] = useState(false)
  const [form, setForm] = useState(EMPTY_FORM)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [deletingId, setDeletingId] = useState<string | null>(null)

  const canUpload = user?.role === 'admin'

  useEffect(() => {
    async function load() {
      try {
        setLoading(true)
        const dashboard = await getDocumentsDashboard(schemeId, categoryFilter)
        setDocuments(dashboard.documents)
      } catch (error) {
        addToast(
          error instanceof Error ? error.message : 'Failed to load documents',
          'error',
        )
      } finally {
        setLoading(false)
      }
    }

    load()
  }, [addToast, categoryFilter, schemeId])

  async function handleUpload() {
    if (!selectedFile || !form.name.trim()) return

    setUploading(true)
    try {
      const storageKey = await fileToDataURL(selectedFile)
      const fileType = fileTypeFor(selectedFile.name)
      const created = await createDocument(schemeId, {
        name: form.name.trim(),
        storage_key: storageKey,
        file_type: fileType,
        category: form.category,
        size_bytes: selectedFile.size,
      })
      if (categoryFilter === 'all' || categoryFilter === created.category) {
        setDocuments(current => [created, ...current])
      }
      setShowModal(false)
      setForm(EMPTY_FORM)
      setSelectedFile(null)
      addToast(`"${created.name}" uploaded successfully`, 'success')
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to upload document',
        'error',
      )
    } finally {
      setUploading(false)
    }
  }

  async function handleDelete(document: SchemeDocumentInfo) {
    setDeletingId(document.id)
    try {
      await deleteDocument(schemeId, document.id)
      setDocuments(current => current.filter(item => item.id !== document.id))
      addToast(`"${document.name}" deleted`, 'success')
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to delete document',
        'error',
      )
    } finally {
      setDeletingId(null)
    }
  }

  const grouped = useMemo(() => groupByCategory(documents), [documents])

  if (loading) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Loading documents…
        </div>
      </div>
    )
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Documents</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Documents</h1>
      <p className="text-[14px] text-muted mb-8">Scheme rules, minutes, insurance certificates, and shared files.</p>

      <div className="flex flex-wrap items-center justify-between gap-3 mb-6">
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

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <label className="text-[12px] font-semibold text-muted">Category:</label>
        <select
          value={categoryFilter}
          onChange={event => setCategoryFilter(event.target.value as 'all' | DocumentCategory)}
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
        {Object.keys(grouped).length === 0 ? (
          <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
            No documents match the selected filter.
          </div>
        ) : Object.entries(grouped).map(([category, docs]) => (
          <div key={category}>
            <h2 className="text-[11px] font-semibold text-muted uppercase tracking-[0.08em] mb-3">{category}</h2>
            <div className="bg-surface border border-border rounded-lg overflow-hidden">
              {docs.map((doc, index) => (
                <div key={doc.id} className={`flex flex-col items-start sm:flex-row sm:items-center gap-2 sm:gap-4 px-5 py-3 text-[13px] ${index < docs.length - 1 ? 'border-b border-border' : ''}`}>
                  <span className={`text-[10px] font-bold px-[6px] py-[2px] rounded uppercase ${FILE_TYPE_STYLES[doc.file_type] ?? 'bg-page text-muted'}`}>
                    {doc.file_type}
                  </span>
                  <div className="flex-1 min-w-0">
                    <div className="font-medium text-ink truncate">{doc.name}</div>
                    <div className="text-[11px] text-muted mt-0.5">
                      {new Date(doc.uploaded_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' })}
                      {' · '}{formatBytes(doc.size_bytes)}
                      {doc.uploaded_by_name && ` · ${doc.uploaded_by_name}`}
                    </div>
                  </div>
                  <button
                    onClick={() => triggerDownload(doc)}
                    className="text-[12px] text-accent font-medium hover:underline flex-shrink-0"
                  >
                    Download
                  </button>
                  {canUpload && (
                    <button
                      onClick={() => handleDelete(doc)}
                      disabled={deletingId === doc.id}
                      className="text-[12px] text-red font-medium hover:underline disabled:opacity-40"
                    >
                      {deletingId === doc.id ? 'Deleting…' : 'Delete'}
                    </button>
                  )}
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>

      <Modal open={showModal} onClose={() => !uploading && setShowModal(false)} title="Upload document">
        <div className="flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Document name *</label>
            <input
              type="text"
              value={form.name}
              onChange={event => setForm(current => ({ ...current, name: event.target.value }))}
              placeholder="e.g. AGM Minutes November 2025"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Category</label>
            <select
              value={form.category}
              onChange={event => setForm(current => ({ ...current, category: event.target.value as DocumentCategory }))}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            >
              {(Object.entries(CATEGORY_LABELS) as [DocumentCategory, string][]).map(([value, label]) => (
                <option key={value} value={value}>{label}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">File *</label>
            <input
              type="file"
              accept=".pdf,.docx,.xlsx,.jpg,.jpeg,.png"
              onChange={event => {
                const file = event.target.files?.[0] ?? null
                setSelectedFile(file)
                if (file && !form.name.trim()) {
                  const fallbackName = file.name.replace(/\.[^.]+$/, '')
                  setForm(current => ({ ...current, name: fallbackName }))
                }
              }}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
            {selectedFile && (
              <p className="text-[11px] text-muted mt-2">
                {selectedFile.name} · {formatBytes(selectedFile.size)}
              </p>
            )}
          </div>
          <div className="flex gap-2 pt-1">
            <button
              onClick={handleUpload}
              disabled={!form.name.trim() || !selectedFile || uploading}
              className="flex-1 bg-accent text-white text-[13px] font-semibold py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {uploading ? 'Uploading…' : 'Upload'}
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

async function fileToDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result))
    reader.onerror = () => reject(new Error('Failed to read file'))
    reader.readAsDataURL(file)
  })
}

function fileTypeFor(fileName: string): DocumentFileType {
  const extension = fileName.split('.').pop()?.toLowerCase()
  switch (extension) {
    case 'docx':
      return 'docx'
    case 'xlsx':
      return 'xlsx'
    case 'jpg':
    case 'jpeg':
      return 'jpg'
    case 'png':
      return 'png'
    default:
      return 'pdf'
  }
}

function triggerDownload(document: SchemeDocumentInfo) {
  const link = window.document.createElement('a')
  link.href = document.storage_key
  link.download = `${document.name}.${document.file_type}`
  link.click()
}
