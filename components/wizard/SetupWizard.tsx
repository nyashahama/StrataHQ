'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { useMockAuth } from '@/lib/mock-auth'

interface SetupWizardProps {
  initialStep?: number
}

const STEP_NAMES = ['Firm', 'Scheme', 'Units', 'Levies', 'Invite']

const INPUT_CLASS =
  'border border-border rounded px-3 py-[10px] text-[14px] text-ink bg-white w-full focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent'
const LABEL_CLASS = 'text-[13px] font-medium text-ink mb-1 block'

function FieldGroup({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="mb-4">
      <label className={LABEL_CLASS}>{label}</label>
      {children}
    </div>
  )
}

export default function SetupWizard({ initialStep = 1 }: SetupWizardProps) {
  const { login } = useMockAuth()
  const router = useRouter()

  const [step, setStep] = useState(initialStep)
  const [error, setError] = useState('')

  // Step 1 — Firm
  const [firmName, setFirmName] = useState('')
  const [contactEmail, setContactEmail] = useState('')
  const [phone, setPhone] = useState('')

  // Step 2 — Scheme
  const [schemeName, setSchemeName] = useState('')
  const [physicalAddress, setPhysicalAddress] = useState('')
  const [schemeNumber, setSchemeNumber] = useState('')

  // Step 3 — Units
  const [unitsText, setUnitsText] = useState('')

  // Step 4 — Levies
  const [baseLevy, setBaseLevy] = useState('')
  const [adminLevy, setAdminLevy] = useState('')
  const [levyPeriod, setLevyPeriod] = useState('Monthly')

  // Step 5 — Invite
  const [invitesText, setInvitesText] = useState('')

  function validate(): boolean {
    setError('')
    if (step === 1) {
      if (!firmName.trim()) {
        setError('Company name is required.')
        return false
      }
      if (!contactEmail.trim()) {
        setError('Contact email is required.')
        return false
      }
    }
    if (step === 2) {
      if (!schemeName.trim()) {
        setError('Scheme name is required.')
        return false
      }
    }
    return true
  }

  function handleNext() {
    if (!validate()) return
    setStep((s) => s + 1)
  }

  function handleBack() {
    setError('')
    setStep((s) => s - 1)
  }

  function handleFinish() {
    login({
      role: 'agent',
      orgName: firmName || 'My Firm',
      schemeName: schemeName || 'My Scheme',
      schemeId: 'scheme-001',
      isWizardComplete: true,
    })
    router.push('/agent')
  }

  const stepName = STEP_NAMES[step - 1]

  return (
    <div className="min-h-screen bg-page flex items-center justify-center px-4 py-12">
      <div className="w-full max-w-[480px]">
        <h1 className="font-serif text-[26px] font-semibold text-ink">
          Set up your account
        </h1>
        <p className="text-[14px] text-muted mb-8">
          Step {step} of 5 — {stepName}
        </p>

        {/* Progress bar */}
        <div className="flex gap-1.5 mb-8">
          {STEP_NAMES.map((_, i) => (
            <div
              key={i}
              className={`flex-1 h-1 rounded-full ${i < step ? 'bg-accent' : 'bg-border'}`}
            />
          ))}
        </div>

        {/* Step content */}
        {step === 1 && (
          <div>
            <FieldGroup label="Company name">
              <input
                type="text"
                className={INPUT_CLASS}
                value={firmName}
                onChange={(e) => setFirmName(e.target.value)}
                placeholder="Acme Property Management"
              />
            </FieldGroup>
            <FieldGroup label="Contact email">
              <input
                type="email"
                className={INPUT_CLASS}
                value={contactEmail}
                onChange={(e) => setContactEmail(e.target.value)}
                placeholder="admin@acme.co.za"
              />
            </FieldGroup>
            <FieldGroup label="Phone">
              <input
                type="tel"
                className={INPUT_CLASS}
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder="+27 11 000 0000"
              />
            </FieldGroup>
          </div>
        )}

        {step === 2 && (
          <div>
            <FieldGroup label="Scheme name">
              <input
                type="text"
                className={INPUT_CLASS}
                value={schemeName}
                onChange={(e) => setSchemeName(e.target.value)}
                placeholder="Sunset Gardens"
              />
            </FieldGroup>
            <FieldGroup label="Physical address">
              <input
                type="text"
                className={INPUT_CLASS}
                value={physicalAddress}
                onChange={(e) => setPhysicalAddress(e.target.value)}
                placeholder="12 Oak Avenue, Johannesburg"
              />
            </FieldGroup>
            <FieldGroup label="Scheme number">
              <input
                type="text"
                className={INPUT_CLASS}
                value={schemeNumber}
                onChange={(e) => setSchemeNumber(e.target.value)}
                placeholder="SS 42/2010"
              />
            </FieldGroup>
          </div>
        )}

        {step === 3 && (
          <div>
            <FieldGroup label="Unit identifiers">
              <textarea
                className={`${INPUT_CLASS} h-32 resize-none`}
                value={unitsText}
                onChange={(e) => setUnitsText(e.target.value)}
                placeholder="1A&#10;1B&#10;2A"
              />
            </FieldGroup>
            <p className="text-[13px] text-muted -mt-2">
              One per line or comma-separated (e.g. 1A, 1B, 2A)
            </p>
          </div>
        )}

        {step === 4 && (
          <div>
            <FieldGroup label="Base levy (ZAR)">
              <input
                type="number"
                className={INPUT_CLASS}
                value={baseLevy}
                onChange={(e) => setBaseLevy(e.target.value)}
                min={0}
                step={0.01}
                placeholder="1200.00"
              />
            </FieldGroup>
            <FieldGroup label="Admin levy (ZAR)">
              <input
                type="number"
                className={INPUT_CLASS}
                value={adminLevy}
                onChange={(e) => setAdminLevy(e.target.value)}
                min={0}
                step={0.01}
                placeholder="1200.00"
              />
            </FieldGroup>
            <FieldGroup label="Levy period">
              <select
                className={INPUT_CLASS}
                value={levyPeriod}
                onChange={(e) => setLevyPeriod(e.target.value)}
              >
                <option value="Monthly">Monthly</option>
                <option value="Quarterly">Quarterly</option>
                <option value="Bi-annual">Bi-annual</option>
                <option value="Annual">Annual</option>
              </select>
            </FieldGroup>
          </div>
        )}

        {step === 5 && (
          <div>
            <FieldGroup label="Invite trustees and residents">
              <textarea
                className={`${INPUT_CLASS} h-28 resize-none font-mono`}
                value={invitesText}
                onChange={(e) => setInvitesText(e.target.value)}
                placeholder={'alice@example.com trustee\nbob@example.com resident'}
              />
            </FieldGroup>
            <p className="text-[13px] text-muted -mt-2">
              One per line: email then role (trustee or resident). E.g.
              &quot;alice@example.com trustee&quot;
            </p>
          </div>
        )}

        {error && <p className="text-[13px] text-red mt-4">{error}</p>}

        {/* Action bar */}
        <div className="flex gap-3 mt-8">
          {step > 1 && (
            <button
              type="button"
              onClick={handleBack}
              className="px-5 py-[10px] text-[14px] font-medium text-ink border border-border rounded hover:bg-[#f0efe9]"
            >
              Back
            </button>
          )}

          {step === 5 && (
            <button
              type="button"
              onClick={handleFinish}
              className="px-5 py-[10px] text-[14px] font-medium text-muted border border-border rounded hover:bg-[#f0efe9]"
            >
              Skip for now
            </button>
          )}

          <button
            type="button"
            onClick={step < 5 ? handleNext : handleFinish}
            className="flex-1 bg-ink text-white text-[14px] font-medium py-[10px] rounded hover:bg-ink-2"
          >
            {step === 5 ? 'Finish' : 'Next →'}
          </button>
        </div>
      </div>
    </div>
  )
}
