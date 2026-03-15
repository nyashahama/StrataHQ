const levyRows = [
  { unit: 'Unit 2A', name: 'Henderson, T.', amount: 'R 2,450', date: '1 Oct', status: 'Paid', pillClass: 'bg-green-bg text-green' },
  { unit: 'Unit 3B', name: 'Molefe, S.', amount: 'R 2,450', date: '1 Oct', status: 'Partial', pillClass: 'bg-yellowbg text-[#92400e]' },
  { unit: 'Unit 5C', name: 'van der Berg, L.', amount: 'R 3,100', date: '1 Oct', status: 'Overdue', pillClass: 'bg-red-bg text-red' },
  { unit: 'Unit 7A', name: 'Naidoo, R.', amount: 'R 2,450', date: '1 Oct', status: 'Paid', pillClass: 'bg-green-bg text-green' },
  { unit: 'Unit 9D', name: 'Khumalo, B.', amount: 'R 2,800', date: '1 Oct', status: 'Paid', pillClass: 'bg-green-bg text-green' },
]

export default function LevyMockPanel() {
  return (
    <div className="bg-white border border-border rounded-lg shadow-lg overflow-hidden">
      {/* Header */}
      <div className="border-b border-border px-[18px] py-[14px] flex items-center justify-between bg-white">
        <div className="flex items-center gap-2">
          <span className="text-[15px]">💳</span>
          <span className="text-[13px] font-semibold text-ink">Levy Status — October 2025</span>
        </div>
        <span className="text-[11px] font-semibold px-2 py-[2px] rounded-full bg-accent-bg text-accent">
          48 units
        </span>
      </div>

      {/* Body */}
      <div className="px-[18px] py-4">
        {/* Column headers */}
        <div className="grid grid-cols-[1fr_auto_auto_auto] gap-3 items-center pb-[6px] text-[11px] font-semibold text-muted">
          <span>UNIT</span>
          <span>AMOUNT</span>
          <span>DUE DATE</span>
          <span>STATUS</span>
        </div>

        {levyRows.map(({ unit, name, amount, date, status, pillClass }) => (
          <div
            key={unit}
            className="grid grid-cols-[1fr_auto_auto_auto] gap-3 items-center py-[10px] border-b border-border last:border-b-0 text-[13px]"
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
