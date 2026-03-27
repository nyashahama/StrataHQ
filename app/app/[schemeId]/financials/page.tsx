'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { mockBudgetLines, mockReserveFund } from '@/lib/mock/financials'

function formatRand(cents: number): string {
  return `R ${(cents / 100).toLocaleString('en-ZA', { minimumFractionDigits: 0 })}`
}

export default function FinancialsPage() {
  const { user } = useMockAuth()

  const totalBudgeted = mockBudgetLines.reduce((s, l) => s + l.budgeted_cents, 0)
  const totalActual   = mockBudgetLines.reduce((s, l) => s + l.actual_cents, 0)
  const surplus = totalBudgeted - totalActual
  const reservePct = Math.round((mockReserveFund.balance_cents / mockReserveFund.target_cents) * 100)

  // Resident: simplified summary only
  if (user?.role === 'resident') {
    return (
      <div className="px-8 py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › Financials</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Financials</h1>
        <p className="text-[14px] text-muted mb-8">Scheme financial health summary.</p>

        {/* Reserve fund */}
        <div className="bg-white border border-border rounded-lg px-6 py-5 mb-4">
          <div className="flex items-center justify-between mb-3">
            <span className="text-[13px] font-semibold text-ink">Reserve Fund</span>
            <span className="text-[12px] text-muted">{reservePct}% of target</span>
          </div>
          <div className="h-3 bg-border rounded-full overflow-hidden mb-2">
            <div className="h-full bg-accent rounded-full" style={{ width: `${reservePct}%` }} />
          </div>
          <div className="flex justify-between text-[12px] text-muted">
            <span>Balance: <strong className="text-ink">{formatRand(mockReserveFund.balance_cents)}</strong></span>
            <span>Target: {formatRand(mockReserveFund.target_cents)}</span>
          </div>
        </div>

        <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-5 text-[13px] text-ink leading-relaxed">
          The scheme's total approved budget for 2025 is <strong>{formatRand(totalBudgeted)}</strong>.
          Expenditure to date is <strong>{formatRand(totalActual)}</strong>.
          The reserve fund stands at {reservePct}% of the 10-year maintenance plan target.
        </div>
      </div>
    )
  }

  // Agent / Trustee: full financial view
  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Financials</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Financials</h1>
      <p className="text-[14px] text-muted mb-8">Budget, expenditure, and reserve fund.</p>

      {/* Stat cards */}
      <div className="grid grid-cols-4 gap-4 mb-8">
        {[
          { label: 'Total budget',  value: formatRand(totalBudgeted) },
          { label: 'Spent to date', value: formatRand(totalActual) },
          { label: 'Reserve fund',  value: formatRand(mockReserveFund.balance_cents) },
          { label: surplus >= 0 ? 'Surplus' : 'Deficit', value: formatRand(Math.abs(surplus)) },
        ].map(({ label, value }) => (
          <div key={label} className="bg-white border border-border rounded-lg px-5 py-4">
            <div className="text-[22px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      {/* Reserve fund bar */}
      <div className="bg-white border border-border rounded-lg px-6 py-5 mb-6">
        <div className="flex items-center justify-between mb-3">
          <span className="text-[13px] font-semibold text-ink">Reserve Fund — {reservePct}% of target</span>
          <span className="text-[12px] text-muted">Target: {formatRand(mockReserveFund.target_cents)}</span>
        </div>
        <div className="h-3 bg-border rounded-full overflow-hidden">
          <div className="h-full bg-accent rounded-full" style={{ width: `${reservePct}%` }} />
        </div>
      </div>

      {/* Budget vs actual table */}
      <div className="bg-white border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Budget vs Actual — 2025</span>
        </div>
        <div className="px-5">
          <div className="grid grid-cols-[1fr_auto_auto_auto] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
            <span>Category</span><span>Budgeted</span><span>Actual</span><span>Variance</span>
          </div>
          {mockBudgetLines.map((line, i) => {
            const variance = line.budgeted_cents - line.actual_cents
            const over = variance < 0
            return (
              <div key={line.id} className={`grid grid-cols-[1fr_auto_auto_auto] gap-4 items-center py-3 text-[13px] ${i < mockBudgetLines.length - 1 ? 'border-b border-border' : ''}`}>
                <span className="text-ink">{line.category}</span>
                <span className="tabular-nums text-muted">{formatRand(line.budgeted_cents)}</span>
                <span className="tabular-nums text-ink font-medium">{formatRand(line.actual_cents)}</span>
                <span className={`tabular-nums text-[12px] font-semibold ${over ? 'text-red' : 'text-green'}`}>
                  {over ? '+' : '-'}{formatRand(Math.abs(variance))}
                </span>
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
