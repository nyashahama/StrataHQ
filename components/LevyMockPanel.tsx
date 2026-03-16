const levyRows = [
  { unit: 'Unit 2A', name: 'Henderson, T.', amount: 'R 2,450', date: '1 Oct', status: 'Paid', pillClass: 'bg-green-bg text-green' },
  { unit: 'Unit 3B', name: 'Molefe, S.', amount: 'R 2,450', date: '1 Oct', status: 'Partial', pillClass: 'bg-yellowbg text-[#92400e]' },
  { unit: 'Unit 5C', name: 'van der Berg, L.', amount: 'R 3,100', date: '1 Oct', status: 'Overdue', pillClass: 'bg-red-bg text-red' },
  { unit: 'Unit 7A', name: 'Naidoo, R.', amount: 'R 2,450', date: '1 Oct', status: 'Paid', pillClass: 'bg-green-bg text-green' },
  { unit: 'Unit 9D', name: 'Khumalo, B.', amount: 'R 2,800', date: '1 Oct', status: 'Paid', pillClass: 'bg-green-bg text-green' },
]

const trendData = [87, 89, 88, 91, 92, 94]
const trendMonths = ['May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct']

export default function LevyMockPanel() {
  return (
    <div className="bg-white border border-border rounded-lg shadow-lg overflow-hidden">
      {/* Collection rate trend header */}
      <div className="border-b border-border px-[18px] pt-[16px] pb-4 bg-white">
        <div className="flex items-center justify-between mb-3">
          <span className="text-[11px] font-semibold text-muted uppercase tracking-[0.08em]">
            Collection rate — 6 months
          </span>
          <span className="text-[11px] font-semibold text-green">↑ +7pp YoY</span>
        </div>
        {/* Mini bar chart */}
        <div className="flex items-end gap-[6px] h-[40px] mb-1">
          {trendData.map((v, i) => {
            const last = i === trendData.length - 1
            const heightPct = ((v - 80) / 20) * 100
            return (
              <div key={i} className="flex-1 flex flex-col items-center justify-end gap-[2px]">
                <span
                  className="text-[8.5px] font-semibold leading-none"
                  style={{ color: last ? '#2B6CB0' : '#A8A49E' }}
                >
                  {v}%
                </span>
                <div
                  className="w-full rounded-[2px]"
                  style={{
                    height: `${heightPct}%`,
                    background: last ? '#2B6CB0' : '#E3E2DF',
                    minHeight: 3,
                  }}
                />
              </div>
            )
          })}
        </div>
        <div className="flex justify-between">
          {trendMonths.map(m => (
            <span key={m} className="flex-1 text-center text-[9px] text-muted-2">{m}</span>
          ))}
        </div>
      </div>

      {/* Levy roll */}
      <div className="px-[18px] py-[10px] bg-page border-b border-border flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-[15px]">💳</span>
          <span className="text-[13px] font-semibold text-ink">Levy Roll — October 2025</span>
        </div>
        <span className="text-[11px] font-semibold px-2 py-[2px] rounded-full bg-accent-bg text-accent">
          48 units
        </span>
      </div>

      <div className="px-[18px] py-3">
        {/* Column headers */}
        <div className="grid grid-cols-[1fr_auto_auto_auto] gap-3 items-center pb-[6px] text-[11px] font-semibold text-muted">
          <span>UNIT</span>
          <span>AMOUNT</span>
          <span>DUE</span>
          <span>STATUS</span>
        </div>

        {levyRows.map(({ unit, name, amount, date, status, pillClass }) => (
          <div
            key={unit}
            className="grid grid-cols-[1fr_auto_auto_auto] gap-3 items-center py-[9px] border-b border-border last:border-b-0 text-[13px]"
          >
            <div>
              <div className="font-semibold text-ink">{unit}</div>
              <div className="text-[12px] text-muted">{name}</div>
            </div>
            <span className="font-semibold text-ink tabular-nums">{amount}</span>
            <span className="text-[12px] text-muted">{date}</span>
            <span className={`text-[11px] font-semibold px-[10px] py-[3px] rounded-full tracking-[0.02em] ${pillClass}`}>
              {status}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}
