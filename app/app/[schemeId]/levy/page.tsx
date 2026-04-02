'use client'

import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'next/navigation'

import Modal from '@/components/Modal'
import ReconcileModal from '@/components/ReconcileModal'
import { useAuth } from '@/lib/auth'
import { createLevyPeriod, getLevyDashboard, reconcileLevyPayments } from '@/lib/levy-api'
import type { LevyAccountInfo, LevyDashboard, ReconcilePaymentInput } from '@/lib/levy'
import { useToast } from '@/lib/toast'

function formatRand(cents: number): string {
  return `R ${(cents / 100).toLocaleString('en-ZA', { minimumFractionDigits: 0 })}`
}

function formatShortDate(value: string): string {
  return new Date(value).toLocaleDateString('en-ZA', {
    day: 'numeric',
    month: 'short',
  })
}

const STATUS_STYLES: Record<string, string> = {
  paid: 'bg-green-bg text-green',
  partial: 'bg-yellowbg text-amber',
  overdue: 'bg-red-bg text-red',
  pending: 'bg-accent-bg text-accent',
}

const EMPTY_PERIOD_FORM = {
  label: '',
  due_date: '',
  amount: '',
}

export default function LevyPaymentsPage() {
  const { user } = useAuth()
  const { addToast } = useToast()
  const params = useParams()
  const schemeId = params.schemeId as string

  const [dashboard, setDashboard] = useState<LevyDashboard | null>(null)
  const [loading, setLoading] = useState(true)
  const [reconcileOpen, setReconcileOpen] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [creatingPeriod, setCreatingPeriod] = useState(false)
  const [reconciling, setReconciling] = useState(false)
  const [periodForm, setPeriodForm] = useState(EMPTY_PERIOD_FORM)

  const isResident = user?.role === 'resident'
  const canEdit = user?.role === 'admin'

  useEffect(() => {
    async function load() {
      try {
        setLoading(true)
        setDashboard(await getLevyDashboard(schemeId))
      } catch (error) {
        addToast(
          error instanceof Error ? error.message : 'Failed to load levy dashboard',
          'error',
        )
      } finally {
        setLoading(false)
      }
    }

    load()
  }, [addToast, schemeId])

  async function refreshDashboard() {
    try {
      setLoading(true)
      setDashboard(await getLevyDashboard(schemeId))
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to load levy dashboard',
        'error',
      )
    } finally {
      setLoading(false)
    }
  }

  async function handleCreatePeriod() {
    const amountCents = Math.round(Number(periodForm.amount) * 100)
    if (!periodForm.label.trim() || !periodForm.due_date || !Number.isFinite(amountCents) || amountCents <= 0) {
      addToast('Enter a label, due date, and valid levy amount', 'error')
      return
    }

    setCreatingPeriod(true)
    try {
      await createLevyPeriod(schemeId, {
        label: periodForm.label.trim(),
        due_date: periodForm.due_date,
        amount_cents: amountCents,
      })
      setCreateOpen(false)
      setPeriodForm(EMPTY_PERIOD_FORM)
      await refreshDashboard()
      addToast('Levy period created and levy roll generated', 'success')
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to create levy period',
        'error',
      )
    } finally {
      setCreatingPeriod(false)
    }
  }

  async function handleReconcileConfirm(payments: ReconcilePaymentInput[]) {
    if (payments.length === 0) return
    setReconciling(true)
    try {
      const result = await reconcileLevyPayments(schemeId, payments)
      setReconcileOpen(false)
      await refreshDashboard()
      addToast(
        `Reconciliation applied: ${result.applied_count} payment${result.applied_count !== 1 ? 's' : ''} recorded.`,
        'success',
      )
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to reconcile payments',
        'error',
      )
    } finally {
      setReconciling(false)
    }
  }

  const levyRoll = dashboard?.levy_roll ?? []
  const currentPeriod = dashboard?.current_period ?? null
  const myAccount = dashboard?.my_account ?? null
  const myPayments = dashboard?.my_payments ?? []
  const overdue = dashboard?.overdue_count ?? 0
  const totalCollected = dashboard?.total_collected_cents ?? 0
  const latestPct = dashboard?.collection_rate_pct ?? 0

  const myMembership = useMemo(
    () => user?.scheme_memberships.find(membership => membership.scheme_id === schemeId) ?? null,
    [schemeId, user?.scheme_memberships],
  )

  if (loading) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Loading levy dashboard…
        </div>
      </div>
    )
  }

  if (!dashboard) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <div className="bg-hover-subtle border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Levy information could not be loaded.
        </div>
      </div>
    )
  }

  if (isResident) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › My Levy</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">My Levy</h1>
        <p className="text-[14px] text-muted mb-8">
          Levy account for {myMembership?.unit_identifier ? `Unit ${myMembership.unit_identifier}` : 'your unit'}.
        </p>

        {!currentPeriod || !myAccount ? (
          <div className="bg-surface border border-border rounded-lg px-6 py-10 text-center text-muted text-[13px]">
            No levy roll has been generated for your unit yet.
          </div>
        ) : (
          <div className="bg-surface border border-border rounded-lg px-6 py-5 mb-6 flex items-center justify-between">
            <div>
              <p className="text-[12px] text-muted mb-1">
                {currentPeriod.label} · due {formatShortDate(currentPeriod.due_date)}
              </p>
              <p className="font-serif text-[32px] font-semibold text-ink leading-none">
                {formatRand(currentPeriod.amount_cents)}
              </p>
            </div>
            <span className={`text-[12px] font-semibold px-3 py-1 rounded-full ${STATUS_STYLES[myAccount.status]}`}>
              {myAccount.status.charAt(0).toUpperCase() + myAccount.status.slice(1)}
            </span>
          </div>
        )}

        <h2 className="text-[14px] font-semibold text-ink mb-3">Payment history</h2>
        {myPayments.length === 0 ? (
          <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-8 text-center text-muted text-[13px] mb-6">
            No payment history available for this unit.
          </div>
        ) : (
          <div className="bg-surface border border-border rounded-lg overflow-hidden mb-6">
            <div className="overflow-x-auto">
              {myPayments.map((payment, index) => (
                <div key={payment.id} className={`flex items-center justify-between px-5 py-3 text-[13px] min-w-[400px] ${index < myPayments.length - 1 ? 'border-b border-border' : ''}`}>
                  <div>
                    <span className="font-medium text-ink">{formatRand(payment.amount_cents)}</span>
                    <span className="text-muted ml-3">{new Date(payment.payment_date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' })}</span>
                  </div>
                  <span className="text-[11px] text-muted font-mono flex-shrink-0">{payment.reference}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        <button onClick={() => addToast('Statement downloads will ship with the documents slice.', 'info')} className="text-[12px] text-accent font-medium border border-accent rounded px-4 py-2 hover:bg-accent-dim transition-colors">
          Download statement (PDF)
        </button>
      </div>
    )
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <div className="flex items-start justify-between mb-8 gap-3">
        <div>
          <p className="text-[12px] text-muted mb-1">Scheme › Levy & Payments</p>
          <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Levy & Payments</h1>
          <p className="text-[14px] text-muted">Levy periods, collection metrics, and bank reconciliation.</p>
        </div>
        {canEdit && (
          <div className="flex flex-col sm:flex-row gap-2">
            <button
              onClick={() => setCreateOpen(true)}
              className="flex-shrink-0 text-[13px] font-semibold border border-border px-4 py-2 rounded-lg hover:bg-page transition-colors"
            >
              New levy period
            </button>
            <button
              onClick={() => setReconcileOpen(true)}
              disabled={!currentPeriod || levyRoll.length === 0 || reconciling}
              className="flex-shrink-0 flex items-center gap-2 text-[13px] font-semibold bg-ink text-surface px-4 py-2 rounded-lg hover:bg-ink/80 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
                <path d="M1 4h12M1 8h8M1 12h5" strokeLinecap="round" />
                <path d="M10 9l2 2 2-2" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
              {reconciling ? 'Reconciling…' : 'Reconcile statement'}
            </button>
          </div>
        )}
      </div>

      {!currentPeriod ? (
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          No levy period has been created for this scheme yet.
        </div>
      ) : (
        <>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
            {[
              { label: 'Collection rate', value: `${latestPct}%` },
              { label: 'Total collected', value: formatRand(totalCollected) },
              { label: 'Overdue', value: String(overdue) },
              { label: 'Monthly levy', value: formatRand(currentPeriod.amount_cents) },
            ].map(({ label, value }) => (
              <div key={label} className="bg-surface border border-border rounded-lg px-5 py-4">
                <div className="text-[24px] font-semibold text-ink font-serif mb-1">{value}</div>
                <div className="text-[12px] text-muted">{label}</div>
              </div>
            ))}
          </div>

          <div className="bg-surface border border-border rounded-lg px-6 py-5 mb-6">
            <p className="text-[11px] font-semibold text-muted uppercase tracking-[0.08em] mb-4">Collection rate</p>
            {dashboard.collection_trend.length === 0 ? (
              <div className="text-[13px] text-muted">No levy history yet.</div>
            ) : (
              <>
                <div className="flex items-end gap-2 h-[56px] mb-1">
                  {dashboard.collection_trend.map((point, index) => {
                    const isLast = index === dashboard.collection_trend.length - 1
                    const heightPct = point.pct
                    return (
                      <div key={point.label} className="flex-1 h-full flex flex-col items-center justify-end gap-1">
                        <span className="text-[9px] font-semibold leading-none" style={{ color: isLast ? '#2B6CB0' : '#A8A49E' }}>{point.pct}%</span>
                        <div className="w-full rounded-[2px]" style={{ height: `${Math.max(heightPct, 8)}%`, background: isLast ? '#2B6CB0' : '#E3E2DF' }} />
                      </div>
                    )
                  })}
                </div>
                <div className="flex">
                  {dashboard.collection_trend.map(point => (
                    <span key={point.label} className="flex-1 text-center text-[9px] text-muted">{point.label}</span>
                  ))}
                </div>
              </>
            )}
          </div>

          <div className="bg-surface border border-border rounded-lg overflow-hidden">
            <div className="px-5 py-3 border-b border-border flex items-center justify-between">
              <span className="text-[13px] font-semibold text-ink">Levy Roll — {currentPeriod.label}</span>
              <span className="text-[11px] font-semibold px-2 py-[2px] rounded-full bg-accent-bg text-accent">{levyRoll.length} shown</span>
            </div>
            <div className="overflow-x-auto">
              <div className="px-5 min-w-[560px]">
                <div className="grid grid-cols-[1fr_auto_auto_auto] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
                  <span>Unit</span><span>Amount</span><span>Due</span><span>Status</span>
                </div>
                {levyRoll.map((account: LevyAccountInfo) => (
                  <div key={account.id} className="grid grid-cols-[1fr_auto_auto_auto] gap-4 items-center py-3 border-b border-border last:border-b-0 text-[13px]">
                    <div>
                      <div className="font-semibold text-ink">Unit {account.unit_identifier}</div>
                      <div className="text-[12px] text-muted">{account.owner_name}</div>
                    </div>
                    <span className="font-semibold text-ink tabular-nums">{formatRand(account.amount_cents)}</span>
                    <span className="text-[12px] text-muted">{formatShortDate(account.due_date)}</span>
                    <div className="flex items-center gap-2">
                      <span className={`text-[11px] font-semibold px-[10px] py-[3px] rounded-full ${STATUS_STYLES[account.status]}`}>
                        {account.status.charAt(0).toUpperCase() + account.status.slice(1)}
                      </span>
                      {account.status === 'overdue' && (
                        <button onClick={() => addToast(`Reminder delivery will be wired with communications. Unit ${account.unit_identifier} flagged for follow-up.`, 'info')} className="text-[11px] text-accent font-medium hover:underline">
                          Remind
                        </button>
                      )}
                    </div>
                  </div>
                ))}
                {levyRoll.length === 0 && (
                  <div className="px-5 py-8 text-center text-[13px] text-muted">
                    No levy accounts have been generated for this period yet.
                  </div>
                )}
              </div>
            </div>
          </div>
        </>
      )}

      <Modal open={createOpen} onClose={() => !creatingPeriod && setCreateOpen(false)} title="New levy period">
        <div className="space-y-4">
          <div>
            <label className="block text-[12px] text-muted mb-1">Period label</label>
            <input
              value={periodForm.label}
              onChange={event => setPeriodForm(current => ({ ...current, label: event.target.value }))}
              placeholder="April 2026"
              className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-[14px] text-ink outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="block text-[12px] text-muted mb-1">Due date</label>
            <input
              type="date"
              value={periodForm.due_date}
              onChange={event => setPeriodForm(current => ({ ...current, due_date: event.target.value }))}
              className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-[14px] text-ink outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="block text-[12px] text-muted mb-1">Monthly levy amount (R)</label>
            <input
              type="number"
              min="0"
              step="0.01"
              value={periodForm.amount}
              onChange={event => setPeriodForm(current => ({ ...current, amount: event.target.value }))}
              placeholder="2450.00"
              className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-[14px] text-ink outline-none focus:border-accent"
            />
          </div>
          <div className="flex justify-end gap-3 pt-2">
            <button
              onClick={() => setCreateOpen(false)}
              className="text-[13px] text-muted hover:text-ink transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleCreatePeriod}
              disabled={creatingPeriod}
              className="text-[13px] font-semibold bg-ink text-surface px-5 py-2 rounded-lg hover:bg-ink/80 transition-colors disabled:opacity-50"
            >
              {creatingPeriod ? 'Creating…' : 'Create period'}
            </button>
          </div>
        </div>
      </Modal>

      {reconcileOpen && currentPeriod && (
        <ReconcileModal
          levyAccounts={levyRoll}
          periodLabel={currentPeriod.label}
          onConfirm={handleReconcileConfirm}
          onClose={() => !reconciling && setReconcileOpen(false)}
        />
      )}
    </div>
  )
}
