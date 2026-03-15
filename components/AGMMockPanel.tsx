export default function AGMMockPanel() {
  return (
    <div className="bg-white border border-border rounded-lg shadow-lg overflow-hidden">
      {/* Header */}
      <div className="border-b border-border px-[18px] py-[14px] flex items-center justify-between bg-white">
        <div className="flex items-center gap-2">
          <span className="text-[15px]">🗳️</span>
          <span className="text-[13px] font-semibold text-ink">AGM — Rosewood Estate</span>
        </div>
        <span className="text-[11px] font-semibold px-2 py-[2px] rounded-full bg-accent-bg text-accent">
          14 Nov 2025
        </span>
      </div>

      {/* Body */}
      <div className="p-4 flex flex-col gap-2">
        {/* Resolution */}
        <div className="bg-page border border-border rounded p-[14px]">
          <div className="text-[11px] font-semibold text-accent mb-[5px]">RESOLUTION 1 OF 3</div>
          <div className="text-[14px] font-semibold text-ink mb-[3px]">
            Approval of 2026 maintenance budget
          </div>
          <div className="text-[12px] text-muted mb-[10px]">
            Total: R 485,000 · 38/48 votes cast
          </div>
          <div>
            <div className="flex justify-between text-[11px] text-muted mb-1">
              <span>In favour · 31 votes</span>
              <span>Against · 7</span>
            </div>
            <div className="h-[6px] bg-border rounded-full overflow-hidden">
              <div className="vote-fill" style={{ width: '81%' }} />
            </div>
          </div>
        </div>

        {/* Quorum */}
        <div className="bg-page border border-border rounded p-[14px]">
          <div className="text-[11px] font-semibold text-accent mb-[5px]">QUORUM STATUS</div>
          <div className="text-[14px] font-semibold text-ink mb-[3px]">Quorum reached ✓</div>
          <div className="text-[12px] text-muted mb-[10px]">
            38 of 48 owners present or represented by proxy
          </div>
          <div>
            <div className="flex justify-between text-[11px] text-muted mb-1">
              <span>Present · 38</span>
              <span>Required · 25</span>
            </div>
            <div className="h-[6px] bg-border rounded-full overflow-hidden">
              <div className="vote-fill" style={{ width: '79%', background: '#276749' }} />
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
