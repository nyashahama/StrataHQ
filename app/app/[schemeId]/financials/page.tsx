'use client'

import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'next/navigation'

import Modal from '@/components/Modal'
import { useAuth } from '@/lib/auth'
import { getFinancialDashboard, updateReserveFund, upsertBudgetLine } from '@/lib/financials-api'
import type { BudgetLineInfo, FinancialDashboard } from '@/lib/financials'
import { useToast } from '@/lib/toast'

function formatRand(cents: number): string {
  return `R ${(cents / 100).toLocaleString('en-ZA', { minimumFractionDigits: 0 })}`
}

export default function FinancialsPage() {
  const { user } = useAuth()
  const { addToast } = useToast()
  const params = useParams()
  const schemeId = params.schemeId as string

  const [dashboard, setDashboard] = useState<FinancialDashboard | null>(null)
  const [loading, setLoading] = useState(true)
  const [selectedPeriod, setSelectedPeriod] = useState('')
  const [showBudgetModal, setShowBudgetModal] = useState(false)
  const [showReserveModal, setShowReserveModal] = useState(false)
  const [savingBudget, setSavingBudget] = useState(false)
  const [savingReserve, setSavingReserve] = useState(false)
  const [budgetForm, setBudgetForm] = useState({
    category: '',
    period_label: '',
    budget_amount: '',
    actual_amount: '',
  })
  const [reserveForm, setReserveForm] = useState({
    balance_amount: '',
    target_amount: '',
  })

  const canManage = user?.role === 'admin' || user?.role === 'trustee'

  useEffect(() => {
    async function load() {
      try {
        setLoading(true)
        const nextDashboard = await getFinancialDashboard(schemeId, selectedPeriod || undefined)
        setDashboard(nextDashboard)
        if (!selectedPeriod && nextDashboard.selected_period) {
          setSelectedPeriod(nextDashboard.selected_period)
        }
      } catch (error) {
        addToast(
          error instanceof Error ? error.message : 'Failed to load financial dashboard',
          'error',
        )
      } finally {
        setLoading(false)
      }
    }

    load()
  }, [addToast, schemeId, selectedPeriod])

  const budgetLines = dashboard?.budget_lines ?? []
  const reserveFund = dashboard?.reserve_fund ?? null
  const levySummary = dashboard?.levy_summary ?? null
  const totalBudgeted = dashboard?.total_budgeted_cents ?? 0
  const totalActual = dashboard?.total_actual_cents ?? 0
  const surplus = dashboard?.surplus_cents ?? 0
  const reservePct = reserveFund && reserveFund.target_cents > 0
    ? Math.round((reserveFund.balance_cents / reserveFund.target_cents) * 100)
    : 0

  const availablePeriods = dashboard?.available_periods ?? []

  const exportRows = useMemo(
    () => [
      ['Category', 'Budgeted', 'Actual', 'Variance'],
      ...budgetLines.map(line => [
        line.category,
        String(line.budgeted_cents / 100),
        String(line.actual_cents / 100),
        String(line.variance_cents / 100),
      ]),
    ],
    [budgetLines],
  )

  function downloadCsv() {
    const content = exportRows.map(row => row.map(value => `"${value.replaceAll('"', '""')}"`).join(',')).join('\n')
    const blob = new Blob([content], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `financials-${selectedPeriod || 'current'}.csv`
    link.click()
    URL.revokeObjectURL(url)
    addToast('Budget export downloaded', 'success')
  }

  async function refreshDashboard(nextPeriod?: string) {
    setLoading(true)
    try {
      const dashboardData = await getFinancialDashboard(schemeId, nextPeriod || selectedPeriod || undefined)
      setDashboard(dashboardData)
      if (dashboardData.selected_period && dashboardData.selected_period !== selectedPeriod) {
        setSelectedPeriod(dashboardData.selected_period)
      }
    } finally {
      setLoading(false)
    }
  }

  async function handleBudgetSave() {
    const budgetedCents = Math.round(Number(budgetForm.budget_amount) * 100)
    const actualCents = Math.round(Number(budgetForm.actual_amount) * 100)
    if (!budgetForm.category.trim() || !budgetForm.period_label.trim() || budgetedCents < 0 || actualCents < 0) {
      addToast('Enter a category, period, budget amount, and actual amount', 'error')
      return
    }

    setSavingBudget(true)
    try {
      await upsertBudgetLine(schemeId, {
        category: budgetForm.category.trim(),
        period_label: budgetForm.period_label.trim(),
        budgeted_cents: budgetedCents,
        actual_cents: actualCents,
      })
      setShowBudgetModal(false)
      setBudgetForm({ category: '', period_label: selectedPeriod || '', budget_amount: '', actual_amount: '' })
      await refreshDashboard(budgetForm.period_label.trim())
      addToast('Budget line saved', 'success')
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to save budget line',
        'error',
      )
    } finally {
      setSavingBudget(false)
    }
  }

  async function handleReserveSave() {
    const balanceCents = Math.round(Number(reserveForm.balance_amount) * 100)
    const targetCents = Math.round(Number(reserveForm.target_amount) * 100)
    if (balanceCents < 0 || targetCents < 0) {
      addToast('Enter valid reserve fund amounts', 'error')
      return
    }

    setSavingReserve(true)
    try {
      await updateReserveFund(schemeId, {
        balance_cents: balanceCents,
        target_cents: targetCents,
      })
      setShowReserveModal(false)
      await refreshDashboard()
      addToast('Reserve fund updated', 'success')
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to update reserve fund',
        'error',
      )
    } finally {
      setSavingReserve(false)
    }
  }

  function openBudgetModal(line?: BudgetLineInfo) {
    setBudgetForm({
      category: line?.category ?? '',
      period_label: line?.period_label ?? selectedPeriod ?? '',
      budget_amount: line ? String(line.budgeted_cents / 100) : '',
      actual_amount: line ? String(line.actual_cents / 100) : '',
    })
    setShowBudgetModal(true)
  }

  function openReserveModal() {
    setReserveForm({
      balance_amount: reserveFund ? String(reserveFund.balance_cents / 100) : '',
      target_amount: reserveFund ? String(reserveFund.target_cents / 100) : '',
    })
    setShowReserveModal(true)
  }

  if (loading) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Loading financial dashboard…
        </div>
      </div>
    )
  }

  // Resident: simplified summary only
  if (user?.role === 'resident') {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › Financials</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Financials</h1>
        <p className="text-[14px] text-muted mb-8">Scheme financial health summary.</p>

        {/* Reserve fund */}
        <div className="bg-surface border border-border rounded-lg px-6 py-5 mb-4">
          <div className="flex items-center justify-between mb-3">
            <span className="text-[13px] font-semibold text-ink">Reserve Fund</span>
            <span className="text-[12px] text-muted">{reservePct}% of target</span>
          </div>
          <div className="h-3 bg-border rounded-full overflow-hidden mb-2">
            <div className="h-full bg-accent rounded-full" style={{ width: `${reservePct}%` }} />
          </div>
          <div className="flex justify-between text-[12px] text-muted">
            <span>Balance: <strong className="text-ink">{formatRand(reserveFund?.balance_cents ?? 0)}</strong></span>
            <span>Target: {formatRand(reserveFund?.target_cents ?? 0)}</span>
          </div>
        </div>

        <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-5 text-[13px] text-ink leading-relaxed">
          The scheme's total approved budget for {dashboard?.selected_period || 'the current period'} is <strong>{formatRand(totalBudgeted)}</strong>.
          Expenditure to date is <strong>{formatRand(totalActual)}</strong>.
          {levySummary ? <> Levy collection is currently <strong>{levySummary.collection_rate_pct}%</strong> for {levySummary.period_label}. </> : ' '}
          The reserve fund stands at {reservePct}% of the 10-year maintenance plan target.
        </div>
      </div>
    )
  }

  // Agent / Trustee: full financial view
  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Financials</p>
      <div className="flex flex-wrap items-center justify-between gap-3 mb-1">
        <h1 className="font-serif text-[28px] font-semibold text-ink">Financials</h1>
        <button
          onClick={downloadCsv}
          className="text-[12px] font-semibold border border-border bg-surface text-ink px-3 py-2 rounded hover:bg-hover-subtle transition-colors"
        >
          Export CSV
        </button>
      </div>
      <p className="text-[14px] text-muted mb-8">Budget, expenditure, and reserve fund.</p>

      <div className="flex flex-wrap items-center justify-between gap-3 mb-6">
        <div className="flex items-center gap-2">
          <label className="text-[12px] font-semibold text-muted">Period</label>
          <select
            value={selectedPeriod}
            onChange={event => setSelectedPeriod(event.target.value)}
            className="border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
          >
            {availablePeriods.length === 0 ? <option value="">Current</option> : availablePeriods.map(period => (
              <option key={period} value={period}>{period}</option>
            ))}
          </select>
        </div>
        {canManage && (
          <div className="flex gap-2">
            <button
              onClick={openReserveModal}
              className="text-[12px] font-semibold border border-border bg-surface text-ink px-3 py-2 rounded hover:bg-hover-subtle transition-colors"
            >
              Update reserve
            </button>
            <button
              onClick={() => openBudgetModal()}
              className="text-[12px] font-semibold bg-accent text-white px-3 py-2 rounded hover:opacity-90 transition-colors"
            >
              + Budget line
            </button>
          </div>
        )}
      </div>

      {/* Stat cards */}
      <div className="grid grid-cols-2 md:grid-cols-5 gap-4 mb-8">
        {[
          { label: 'Total budget',  value: formatRand(totalBudgeted) },
          { label: 'Spent to date', value: formatRand(totalActual) },
          { label: 'Reserve fund',  value: formatRand(reserveFund?.balance_cents ?? 0) },
          { label: surplus >= 0 ? 'Surplus' : 'Deficit', value: formatRand(Math.abs(surplus)) },
          { label: 'Levy collected', value: levySummary ? `${levySummary.collection_rate_pct}%` : 'No roll' },
        ].map(({ label, value }) => (
          <div key={label} className="bg-surface border border-border rounded-lg px-5 py-4">
            <div className="text-[22px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      {/* Reserve fund bar */}
      <div className="bg-surface border border-border rounded-lg px-6 py-5 mb-6">
        <div className="flex items-center justify-between mb-3">
          <span className="text-[13px] font-semibold text-ink">Reserve Fund — {reservePct}% of target</span>
          <span className="text-[12px] text-muted">Target: {formatRand(reserveFund?.target_cents ?? 0)}</span>
        </div>
        <div className="h-3 bg-border rounded-full overflow-hidden">
          <div className="h-full bg-accent rounded-full" style={{ width: `${reservePct}%` }} />
        </div>
      </div>

      {levySummary && (
        <div className="bg-surface border border-border rounded-lg px-6 py-5 mb-6">
          <div className="flex items-center justify-between mb-3">
            <span className="text-[13px] font-semibold text-ink">Levy collection — {levySummary.period_label}</span>
            <span className="text-[12px] text-muted">{levySummary.overdue_count} overdue account{levySummary.overdue_count !== 1 ? 's' : ''}</span>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 text-[13px]">
            <div className="bg-page/50 border border-border rounded-lg px-4 py-3">
              <div className="text-muted text-[11px] mb-1">Billed</div>
              <div className="font-semibold text-ink">{formatRand(levySummary.total_billed_cents)}</div>
            </div>
            <div className="bg-page/50 border border-border rounded-lg px-4 py-3">
              <div className="text-muted text-[11px] mb-1">Collected</div>
              <div className="font-semibold text-ink">{formatRand(levySummary.total_collected_cents)}</div>
            </div>
            <div className="bg-page/50 border border-border rounded-lg px-4 py-3">
              <div className="text-muted text-[11px] mb-1">Collection rate</div>
              <div className="font-semibold text-ink">{levySummary.collection_rate_pct}%</div>
            </div>
          </div>
        </div>
      )}

      {/* Budget vs actual table */}
      <div className="bg-surface border border-border rounded-lg">
        <div className="px-5 py-3 border-b border-border flex items-center justify-between">
          <span className="text-[13px] font-semibold text-ink">Budget vs Actual — {dashboard?.selected_period || 'Current period'}</span>
        </div>
        <div className="overflow-x-auto">
          <div className="px-5 min-w-[420px]">
              <div className="grid grid-cols-[1fr_auto_auto_auto] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
                <span>Category</span><span>Budgeted</span><span>Actual</span><span>Variance</span>
              </div>
              {budgetLines.map((line, i) => {
                const variance = line.budgeted_cents - line.actual_cents
                const over = variance < 0
                return (
                  <div key={line.id} className={`grid grid-cols-[1fr_auto_auto_auto_auto] gap-4 items-center py-3 text-[13px] ${i < budgetLines.length - 1 ? 'border-b border-border' : ''}`}>
                    <span className="text-ink">{line.category}</span>
                    <span className="tabular-nums text-muted">{formatRand(line.budgeted_cents)}</span>
                    <span className="tabular-nums text-ink font-medium">{formatRand(line.actual_cents)}</span>
                    <span className={`tabular-nums text-[12px] font-semibold ${over ? 'text-red' : 'text-green'}`}>
                      {over ? '+' : '-'}{formatRand(Math.abs(variance))}
                    </span>
                    {canManage && (
                      <button
                        onClick={() => openBudgetModal(line)}
                        className="text-[11px] text-accent font-medium hover:underline"
                      >
                        Edit
                      </button>
                    )}
                  </div>
                )
              })}
              {budgetLines.length === 0 && (
                <div className="py-10 text-center text-[13px] text-muted">
                  No budget lines have been captured for this period yet.
                </div>
              )}
          </div>
        </div>
      </div>

      <Modal open={showBudgetModal} onClose={() => !savingBudget && setShowBudgetModal(false)} title="Budget line">
        <div className="flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Category</label>
            <input
              type="text"
              value={budgetForm.category}
              onChange={event => setBudgetForm(current => ({ ...current, category: event.target.value }))}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Period label</label>
            <input
              type="text"
              value={budgetForm.period_label}
              onChange={event => setBudgetForm(current => ({ ...current, period_label: event.target.value }))}
              placeholder="e.g. 2026"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Budget amount (R)</label>
              <input
                type="number"
                min="0"
                step="0.01"
                value={budgetForm.budget_amount}
                onChange={event => setBudgetForm(current => ({ ...current, budget_amount: event.target.value }))}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
              />
            </div>
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Actual amount (R)</label>
              <input
                type="number"
                min="0"
                step="0.01"
                value={budgetForm.actual_amount}
                onChange={event => setBudgetForm(current => ({ ...current, actual_amount: event.target.value }))}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
              />
            </div>
          </div>
          <div className="flex justify-end gap-3 pt-1">
            <button onClick={() => setShowBudgetModal(false)} className="text-[12px] text-muted hover:text-ink px-3 py-2">
              Cancel
            </button>
            <button
              onClick={handleBudgetSave}
              disabled={savingBudget}
              className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {savingBudget ? 'Saving…' : 'Save'}
            </button>
          </div>
        </div>
      </Modal>

      <Modal open={showReserveModal} onClose={() => !savingReserve && setShowReserveModal(false)} title="Reserve fund">
        <div className="flex flex-col gap-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Balance (R)</label>
              <input
                type="number"
                min="0"
                step="0.01"
                value={reserveForm.balance_amount}
                onChange={event => setReserveForm(current => ({ ...current, balance_amount: event.target.value }))}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
              />
            </div>
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Target (R)</label>
              <input
                type="number"
                min="0"
                step="0.01"
                value={reserveForm.target_amount}
                onChange={event => setReserveForm(current => ({ ...current, target_amount: event.target.value }))}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
              />
            </div>
          </div>
          <div className="flex justify-end gap-3 pt-1">
            <button onClick={() => setShowReserveModal(false)} className="text-[12px] text-muted hover:text-ink px-3 py-2">
              Cancel
            </button>
            <button
              onClick={handleReserveSave}
              disabled={savingReserve}
              className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {savingReserve ? 'Saving…' : 'Save'}
            </button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
