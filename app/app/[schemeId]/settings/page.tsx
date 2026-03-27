'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { mockScheme } from '@/lib/mock/scheme'

export default function SchemeSettingsPage() {
  const { user } = useMockAuth()

  return (
    <div className="px-8 py-8 max-w-[700px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Settings</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Scheme Settings</h1>
      <p className="text-[14px] text-muted mb-8">Manage scheme details and configuration.</p>

      {/* Scheme details */}
      <div className="bg-white border border-border rounded-lg overflow-hidden mb-6">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Scheme details</span>
        </div>
        <div className="px-5 py-4 flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Scheme name</label>
            <input
              type="text"
              defaultValue={mockScheme.name}
              disabled={user?.role !== 'agent'}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent disabled:bg-page disabled:text-muted"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Physical address</label>
            <input
              type="text"
              defaultValue={mockScheme.address}
              disabled={user?.role !== 'agent'}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent disabled:bg-page disabled:text-muted"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Total units</label>
            <input
              type="number"
              defaultValue={mockScheme.unit_count}
              disabled={user?.role !== 'agent'}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent disabled:bg-page disabled:text-muted"
            />
          </div>
          {user?.role === 'agent' && (
            <div>
              <button className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:bg-[#245a96] transition-colors">
                Save changes
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Levy configuration */}
      <div className="bg-white border border-border rounded-lg overflow-hidden mb-6">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Levy configuration</span>
        </div>
        <div className="px-5 py-4 flex flex-col gap-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Base levy (ZAR)</label>
              <input
                type="text"
                defaultValue="2 450.00"
                disabled={user?.role !== 'agent'}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent disabled:bg-page disabled:text-muted"
              />
            </div>
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Levy period</label>
              <select
                disabled={user?.role !== 'agent'}
                defaultValue="Monthly"
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent disabled:bg-page disabled:text-muted"
              >
                <option>Monthly</option>
                <option>Quarterly</option>
              </select>
            </div>
          </div>
          {user?.role === 'agent' && (
            <div>
              <button className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:bg-[#245a96] transition-colors">
                Update levy settings
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Notifications */}
      <div className="bg-white border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Notifications</span>
        </div>
        <div className="px-5 py-4 flex flex-col gap-3">
          {[
            { label: 'Overdue levy reminders', defaultChecked: true },
            { label: 'Maintenance SLA breach alerts', defaultChecked: true },
            { label: 'New maintenance requests', defaultChecked: false },
            { label: 'AGM reminders (30 days before)', defaultChecked: true },
          ].map(item => (
            <label key={item.label} className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                defaultChecked={item.defaultChecked}
                disabled={user?.role !== 'agent'}
                className="w-4 h-4 accent-accent"
              />
              <span className="text-[13px] text-ink">{item.label}</span>
            </label>
          ))}
        </div>
      </div>
    </div>
  )
}
