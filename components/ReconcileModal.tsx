'use client'
import { useState, useRef, useCallback } from 'react'
import { parseBankStatementCSV, matchTransactions, summariseMatches } from '@/lib/reconcile'
import { SAMPLE_BANK_STATEMENT_CSV } from '@/lib/mock/bank-statement'
import type { LevyAccount } from '@/lib/mock/levy'
import type { ReconcileMatch } from '@/lib/reconcile'

interface ReconcileModalProps {
  levyAccounts: LevyAccount[]
  periodLabel: string
  onConfirm: (updates: Array<{ id: string; paid_cents: number; status: LevyAccount['status'] }>) => void
  onClose: () => void
}

type Step = 'upload' | 'review' | 'confirm'

const CONFIDENCE_STYLES: Record<string, string> = {
  high:      'bg-green-bg text-green',
  medium:    'bg-yellowbg text-amber',
  low:       'bg-accent-bg text-accent',
  unmatched: 'bg-red-bg text-red',
}

const CONFIDENCE_LABELS: Record<string, string> = {
  high: 'Auto', medium: 'Review', low: 'Weak', unmatched: 'No match',
}

function formatRand(cents: number) {
  return `R ${(cents / 100).toLocaleString('en-ZA', { minimumFractionDigits: 0 })}`
}

