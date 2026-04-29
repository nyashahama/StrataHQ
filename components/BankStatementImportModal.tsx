'use client'
import { useState, useRef, useEffect, useCallback } from 'react'
import {
  importBankStatementCsv,
  getBankStatementImport,
  applyBankStatementImport,
} from '@/lib/levy-api'
import type {
  LevyAccountInfo,
  BankStatementImportResponse,
  BankStatementManualMatchInput,
} from '@/lib/levy'

interface BankStatementImportModalProps {
  schemeId: string
  levyAccounts: LevyAccountInfo[]
  periodLabel: string
  onApplied: () => void
  onClose: () => void
}

type Step = 'upload' | 'review'

const ROW_STATUS_STYLES: Record<string, string> = {
  matched: 'bg-green-bg text-green',
  ambiguous: 'bg-yellowbg text-amber',
  unmatched: 'bg-red-bg text-red',
  applied: 'bg-green-bg text-green',
  skipped: 'bg-accent-bg text-accent',
  failed: 'bg-red-bg text-red',
}

function formatRand(cents: number): string {
  return `R ${(cents / 100).toLocaleString('en-ZA', { minimumFractionDigits: 0 })}`
}

export default function BankStatementImportModal({
  schemeId,
  levyAccounts,
  periodLabel,
  onApplied,
  onClose,
}: BankStatementImportModalProps) {
  const [, setStep] = useState<Step>('upload')
  const [importData, setImportData] = useState<BankStatementImportResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const [applying, setApplying] = useState(false)
  const [fileName, setFileName] = useState<string | null>(null)
  const [manualMatches, setManualMatches] = useState<Map<string, string>>(new Map())
  const fileInputRef = useRef<HTMLInputElement>(null)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const pollImport = useCallback(async (importId: string) => {
    try {
      const data = await getBankStatementImport(schemeId, importId)
      if (data.status === 'review_required' || data.status === 'applied') {
        if (pollRef.current) {
          clearInterval(pollRef.current)
          pollRef.current = null
        }
        setImportData(data)
        setStep('review')
        if (data.status === 'applied') {
          onApplied()
        }
        return
      }
      if (data.status === 'failed') {
        if (pollRef.current) {
          clearInterval(pollRef.current)
          pollRef.current = null
        }
        setError('The bank statement import failed. Please check the CSV format and try again.')
      }
    } catch {
      return
    }
  }, [schemeId, onApplied])

  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current)
    }
  }, [])

  async function handleUpload() {
    const file = fileInputRef.current?.files?.[0]
    if (!file) {
      setError('Please select a CSV file.')
      return
    }

    setError(null)
    setUploading(true)
    try {
      const data = await importBankStatementCsv(schemeId, 'fnb', file)
      setFileName(file.name)
      if (data.status === 'queued' || data.status === 'processing') {
        setImportData(data)
        pollRef.current = setInterval(() => pollImport(data.id), 2000)
      } else {
        setImportData(data)
        setStep('review')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to upload bank statement.')
    } finally {
      setUploading(false)
    }
  }

  async function handleApply() {
    setApplying(true)
    setError(null)
    try {
      const matches: BankStatementManualMatchInput[] = []
      const rows = importData?.rows ?? []
      for (const row of rows) {
        if (row.status === 'applied' || row.status === 'skipped') continue
        const matchedAccountId = row.matched_levy_account_id ?? manualMatches.get(row.id)
        if (!matchedAccountId) continue

        matches.push({
          row_id: row.id,
          account_id: matchedAccountId,
          payment_date: row.transaction_date,
          amount_cents: row.amount_cents,
          reference: `FNB-${row.row_number}-${row.transaction_date}-${row.amount_cents}`,
          bank_ref: row.reference,
        })
      }

      if (matches.length === 0) {
        setError('No rows to apply.')
        setApplying(false)
        return
      }

      const result = await applyBankStatementImport(schemeId, importData!.id, matches)
      setImportData(result)
      onApplied()
      setStep('upload')
      setManualMatches(new Map())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to apply bank statement import.')
    } finally {
      setApplying(false)
    }
  }

  function handleFileSelect() {
    fileInputRef.current?.click()
  }

  function setMatch(rowId: string, accountId: string) {
    setManualMatches(prev => {
      const next = new Map(prev)
      next.set(rowId, accountId)
      return next
    })
  }

  function renderUploadStep() {
    return (
      <div className="space-y-4">
        <div>
          <label className="block text-[12px] text-muted mb-2">
            Select an FNB CSV bank statement to import and auto-match payments.
          </label>
          <input
            ref={fileInputRef}
            type="file"
            accept=".csv"
            onChange={() => setError(null)}
            className="hidden"
          />
          <button
            onClick={handleFileSelect}
            onDrop={event => {
              event.preventDefault()
              const file = event.dataTransfer.files?.[0]
              if (file && fileInputRef.current) {
                const dt = new DataTransfer()
                dt.items.add(file)
                fileInputRef.current.files = dt.files
                setError(null)
              }
            }}
            onDragOver={event => event.preventDefault()}
            className="w-full border-2 border-dashed border-border rounded-lg px-6 py-8 text-center text-[13px] text-muted hover:border-accent transition-colors cursor-pointer"
          >
            <p className="mb-1 font-semibold text-ink">Choose a CSV file</p>
            <p className="text-[12px]">or drag and drop your bank statement here</p>
            <p className="text-[11px] text-muted mt-3">
              {fileName ?? 'No file selected'}
            </p>
          </button>
        </div>

        <div>
          <label className="block text-[12px] text-muted mb-1">Bank</label>
          <select className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-[14px] text-ink outline-none focus:border-accent">
            <option value="fnb">First National Bank (FNB)</option>
          </select>
        </div>

        {error && (
          <div className="bg-red-bg border border-red/20 rounded-lg px-4 py-3 text-[13px] text-red">
            {error}
          </div>
        )}

        <div className="flex justify-end gap-3 pt-2">
          <button
            onClick={onClose}
            className="text-[13px] text-muted hover:text-ink transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleUpload}
            disabled={uploading}
            className="text-[13px] font-semibold bg-ink text-surface px-5 py-2 rounded-lg hover:bg-ink/80 transition-colors disabled:opacity-50"
          >
            {uploading ? 'Uploading…' : 'Upload & process'}
          </button>
        </div>
      </div>
    )
  }

  function renderReviewStep() {
    const rows = importData?.rows ?? []

    return (
      <div className="space-y-4">
        <div className="flex items-center gap-4 text-[13px]">
          <div className="bg-green-bg text-green px-2 py-[2px] rounded text-[12px] font-semibold">
            {importData?.matched_rows ?? 0} matched
          </div>
          <div className="bg-yellowbg text-amber px-2 py-[2px] rounded text-[12px] font-semibold">
            {importData?.ambiguous_rows ?? 0} review
          </div>
          <div className="bg-red-bg text-red px-2 py-[2px] rounded text-[12px] font-semibold">
            {importData?.unmatched_rows ?? 0} unmatched
          </div>
        </div>

        <div className="max-h-64 overflow-y-auto border border-border rounded-lg">
          {rows.map(row => (
            <div
              key={row.id}
              className="flex items-center justify-between px-4 py-3 border-b border-border last:border-b-0 text-[13px]"
            >
              <div className="flex-1 min-w-0">
                <p className="font-medium text-ink truncate">
                  {row.description || row.reference || '—'}
                </p>
                <p className="text-[12px] text-muted">
                  {row.transaction_date} · {formatRand(row.amount_cents)}
                  {row.unit_identifier && ` · Unit ${row.unit_identifier}`}
                </p>
              </div>
              <div className="flex items-center gap-3 flex-shrink-0">
                <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${ROW_STATUS_STYLES[row.status] ?? 'bg-accent-bg text-accent'}`}>
                  {row.status}
                </span>
                {(row.status === 'ambiguous' || row.status === 'unmatched') && (
                  <select
                    value={manualMatches.get(row.id) ?? row.matched_levy_account_id ?? ''}
                    onChange={event => setMatch(row.id, event.target.value)}
                    className="text-[12px] border border-border rounded px-2 py-1 bg-surface text-ink outline-none focus:border-accent"
                  >
                    <option value="">Select unit</option>
                    {levyAccounts.map(acc => (
                      <option key={acc.id} value={acc.id}>
                        Unit {acc.unit_identifier} — {acc.owner_name}
                      </option>
                    ))}
                  </select>
                )}
              </div>
            </div>
          ))}
          {rows.length === 0 && (
            <div className="px-5 py-8 text-center text-[13px] text-muted">
              {importData?.status === 'queued' || importData?.status === 'processing'
                ? 'Processing bank statement…'
                : 'No rows found in the bank statement.'}
            </div>
          )}
        </div>

        {error && (
          <div className="bg-red-bg border border-red/20 rounded-lg px-4 py-3 text-[13px] text-red">
            {error}
          </div>
        )}

        <div className="flex justify-end gap-3 pt-2">
          <button
            onClick={() => {
              setStep('upload')
              setManualMatches(new Map())
              setImportData(null)
            }}
            className="text-[13px] text-muted hover:text-ink transition-colors"
          >
            Back
          </button>
          <button
            onClick={handleApply}
            disabled={applying}
            className="text-[13px] font-semibold bg-ink text-surface px-5 py-2 rounded-lg hover:bg-ink/80 transition-colors disabled:opacity-50"
          >
            {applying ? 'Applying…' : importData?.status === 'applied' ? 'Done' : 'Apply payments'}
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-ink/20 backdrop-blur-sm">
      <div className="bg-surface border border-border rounded-xl shadow-xl w-full max-w-lg mx-4 p-6">
        <div className="flex items-center justify-between mb-5">
          <div>
            <h2 className="text-[16px] font-serif font-semibold text-ink">
              Import Bank Statement
            </h2>
            <p className="text-[12px] text-muted mt-1">{periodLabel}</p>
          </div>
          <button
            onClick={onClose}
            className="text-muted hover:text-ink transition-colors p-1"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
              <path d="M4 4l8 8M12 4l-8 8" strokeLinecap="round" />
            </svg>
          </button>
        </div>

        {importData && importData.status !== 'queued' ? renderReviewStep() : renderUploadStep()}
      </div>
    </div>
  )
}
