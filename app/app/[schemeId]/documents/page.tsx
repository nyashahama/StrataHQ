'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { mockDocuments, type SchemeDocument } from '@/lib/mock/documents'

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

// Group documents by category
function groupByCategory(docs: SchemeDocument[]): Record<string, SchemeDocument[]> {
  return docs.reduce((acc, doc) => {
    const key = CATEGORY_LABELS[doc.category]
    if (!acc[key]) acc[key] = []
    acc[key].push(doc)
    return acc
  }, {} as Record<string, SchemeDocument[]>)
}

export default function DocumentsPage() {
  const { user } = useMockAuth()
  const canUpload = user?.role === 'agent'
  const grouped = groupByCategory(mockDocuments)

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Documents</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Documents</h1>
      <p className="text-[14px] text-muted mb-8">Scheme rules, minutes, and shared files.</p>

      <div className="flex items-center justify-between mb-6">
        <span className="text-[13px] text-muted">{mockDocuments.length} documents</span>
        {canUpload && (
          <button className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:bg-[#245a96] transition-colors">
            + Upload document
          </button>
        )}
      </div>

      <div className="flex flex-col gap-6">
        {Object.entries(grouped).map(([category, docs]) => (
          <div key={category}>
            <h2 className="text-[11px] font-semibold text-muted uppercase tracking-[0.08em] mb-3">{category}</h2>
            <div className="bg-white border border-border rounded-lg overflow-hidden">
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
                  <button className="text-[12px] text-accent font-medium hover:underline flex-shrink-0">
                    Download
                  </button>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
