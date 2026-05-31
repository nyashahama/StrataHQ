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
  accent?: 'blue' | 'warm'
}

function CheckIcon() {
  return (
    <span className="w-[20px] h-[20px] rounded-full bg-green-bg border border-[rgba(39,103,73,0.18)] grid place-items-center flex-shrink-0 mt-[1px] text-[10px] text-green font-bold">
      ✓
    </span>
  )
}

function FeatureBlock({ tag, heading, body, items, MockPanel, flip = false, accent = 'blue' }: FeatureBlockProps) {
  const isWarm = accent === 'warm'

  return (
    <section className="padding-section border-t border-border">
      <div className="max-w-[1080px] mx-auto px-container">
        <div
          className={`grid grid-cols-1 md:grid-cols-2 gap-[clamp(40px,6vw,80px)] items-center ${
            flip ? 'md:[direction:rtl]' : ''
          }`}
        >
          {/* Text */}
          <div className={flip ? '[direction:ltr]' : ''}>
            <span className={`inline-block text-[11px] font-semibold tracking-[0.1em] uppercase rounded-lg px-3 py-1.5 mb-4 border ${
              isWarm
                ? 'text-[var(--color-hero-warm)] bg-[rgba(244,162,97,0.08)] border-[rgba(244,162,97,0.15)]'
                : 'text-accent bg-accent-bg border-[rgba(43,108,176,0.12)]'
            }`}>
              {tag}
            </span>
            <h3 className="font-serif text-clamp-feature font-bold leading-[1.18] tracking-[-0.015em] text-ink mb-4">
              {heading}
            </h3>
            <p className="text-[15px] text-ink-2 leading-[1.7] mb-6">{body}</p>
            <ul className="flex flex-col gap-2.5">
              {items.map(({ check }) => (
                <li key={check} className="flex items-start gap-3 text-[14px] text-ink-2 leading-[1.5]">
                  <CheckIcon />
                  {check}
                </li>
              ))}
            </ul>
          </div>

          {/* Mock panel */}
          <div className={`reveal ${flip ? '[direction:ltr]' : ''}`}>
            <div className="card-lift rounded-xl">
              <MockPanel />
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}

export default function FeaturesSection() {
  return (
    <section id="features">
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
        accent="warm"
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
      />
    </section>
  )
}
