'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { useToast } from '@/lib/toast'
import { mockLevyRoll, mockLevyPeriod, mockCollectionTrend, mockUnit4BPayments } from '@/lib/mock/levy'

function formatRand(cents: number): string {
  return `R ${(cents / 100).toLocaleString('en-ZA', { minimumFractionDigits: 0 })}`
}

const STATUS_STYLES: Record<string, string> = {
  paid:    'bg-green-bg text-green',
  partial: 'bg-yellowbg text-amber',
  overdue: 'bg-red-bg text-red',
  pending: 'bg-accent-bg text-accent',
}

export default function LevyPaymentsPage() {
  const { user } = useMockAuth()
  const { addToast } = useToast()

  if (user?.role === 'resident') {
    const myAccount = mockLevyRoll.find(a => a.unit_identifier === user.unitIdentifier)
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › My Levy</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">My Levy</h1>
        <p className="text-[14px] text-muted mb-8">Levy account for Unit {user.unitIdentifier}.</p>

        {/* Current levy card */}
        <div className="bg-surface border border-border rounded-lg px-6 py-5 mb-6 flex items-center justify-between">
          <div>
            <p className="text-[12px] text-muted mb-1">{mockLevyPeriod.label} · due {new Date(mockLevyPeriod.due_date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' })}</p>
            <p className="font-serif text-[32px] font-semibold text-ink leading-none">{formatRand(mockLevyPeriod.amount_cents)}</p>
          </div>
          {myAccount && (
            <span className={`text-[12px] font-semibold px-3 py-1 rounded-full ${STATUS_STYLES[myAccount.status]}`}>
              {myAccount.status.charAt(0).toUpperCase() + myAccount.status.slice(1)}
            </span>
          )}
        </div>

        {/* Payment history */}
        <h2 className="text-[14px] font-semibold text-ink mb-3">Payment history</h2>
        {(() => {
          const myLevyAccount2 = mockLevyRoll.find(a => a.unit_identifier === user?.unitIdentifier)
          const myPayments = myLevyAccount2
            ? mockUnit4BPayments.filter(p => p.levy_account_id === myLevyAccount2.id)
            : []
          if (myPayments.length === 0) {
            return (
              <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-8 text-center text-muted text-[13px] mb-6">
                No payment history available for this unit.
              </div>
            )
          }
          return (
            <div className="bg-surface border border-border rounded-lg overflow-hidden mb-6">
              <div className="overflow-x-auto">
                {myPayments.map((p, i) => (
                  <div key={p.id} className={`flex items-center justify-between px-5 py-3 text-[13px] min-w-[400px] ${i < myPayments.length - 1 ? 'border-b border-border' : ''}`}>
                    <div>
                      <span className="font-medium text-ink">{formatRand(p.amount_cents)}</span>
                      <span className="text-muted ml-3">{new Date(p.date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' })}</span>
                    </div>
                    <span className="text-[11px] text-muted font-mono flex-shrink-0">{p.reference}</span>
                  </div>
                ))}
              </div>
            </div>
          )
        })()}

        <button onClick={() => addToast('Statement download started — check your downloads folder.', 'info')} className="text-[12px] text-accent font-medium border border-accent rounded px-4 py-2 hover:bg-accent-dim transition-colors">
          Download statement (PDF)
        </button>
      </div>
    )
  }

  // Agent / Trustee view
  const canEdit = user?.role === 'agent'
  const collected = mockLevyRoll.filter(a => a.status === 'paid').length
  const overdue = mockLevyRoll.filter(a => a.status === 'overdue').length
  const totalCollected = mockLevyRoll.reduce((sum, a) => sum + a.paid_cents, 0)
  const latestPct = mockCollectionTrend[mockCollectionTrend.length - 1].pct

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Levy & Payments</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Levy & Payments</h1>
      <p className="text-[14px] text-muted mb-8">Levy collection, statements, and payment history.</p>

      {/* Stat cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
        {[
          { label: 'Collection rate', value: `${latestPct}%` },
          { label: 'Total collected', value: formatRand(totalCollected) },
          { label: 'Overdue', value: String(overdue) },
          { label: 'Monthly levy', value: formatRand(mockLevyPeriod.amount_cents) },
        ].map(({ label, value }) => (
          <div key={label} className="bg-surface border border-border rounded-lg px-5 py-4">
            <div className="text-[24px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      {/* Collection trend chart */}
      <div className="bg-surface border border-border rounded-lg px-6 py-5 mb-6">
        <p className="text-[11px] font-semibold text-muted uppercase tracking-[0.08em] mb-4">Collection rate — 6 months</p>
        <div className="flex items-end gap-2 h-[56px] mb-1">
          {mockCollectionTrend.map((d, i) => {
            const isLast = i === mockCollectionTrend.length - 1
            const heightPct = ((d.pct - 80) / 20) * 100
            return (
              <div key={d.month} className="flex-1 h-full flex flex-col items-center justify-end gap-1">
                <span className="text-[9px] font-semibold leading-none" style={{ color: isLast ? '#2B6CB0' : '#A8A49E' }}>{d.pct}%</span>
                <div className="w-full rounded-[2px]" style={{ height: `${Math.max(heightPct, 8)}%`, background: isLast ? '#2B6CB0' : '#E3E2DF' }} />
              </div>
            )
          })}
        </div>
        <div className="flex">
          {mockCollectionTrend.map(d => (
            <span key={d.month} className="flex-1 text-center text-[9px] text-muted">{d.month}</span>
          ))}
        </div>
      </div>

      {/* Levy roll */}
      <div className="bg-surface border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border flex items-center justify-between">
          <span className="text-[13px] font-semibold text-ink">Levy Roll — {mockLevyPeriod.label}</span>
          <span className="text-[11px] font-semibold px-2 py-[2px] rounded-full bg-accent-bg text-accent">{mockLevyRoll.length} shown</span>
        </div>
        <div className="overflow-x-auto">
          <div className="px-5 min-w-[560px]">
            <div className="grid grid-cols-[1fr_auto_auto_auto] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
              <span>Unit</span><span>Amount</span><span>Due</span><span>Status</span>
            </div>
            {mockLevyRoll.map((account) => (
              <div key={account.id} className="grid grid-cols-[1fr_auto_auto_auto] gap-4 items-center py-3 border-b border-border last:border-b-0 text-[13px]">
                <div>
                  <div className="font-semibold text-ink">Unit {account.unit_identifier}</div>
                  <div className="text-[12px] text-muted">{account.owner_name}</div>
                </div>
                <span className="font-semibold text-ink tabular-nums">{formatRand(account.amount_cents)}</span>
                <span className="text-[12px] text-muted">{new Date(account.due_date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' })}</span>
                <div className="flex items-center gap-2">
                  <span className={`text-[11px] font-semibold px-[10px] py-[3px] rounded-full ${STATUS_STYLES[account.status]}`}>
                    {account.status.charAt(0).toUpperCase() + account.status.slice(1)}
                  </span>
                  {canEdit && account.status === 'overdue' && (
                    <button onClick={() => addToast(`Reminder sent to ${account.owner_name}`, 'info')} className="text-[11px] text-accent font-medium hover:underline">Remind</button>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
