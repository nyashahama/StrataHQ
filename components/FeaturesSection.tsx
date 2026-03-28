import LevyMockPanel from './LevyMockPanel'
import MaintenanceMockPanel from './MaintenanceMockPanel'
import AGMMockPanel from './AGMMockPanel'

interface FeatureItem {
  check: string
}

interface FeatureBlockProps {
  tag: string
  heading: React.ReactNode
  body: string
  items: FeatureItem[]
  MockPanel: React.ComponentType
  flip?: boolean
  bg?: string
}

function CheckIcon() {
  return (
    <span className="w-[18px] h-[18px] rounded-full bg-green-bg border border-[rgba(39,103,73,0.2)] grid place-items-center flex-shrink-0 mt-[1px] text-[9px] text-green">
      ✓
    </span>
  )
}

function FeatureBlock({ tag, heading, body, items, MockPanel, flip = false, bg }: FeatureBlockProps) {
  return (
    <section className={`padding-section border-t border-border ${bg ?? ''}`}>
      <div className="max-w-container mx-auto px-container">
        <div
          className={`grid grid-cols-1 md:grid-cols-2 gap-[clamp(40px,6vw,80px)] items-center ${
            flip ? 'md:[direction:rtl]' : ''
          }`}
        >
          {/* Text */}
          <div className={flip ? '[direction:ltr]' : ''}>
            <span className="inline-block text-[11px] font-semibold tracking-[0.08em] uppercase text-accent bg-accent-bg rounded px-2 py-[3px] mb-[14px]">
              {tag}
            </span>
            <h3 className="font-serif text-clamp-feature font-bold leading-[1.2] tracking-[-0.015em] text-ink mb-[14px]">
              {heading}
            </h3>
            <p className="text-[15px] text-ink-2 leading-[1.7] mb-5">{body}</p>
            <ul className="flex flex-col gap-2">
              {items.map(({ check }) => (
                <li key={check} className="flex items-start gap-[10px] text-[14px] text-ink-2">
                  <CheckIcon />
                  {check}
                </li>
              ))}
            </ul>
          </div>

          {/* Mock panel */}
          <div className={`reveal ${flip ? '[direction:ltr]' : ''}`}>
            <MockPanel />
          </div>
        </div>
      </div>
    </section>
  )
}

export default function FeaturesSection() {
  return (
    <>
      <FeatureBlock
        tag="Levy & Payments"
        heading={<>Automate collections.<br />Eliminate the chase.</>}
        body="Statements go out on time, reminders escalate automatically, and your arrears ledger updates in real time. You see exceptions — not spreadsheets."
        items={[
          { check: 'Automated monthly statements per unit' },
          { check: 'Configurable arrears escalation rules' },
          { check: 'PayFast and EFT reconciliation' },
          { check: 'Full debtors age analysis report' },
          { check: 'Attorney handoff workflow built in' },
        ]}
        MockPanel={LevyMockPanel}
        bg="bg-surface border-b border-border"
      />

      <FeatureBlock
        tag="Maintenance"
        heading={<>Every job tracked,<br />from photo to sign-off.</>}
        body="Residents submit requests with photos. Trustees approve. Contractors get assigned. SLA timers run. Nothing falls through the cracks."
        items={[
          { check: 'Photo-documented resident submissions' },
          { check: 'Contractor assignment and notification' },
          { check: 'SLA tracking with breach alerts' },
          { check: 'Trustee approval workflow for large jobs' },
          { check: 'Full maintenance history per unit' },
        ]}
        MockPanel={MaintenanceMockPanel}
        flip
      />

      <FeatureBlock
        tag="AGM & Voting"
        heading={<>Run compliant AGMs<br />without the chaos.</>}
        body="Digital proxy collection, automatic quorum calculation, secure live voting and instant results. Fully aligned with STSMA Act requirements."
        items={[
          { check: 'Digital notice and agenda distribution' },
          { check: 'Proxy form collection with audit trail' },
          { check: 'Automatic quorum tracking' },
          { check: 'Secure live vote casting per resolution' },
          { check: 'Signed minutes generated automatically' },
        ]}
        MockPanel={AGMMockPanel}
        bg="bg-surface border-b border-border"
      />
    </>
  )
}