export default function ReconcileModal({
  levyAccounts,
  periodLabel,
  onConfirm,
  onClose,
}: ReconcileModalProps) {
  const [step, setStep] = useState<Step>('upload')
  const [matches, setMatches] = useState<ReconcileMatch[]>([])
  const [error, setError] = useState<string | null>(null)
  const [fileName, setFileName] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const processCSV = useCallback(
    (csv: string, name: string) => {
      setError(null)
      try {
        const transactions = parseBankStatementCSV(csv)
        if (transactions.length === 0) {
          setError(
            'No valid credit transactions found in this file. Make sure it is a bank statement CSV with a Date, Description, and Amount column.'
          )
          return
        }
        const result = matchTransactions(transactions, levyAccounts)
        setMatches(result)
        setFileName(name)
        setStep('review')
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Could not parse this file.')
      }
    },
    [levyAccounts]
  )

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = ev => processCSV(ev.target?.result as string, file.name)
    reader.readAsText(file)
  }

  function handleLoadSample() {
    processCSV(SAMPLE_BANK_STATEMENT_CSV, 'sample-statement.csv')
  }

  function handleConfirm() {
    const updates = matches
      .filter(m => m.account !== null)
      .map(m => ({ id: m.account!.id, paid_cents: m.new_paid_cents, status: m.new_status }))
    onConfirm(updates)
  }

  const summary = matches.length > 0 ? summariseMatches(matches) : null
  const matchedCount = matches.filter(m => m.account !== null).length

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-ink/50 backdrop-blur-[2px]">
      <div className="bg-surface border border-border rounded-xl shadow-2xl w-full max-w-[720px] max-h-[90vh] flex flex-col">

        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border flex-shrink-0">
          <div>
            <h2 className="font-serif text-[18px] font-semibold text-ink">Reconcile bank statement</h2>
            <p className="text-[12px] text-muted mt-0.5">{periodLabel}</p>
          </div>
          <div className="flex items-center gap-3">
            {/* Step indicators */}
            {(['upload', 'review', 'confirm'] as Step[]).map((s, i) => (
              <div key={s} className="flex items-center gap-2">
                {i > 0 && <div className="w-6 h-px bg-border" />}
                <div
                  className={`flex items-center gap-1.5 text-[12px] font-medium ${step === s ? 'text-ink' : 'text-muted'}`}
                >
                  <div
                    className={`w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-bold ${step === s ? 'bg-ink text-surface' : 'bg-border text-muted'}`}
                  >
                    {i + 1}
                  </div>
                  <span className="hidden sm:inline capitalize">{s}</span>
                </div>
              </div>
            ))}
            <button
              onClick={onClose}
              className="ml-2 text-muted hover:text-ink transition-colors p-1 rounded"
            >
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
                <line x1="3" y1="3" x2="13" y2="13" />
                <line x1="13" y1="3" x2="3" y2="13" />
              </svg>
            </button>
          </div>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-5">

          {/* ── Step 1: Upload ── */}
          {step === 'upload' && (
            <div>
              <p className="text-[14px] text-muted mb-6">
                Upload your bank statement CSV. StrataHQ will automatically match deposits to levy
                accounts using unit identifiers, owner names, and amounts.
              </p>

              <div
                className="border-2 border-dashed border-border rounded-lg px-8 py-12 text-center cursor-pointer hover:border-accent hover:bg-accent-dim transition-colors"
                onClick={() => fileInputRef.current?.click()}
              >
                <svg
                  className="w-8 h-8 text-muted mx-auto mb-3"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth="1.5"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5"
                  />
                </svg>
                <p className="text-[14px] font-medium text-ink mb-1">Drop CSV file here</p>
                <p className="text-[12px] text-muted">
                  Standard Bank, FNB, Nedbank, ABSA, Capitec formats supported
                </p>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".csv,.txt"
                  className="hidden"
                  onChange={handleFileChange}
                />
              </div>

              {error && (
                <div className="mt-4 px-4 py-3 bg-red-bg border border-red rounded-lg text-[13px] text-red">
                  {error}
                </div>
              )}

              <div className="flex items-center gap-3 mt-4">
                <div className="flex-1 h-px bg-border" />
                <span className="text-[12px] text-muted">or</span>
                <div className="flex-1 h-px bg-border" />
              </div>

              <button
                onClick={handleLoadSample}
                className="mt-4 w-full text-[13px] text-accent font-medium border border-accent rounded-lg px-4 py-3 hover:bg-accent-dim transition-colors"
              >
                Load sample statement (demo)
              </button>
            </div>
          )}

          {/* ── Step 2: Review ── */}
          {step === 'review' && (
            <div>
              <div className="flex items-center justify-between mb-4">
                <p className="text-[13px] text-muted">
                  {fileName} · {matches.length} transaction{matches.length !== 1 ? 's' : ''} ·{' '}
                  {matchedCount} matched
                </p>
                {summary && (
                  <div className="flex gap-2">
                    <span className={`text-[11px] font-semibold px-2 py-1 rounded-full ${CONFIDENCE_STYLES.high}`}>
                      {summary.matched_high} auto
                    </span>
                    {summary.matched_medium > 0 && (
                      <span className={`text-[11px] font-semibold px-2 py-1 rounded-full ${CONFIDENCE_STYLES.medium}`}>
                        {summary.matched_medium} review
                      </span>
                    )}
                    {summary.unmatched > 0 && (
                      <span className={`text-[11px] font-semibold px-2 py-1 rounded-full ${CONFIDENCE_STYLES.unmatched}`}>
                        {summary.unmatched} unmatched
                      </span>
                    )}
                  </div>
                )}
              </div>

              <div className="border border-border rounded-lg overflow-hidden">
                <div className="grid grid-cols-[1fr_90px_120px_80px] gap-3 px-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border bg-page">
                  <span>Transaction</span>
                  <span>Amount</span>
                  <span>Matched unit</span>
                  <span>Confidence</span>
                </div>
                {matches.map((m, i) => (
                  <div
                    key={i}
                    className={`grid grid-cols-[1fr_90px_120px_80px] gap-3 items-center px-4 py-3 text-[13px] ${i < matches.length - 1 ? 'border-b border-border' : ''} ${m.confidence === 'unmatched' ? 'opacity-50' : ''}`}
                  >
                    <div>
                      <div className="font-medium text-ink truncate max-w-[220px]">
                        {m.transaction.description}
                      </div>
                      <div className="text-[11px] text-muted">{m.transaction.date}</div>
                    </div>
                    <span className="font-semibold text-ink tabular-nums">
                      {formatRand(m.transaction.amount_cents)}
                    </span>
                    <div>
                      {m.account ? (
                        <>
                          <div className="font-medium text-ink">Unit {m.account.unit_identifier}</div>
                          <div className="text-[11px] text-muted truncate">{m.account.owner_name}</div>
                        </>
                      ) : (
                        <span className="text-muted text-[12px]">—</span>
                      )}
                    </div>
                    <span
                      className={`text-[11px] font-semibold px-2 py-1 rounded-full text-center inline-block ${CONFIDENCE_STYLES[m.confidence]}`}
                    >
                      {CONFIDENCE_LABELS[m.confidence]}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* ── Step 3: Confirm ── */}
          {step === 'confirm' && summary && (
            <div>
              <div className="bg-green-bg border border-green rounded-lg px-5 py-4 mb-6">
                <p className="text-[14px] font-semibold text-green mb-1">
                  Ready to apply reconciliation
                </p>
                <p className="text-[13px] text-green/80">
                  {matchedCount} payment{matchedCount !== 1 ? 's' : ''} totalling{' '}
                  {formatRand(summary.matched_amount_cents)} will be recorded against their levy
                  accounts.
                  {summary.unmatched > 0 &&
                    ` ${summary.unmatched} transaction${summary.unmatched !== 1 ? 's' : ''} will be skipped.`}
                </p>
              </div>

              <div className="border border-border rounded-lg overflow-hidden">
                <div className="px-4 py-3 border-b border-border bg-page text-[11px] font-semibold text-muted uppercase tracking-wide">
                  Changes to be applied
                </div>
                {matches
                  .filter(m => m.account !== null)
                  .map((m, i, arr) => (
                    <div
                      key={i}
                      className={`flex items-center justify-between px-4 py-3 text-[13px] ${i < arr.length - 1 ? 'border-b border-border' : ''}`}
                    >
                      <div>
                        <span className="font-medium text-ink">Unit {m.account!.unit_identifier}</span>
                        <span className="text-muted ml-2">{m.account!.owner_name}</span>
                      </div>
                      <div className="flex items-center gap-3">
                        <span className="text-muted text-[12px] capitalize line-through">
                          {m.account!.status}
                        </span>
                        <svg
                          width="12"
                          height="12"
                          viewBox="0 0 12 12"
                          fill="none"
                          stroke="currentColor"
                          className="text-muted flex-shrink-0"
                        >
                          <path d="M2 6h8M7 3l3 3-3 3" strokeWidth="1.5" strokeLinecap="round" />
                        </svg>
                        <span
                          className={`text-[11px] font-semibold px-2 py-0.5 rounded-full capitalize ${
                            m.new_status === 'paid'
                              ? 'bg-green-bg text-green'
                              : m.new_status === 'partial'
                                ? 'bg-yellowbg text-amber'
                                : 'bg-red-bg text-red'
                          }`}
                        >
                          {m.new_status}
                        </span>
                        <span className="font-semibold text-ink tabular-nums">
                          {formatRand(m.new_paid_cents)}
                        </span>
                      </div>
                    </div>
                  ))}
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-6 py-4 border-t border-border flex-shrink-0">
          <button
            onClick={() =>
              step === 'upload'
                ? onClose()
                : setStep(step === 'confirm' ? 'review' : 'upload')
            }
            className="text-[13px] text-muted hover:text-ink transition-colors"
          >
            {step === 'upload' ? 'Cancel' : '← Back'}
          </button>

          {step === 'review' && (
            <button
              onClick={() => setStep('confirm')}
              disabled={matchedCount === 0}
              className="text-[13px] font-semibold bg-ink text-surface px-5 py-2 rounded-lg hover:bg-ink/80 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Review {matchedCount} match{matchedCount !== 1 ? 'es' : ''} →
            </button>
          )}

          {step === 'confirm' && (
            <button
              onClick={handleConfirm}
              className="text-[13px] font-semibold bg-green text-white px-5 py-2 rounded-lg hover:bg-green/80 transition-colors"
            >
              Apply reconciliation
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
