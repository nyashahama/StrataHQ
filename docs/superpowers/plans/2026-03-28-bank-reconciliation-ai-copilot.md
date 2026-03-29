# Bank Statement Reconciliation + AI Scheme Copilot — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add two killer features — (1) drag-and-drop bank statement CSV reconciliation that auto-matches EFT payments to levy accounts, and (2) a floating AI copilot chat panel that answers natural-language questions about any scheme using Claude.

**Architecture:** Reconciliation runs entirely in the browser — pure TypeScript matching algorithm against the existing mock levy roll, with a 3-step modal UI mounted on the levy page. The AI copilot uses a Next.js API route (`/api/copilot`) that streams Claude responses with all mock scheme data injected as system context. A floating `<Copilot>` component mounts in the scheme and agent layouts.

**Tech Stack:** @anthropic-ai/sdk (streaming), Next.js App Router API routes, React hooks for step state / chat history, Tailwind for UI. No new data stores — reconciliation updates existing React state; copilot history is ephemeral in component state.

---

## File Map

| Action | File | Purpose |
|--------|------|---------|
| Add dep | `package.json` | Add `@anthropic-ai/sdk` |
| Create | `lib/mock/bank-statement.ts` | Sample bank statement CSV string + typed transactions for demo |
| Create | `lib/reconcile.ts` | CSV column detection, transaction parsing, and matching algorithm |
| Create | `components/ReconcileModal.tsx` | 3-step reconciliation modal (Upload → Review → Confirm) |
| Modify | `app/app/[schemeId]/levy/page.tsx` | Convert levy roll to local state, add Reconcile button, mount modal |
| Create | `app/api/copilot/route.ts` | Streaming POST handler that calls Claude with scheme context |
| Create | `components/Copilot.tsx` | Floating chat button + sliding panel with streaming responses |
| Modify | `app/app/[schemeId]/layout.tsx` | Mount `<Copilot>` (agent + trustee only) |
| Modify | `app/agent/layout.tsx` | Mount `<Copilot>` on portfolio pages |

---

## Part A — Bank Statement Reconciliation

### Task 1: Install @anthropic-ai/sdk

**Files:**
- Modify: `package.json`

- [ ] **Step 1: Install the SDK**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app
npm install @anthropic-ai/sdk
```

Expected: `added N packages` with no errors.

- [ ] **Step 2: Verify it appears in package.json**

Check that `"@anthropic-ai/sdk"` appears under `"dependencies"` in `package.json`.

- [ ] **Step 3: Commit**

```bash
git add package.json package-lock.json
git commit -m "chore: add @anthropic-ai/sdk dependency"
```

---

### Task 2: Create sample bank statement mock data

**Files:**
- Create: `lib/mock/bank-statement.ts`

This file provides a realistic South African bank statement CSV for demo purposes. The data intentionally maps to the existing mock levy roll:
- Units 1A, 4B, 5A, 7B, 8A → full payment (high confidence)
- Unit 2B → partial R1 200 of R2 450 (medium confidence, unit ID in description)
- Unit 6C → partial R500 (low confidence, surname only)
- Unit 3A (van der Berg) → no match (overdue, no reference in statement)
- Bank charges → negative amount, filtered out by algorithm

- [ ] **Step 1: Create the file**

```typescript
// lib/mock/bank-statement.ts

export interface BankTransaction {
  date: string
  description: string
  amount_cents: number   // positive = credit, negative = debit
  running_balance_cents: number
}

export const SAMPLE_BANK_STATEMENT_CSV = `Date,Description,Amount,Balance
2025-10-01,INTERNET TRF FROM HENDERSON T UNIT1A LEVY OCT,2450.00,15250.00
2025-10-02,INTERNET TRF MOLEFE UNIT 2B LEVY OCT,1200.00,16450.00
2025-10-01,INTERNET TRF KHUMALO B 5A OCT LEVY,2450.00,18900.00
2025-10-03,INTERNET TRF NAIDOO R UNIT4B OCT,2450.00,21350.00
2025-10-05,INTERNET TRF PETERSEN M 7B LEVY OCT,2450.00,23800.00
2025-10-02,INTERNET TRF DLAMINI S UNIT 8A LEVY,2450.00,26250.00
2025-10-10,BANK CHARGES AND FEES,-85.00,26165.00
2025-10-07,INTERNET TRF ABRAHAMS J 6C PARTIAL LEVY,500.00,26665.00`

export function parseSampleStatement(): BankTransaction[] {
  return parseBankStatementCSV(SAMPLE_BANK_STATEMENT_CSV)
}

// Re-export parser so the modal can call it on uploaded files too
export { parseBankStatementCSV } from '@/lib/reconcile'
```

- [ ] **Step 2: Commit placeholder** (will be complete after Task 3 creates reconcile.ts)

We'll commit both files together at the end of Task 3.

---

### Task 3: Create the reconciliation algorithm

**Files:**
- Create: `lib/reconcile.ts`

- [ ] **Step 1: Create the file with all types and functions**

```typescript
// lib/reconcile.ts
import type { LevyAccount } from '@/lib/mock/levy'

// ── Types ──────────────────────────────────────────────────────────────────

export interface ParsedTransaction {
  date: string
  description: string
  amount_cents: number   // positive = credit (inbound payment)
  running_balance_cents: number
  raw: string            // original CSV row for debugging
}

export type MatchConfidence = 'high' | 'medium' | 'low' | 'unmatched'

export interface ReconcileMatch {
  transaction: ParsedTransaction
  account: LevyAccount | null
  confidence: MatchConfidence
  match_reason: string
  // what applying this match would do:
  new_paid_cents: number
  new_status: LevyAccount['status']
}

// ── CSV Parsing ────────────────────────────────────────────────────────────

/**
 * Parse a CSV string into transactions.
 * Handles variable column order by inspecting the header row.
 * Skips rows with negative amounts (debits / bank charges).
 */
export function parseBankStatementCSV(csv: string): ParsedTransaction[] {
  const lines = csv.trim().split('\n').map(l => l.trim()).filter(Boolean)
  if (lines.length < 2) return []

  // Detect column indices from header
  const header = lines[0].toLowerCase()
  const cols = header.split(',').map(c => c.replace(/["\s]/g, ''))

  const dateIdx    = cols.findIndex(c => c.includes('date'))
  const descIdx    = cols.findIndex(c => c.includes('desc') || c.includes('narr') || c.includes('reference') || c.includes('transaction'))
  const amtIdx     = cols.findIndex(c => c.includes('amount') || c === 'amt' || c.includes('credit'))
  const balIdx     = cols.findIndex(c => c.includes('balance') || c === 'bal')

  if (dateIdx === -1 || descIdx === -1 || amtIdx === -1) {
    throw new Error('Could not detect required columns (Date, Description, Amount) in this CSV. Check the file format.')
  }

  const transactions: ParsedTransaction[] = []

  for (let i = 1; i < lines.length; i++) {
    const row = splitCSVRow(lines[i])
    if (row.length <= Math.max(dateIdx, descIdx, amtIdx)) continue

    const amtRaw = row[amtIdx].replace(/[",\s]/g, '')
    const amount_cents = Math.round(parseFloat(amtRaw) * 100)

    if (isNaN(amount_cents) || amount_cents <= 0) continue  // skip debits / non-numeric

    const balRaw = balIdx !== -1 ? row[balIdx].replace(/[",\s]/g, '') : '0'
    const running_balance_cents = Math.round(parseFloat(balRaw) * 100) || 0

    transactions.push({
      date: row[dateIdx].replace(/"/g, '').trim(),
      description: row[descIdx].replace(/"/g, '').trim(),
      amount_cents,
      running_balance_cents,
      raw: lines[i],
    })
  }

  return transactions
}

/** Handles quoted CSV fields that may contain commas */
function splitCSVRow(row: string): string[] {
  const result: string[] = []
  let current = ''
  let inQuotes = false
  for (const ch of row) {
    if (ch === '"') { inQuotes = !inQuotes }
    else if (ch === ',' && !inQuotes) { result.push(current); current = '' }
    else { current += ch }
  }
  result.push(current)
  return result
}

// ── Matching Algorithm ─────────────────────────────────────────────────────

/**
 * Match each parsed transaction to a levy account.
 *
 * Priority:
 * 1. HIGH   — unit identifier found in description (e.g. "UNIT 4B", "4B LEVY", "UNIT4B")
 * 2. MEDIUM — exact amount match + owner surname found in description
 * 3. LOW    — exact amount match only (ambiguous if multiple units have same levy)
 * 4. UNMATCHED — nothing found
 *
 * Each account can only be matched once (first/highest confidence wins).
 */
export function matchTransactions(
  transactions: ParsedTransaction[],
  accounts: LevyAccount[],
): ReconcileMatch[] {
  const usedAccountIds = new Set<string>()

  return transactions.map(tx => {
    const descUpper = tx.description.toUpperCase()

    // 1. Try unit identifier match
    for (const account of accounts) {
      if (usedAccountIds.has(account.id)) continue
      const uid = account.unit_identifier.toUpperCase()
      // Match patterns: "UNIT 4B", "UNIT4B", " 4B ", "4B LEVY", "4BLEVY"
      const patterns = [
        `UNIT ${uid}`, `UNIT${uid}`, ` ${uid} `, ` ${uid},`, `${uid} LEVY`,
        `${uid}LEVY`, `-${uid}-`, `/${uid}/`,
      ]
      if (patterns.some(p => descUpper.includes(p))) {
        usedAccountIds.add(account.id)
        return buildMatch(tx, account, 'high', `Unit identifier "${account.unit_identifier}" found in description`)
      }
    }

    // 2. Try surname + amount match
    for (const account of accounts) {
      if (usedAccountIds.has(account.id)) continue
      const surname = extractSurname(account.owner_name).toUpperCase()
      if (surname.length < 3) continue  // too short to be meaningful
      const amountMatch = tx.amount_cents === account.amount_cents || tx.amount_cents < account.amount_cents
      if (amountMatch && descUpper.includes(surname)) {
        usedAccountIds.add(account.id)
        return buildMatch(tx, account, 'medium', `Surname "${surname}" + amount match`)
      }
    }

    // 3. Try exact amount match (last resort — risky if all levies are the same)
    const exactAmountAccounts = accounts.filter(
      a => !usedAccountIds.has(a.id) && tx.amount_cents === a.amount_cents && a.status !== 'paid'
    )
    if (exactAmountAccounts.length === 1) {
      usedAccountIds.add(exactAmountAccounts[0].id)
      return buildMatch(tx, exactAmountAccounts[0], 'low', 'Exact amount match (no name/unit reference found)')
    }

    return { transaction: tx, account: null, confidence: 'unmatched', match_reason: 'No matching levy account found', new_paid_cents: 0, new_status: 'pending' as const }
  })
}

function extractSurname(ownerName: string): string {
  // Format is "Surname, F." or "van der Berg, L." — take everything before the comma
  return ownerName.split(',')[0].trim()
}

function buildMatch(
  tx: ParsedTransaction,
  account: LevyAccount,
  confidence: MatchConfidence,
  match_reason: string,
): ReconcileMatch {
  const new_paid_cents = account.paid_cents + tx.amount_cents
  const new_status: LevyAccount['status'] =
    new_paid_cents >= account.amount_cents ? 'paid' :
    new_paid_cents > 0 ? 'partial' : 'overdue'

  return { transaction: tx, account, confidence, match_reason, new_paid_cents, new_status }
}

// ── Summary helpers ────────────────────────────────────────────────────────

export interface ReconcileSummary {
  total_transactions: number
  matched_high: number
  matched_medium: number
  matched_low: number
  unmatched: number
  total_amount_cents: number
  matched_amount_cents: number
}

export function summariseMatches(matches: ReconcileMatch[]): ReconcileSummary {
  return {
    total_transactions: matches.length,
    matched_high:   matches.filter(m => m.confidence === 'high').length,
    matched_medium: matches.filter(m => m.confidence === 'medium').length,
    matched_low:    matches.filter(m => m.confidence === 'low').length,
    unmatched:      matches.filter(m => m.confidence === 'unmatched').length,
    total_amount_cents:   matches.reduce((s, m) => s + m.transaction.amount_cents, 0),
    matched_amount_cents: matches.filter(m => m.account).reduce((s, m) => s + m.transaction.amount_cents, 0),
  }
}
```

- [ ] **Step 2: Update `lib/mock/bank-statement.ts` to import from reconcile**

Now that `reconcile.ts` exists, update `bank-statement.ts` (created in Task 2) to properly import:

```typescript
// lib/mock/bank-statement.ts
import { parseBankStatementCSV } from '@/lib/reconcile'

export interface BankTransaction {
  date: string
  description: string
  amount_cents: number
  running_balance_cents: number
}

export const SAMPLE_BANK_STATEMENT_CSV = `Date,Description,Amount,Balance
2025-10-01,INTERNET TRF FROM HENDERSON T UNIT1A LEVY OCT,2450.00,15250.00
2025-10-02,INTERNET TRF MOLEFE UNIT 2B LEVY OCT,1200.00,16450.00
2025-10-01,INTERNET TRF KHUMALO B 5A OCT LEVY,2450.00,18900.00
2025-10-03,INTERNET TRF NAIDOO R UNIT4B OCT,2450.00,21350.00
2025-10-05,INTERNET TRF PETERSEN M 7B LEVY OCT,2450.00,23800.00
2025-10-02,INTERNET TRF DLAMINI S UNIT 8A LEVY,2450.00,26250.00
2025-10-10,BANK CHARGES AND FEES,-85.00,26165.00
2025-10-07,INTERNET TRF ABRAHAMS J 6C PARTIAL LEVY,500.00,26665.00`

export function parseSampleStatement() {
  return parseBankStatementCSV(SAMPLE_BANK_STATEMENT_CSV)
}
```

- [ ] **Step 3: Verify algorithm logic manually**

Run a quick mental trace:
- `INTERNET TRF FROM HENDERSON T UNIT1A LEVY OCT` → contains `UNIT1A` (matches pattern `UNIT${uid}` for uid=`1A`) → HIGH ✓
- `INTERNET TRF MOLEFE UNIT 2B LEVY OCT` → contains `UNIT 2B` → HIGH ✓
- `INTERNET TRF KHUMALO B 5A OCT LEVY` → contains ` 5A ` (space-5A-space pattern) → HIGH ✓
- `BANK CHARGES AND FEES` with amount `-85.00` → filtered out (negative) ✓
- `INTERNET TRF ABRAHAMS J 6C PARTIAL LEVY` → contains ` 6C ` → HIGH ✓
- No reference to 3A (van der Berg) → UNMATCHED ✓

- [ ] **Step 4: Commit**

```bash
git add lib/reconcile.ts lib/mock/bank-statement.ts
git commit -m "feat: add bank statement CSV parser and levy matching algorithm"
```

---

### Task 4: Create the ReconcileModal component

**Files:**
- Create: `components/ReconcileModal.tsx`

This is a 3-step modal: Upload → Review → Confirm.

- [ ] **Step 1: Create the component**

```typescript
// components/ReconcileModal.tsx
'use client'
import { useState, useRef, useCallback } from 'react'
import { parseBankStatementCSV, matchTransactions, summariseMatches } from '@/lib/reconcile'
import { SAMPLE_BANK_STATEMENT_CSV } from '@/lib/mock/bank-statement'
import type { LevyAccount } from '@/lib/mock/levy'
import type { ReconcileMatch } from '@/lib/reconcile'

interface ReconcileModalProps {
  levyAccounts: LevyAccount[]
  periodLabel: string
  onConfirm: (updates: Array<{ id: string; paid_cents: number; status: LevyAccount['status'] }>) => void
  onClose: () => void
}

type Step = 'upload' | 'review' | 'confirm'

const CONFIDENCE_STYLES: Record<string, string> = {
  high:      'bg-green-bg text-green',
  medium:    'bg-yellowbg text-amber',
  low:       'bg-accent-bg text-accent',
  unmatched: 'bg-red-bg text-red',
}

const CONFIDENCE_LABELS: Record<string, string> = {
  high: 'Auto', medium: 'Review', low: 'Weak', unmatched: 'No match',
}

function formatRand(cents: number) {
  return `R ${(cents / 100).toLocaleString('en-ZA', { minimumFractionDigits: 0 })}`
}

export default function ReconcileModal({ levyAccounts, periodLabel, onConfirm, onClose }: ReconcileModalProps) {
  const [step, setStep] = useState<Step>('upload')
  const [matches, setMatches] = useState<ReconcileMatch[]>([])
  const [error, setError] = useState<string | null>(null)
  const [fileName, setFileName] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const processCSV = useCallback((csv: string, name: string) => {
    setError(null)
    try {
      const transactions = parseBankStatementCSV(csv)
      if (transactions.length === 0) {
        setError('No valid credit transactions found in this file. Make sure it is a bank statement CSV with a Date, Description, and Amount column.')
        return
      }
      const result = matchTransactions(transactions, levyAccounts)
      setMatches(result)
      setFileName(name)
      setStep('review')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not parse this file.')
    }
  }, [levyAccounts])

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = ev => processCSV(ev.target?.result as string, file.name)
    reader.readAsText(file)
  }

  function handleLoadSample() {
    processCSV(SAMPLE_BANK_STATEMENT_CSV, 'sample-statement.csv')
  }

  function handleConfirm() {
    const updates = matches
      .filter(m => m.account !== null)
      .map(m => ({ id: m.account!.id, paid_cents: m.new_paid_cents, status: m.new_status }))
    onConfirm(updates)
  }

  const summary = matches.length > 0 ? summariseMatches(matches) : null
  const matchedCount = matches.filter(m => m.account !== null).length

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-ink/50 backdrop-blur-[2px]">
      <div className="bg-surface border border-border rounded-xl shadow-2xl w-full max-w-[720px] max-h-[90vh] flex flex-col">

        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border flex-shrink-0">
          <div>
            <h2 className="font-serif text-[18px] font-semibold text-ink">Reconcile bank statement</h2>
            <p className="text-[12px] text-muted mt-0.5">{periodLabel}</p>
          </div>
          <div className="flex items-center gap-4">
            {/* Step indicators */}
            {(['upload', 'review', 'confirm'] as Step[]).map((s, i) => (
              <div key={s} className="flex items-center gap-2">
                {i > 0 && <div className="w-8 h-px bg-border" />}
                <div className={`flex items-center gap-1.5 text-[12px] font-medium ${step === s ? 'text-ink' : 'text-muted'}`}>
                  <div className={`w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-bold ${step === s ? 'bg-ink text-surface' : 'bg-border text-muted'}`}>
                    {i + 1}
                  </div>
                  <span className="hidden sm:inline capitalize">{s}</span>
                </div>
              </div>
            ))}
            <button onClick={onClose} className="ml-2 text-muted hover:text-ink transition-colors p-1 rounded">
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
                <line x1="3" y1="3" x2="13" y2="13" /><line x1="13" y1="3" x2="3" y2="13" />
              </svg>
            </button>
          </div>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-5">

          {/* ── Step 1: Upload ── */}
          {step === 'upload' && (
            <div>
              <p className="text-[14px] text-muted mb-6">
                Upload your bank statement CSV. StrataHQ will automatically match deposits to levy accounts using unit identifiers, owner names, and amounts.
              </p>

              {/* Drop zone */}
              <div
                className="border-2 border-dashed border-border rounded-lg px-8 py-12 text-center cursor-pointer hover:border-accent hover:bg-accent-dim transition-colors"
                onClick={() => fileInputRef.current?.click()}
              >
                <svg className="w-8 h-8 text-muted mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="1.5">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5" />
                </svg>
                <p className="text-[14px] font-medium text-ink mb-1">Drop CSV file here</p>
                <p className="text-[12px] text-muted">Standard Bank, FNB, Nedbank, ABSA, Capitec formats supported</p>
                <input ref={fileInputRef} type="file" accept=".csv,.txt" className="hidden" onChange={handleFileChange} />
              </div>

              {error && (
                <div className="mt-4 px-4 py-3 bg-red-bg border border-red rounded-lg text-[13px] text-red">{error}</div>
              )}

              <div className="flex items-center gap-3 mt-4">
                <div className="flex-1 h-px bg-border" />
                <span className="text-[12px] text-muted">or</span>
                <div className="flex-1 h-px bg-border" />
              </div>

              <button
                onClick={handleLoadSample}
                className="mt-4 w-full text-[13px] text-accent font-medium border border-accent rounded-lg px-4 py-3 hover:bg-accent-dim transition-colors"
              >
                Load sample statement (demo)
              </button>
            </div>
          )}

          {/* ── Step 2: Review ── */}
          {step === 'review' && (
            <div>
              <div className="flex items-center justify-between mb-4">
                <p className="text-[13px] text-muted">
                  {fileName} · {matches.length} transaction{matches.length !== 1 ? 's' : ''} · {matchedCount} matched
                </p>
                {summary && (
                  <div className="flex gap-2">
                    <span className={`text-[11px] font-semibold px-2 py-1 rounded-full ${CONFIDENCE_STYLES.high}`}>{summary.matched_high} auto</span>
                    {summary.matched_medium > 0 && <span className={`text-[11px] font-semibold px-2 py-1 rounded-full ${CONFIDENCE_STYLES.medium}`}>{summary.matched_medium} review</span>}
                    {summary.unmatched > 0 && <span className={`text-[11px] font-semibold px-2 py-1 rounded-full ${CONFIDENCE_STYLES.unmatched}`}>{summary.unmatched} unmatched</span>}
                  </div>
                )}
              </div>

              <div className="border border-border rounded-lg overflow-hidden">
                <div className="grid grid-cols-[1fr_90px_100px_80px] gap-3 px-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border bg-page">
                  <span>Transaction</span><span>Amount</span><span>Matched unit</span><span>Confidence</span>
                </div>
                {matches.map((m, i) => (
                  <div
                    key={i}
                    className={`grid grid-cols-[1fr_90px_100px_80px] gap-3 items-center px-4 py-3 text-[13px] ${i < matches.length - 1 ? 'border-b border-border' : ''} ${m.confidence === 'unmatched' ? 'opacity-50' : ''}`}
                  >
                    <div>
                      <div className="font-medium text-ink truncate">{m.transaction.description}</div>
                      <div className="text-[11px] text-muted">{m.transaction.date}</div>
                    </div>
                    <span className="font-semibold text-ink tabular-nums">{formatRand(m.transaction.amount_cents)}</span>
                    <div>
                      {m.account ? (
                        <>
                          <div className="font-medium text-ink">Unit {m.account.unit_identifier}</div>
                          <div className="text-[11px] text-muted truncate">{m.account.owner_name}</div>
                        </>
                      ) : (
                        <span className="text-muted text-[12px]">—</span>
                      )}
                    </div>
                    <span className={`text-[11px] font-semibold px-2 py-1 rounded-full text-center ${CONFIDENCE_STYLES[m.confidence]}`}>
                      {CONFIDENCE_LABELS[m.confidence]}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* ── Step 3: Confirm ── */}
          {step === 'confirm' && summary && (
            <div>
              <div className="bg-green-bg border border-green rounded-lg px-5 py-4 mb-6">
                <p className="text-[14px] font-semibold text-green mb-1">Ready to apply reconciliation</p>
                <p className="text-[13px] text-green/80">
                  {matchedCount} payment{matchedCount !== 1 ? 's' : ''} totalling {formatRand(summary.matched_amount_cents)} will be recorded against their levy accounts.
                  {summary.unmatched > 0 && ` ${summary.unmatched} transaction${summary.unmatched !== 1 ? 's' : ''} will be skipped.`}
                </p>
              </div>

              <div className="border border-border rounded-lg overflow-hidden">
                <div className="px-4 py-3 border-b border-border bg-page text-[11px] font-semibold text-muted uppercase tracking-wide">
                  Changes to be applied
                </div>
                {matches.filter(m => m.account !== null).map((m, i, arr) => (
                  <div key={i} className={`flex items-center justify-between px-4 py-3 text-[13px] ${i < arr.length - 1 ? 'border-b border-border' : ''}`}>
                    <div>
                      <span className="font-medium text-ink">Unit {m.account!.unit_identifier}</span>
                      <span className="text-muted ml-2">{m.account!.owner_name}</span>
                    </div>
                    <div className="flex items-center gap-3">
                      <span className="text-muted text-[12px] line-through">{m.account!.status}</span>
                      <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" className="text-muted"><path d="M2 6h8M7 3l3 3-3 3" strokeWidth="1.5" strokeLinecap="round" /></svg>
                      <span className={`text-[11px] font-semibold px-2 py-0.5 rounded-full ${
                        m.new_status === 'paid' ? 'bg-green-bg text-green' :
                        m.new_status === 'partial' ? 'bg-yellowbg text-amber' : 'bg-red-bg text-red'
                      }`}>{m.new_status}</span>
                      <span className="font-semibold text-ink tabular-nums">{formatRand(m.new_paid_cents)}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-6 py-4 border-t border-border flex-shrink-0">
          <button
            onClick={() => step === 'upload' ? onClose() : setStep(step === 'confirm' ? 'review' : 'upload')}
            className="text-[13px] text-muted hover:text-ink transition-colors"
          >
            {step === 'upload' ? 'Cancel' : '← Back'}
          </button>

          {step === 'review' && (
            <button
              onClick={() => setStep('confirm')}
              disabled={matchedCount === 0}
              className="text-[13px] font-semibold bg-ink text-surface px-5 py-2 rounded-lg hover:bg-ink/80 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Review {matchedCount} match{matchedCount !== 1 ? 'es' : ''} →
            </button>
          )}

          {step === 'confirm' && (
            <button
              onClick={handleConfirm}
              className="text-[13px] font-semibold bg-green text-white px-5 py-2 rounded-lg hover:bg-green/80 transition-colors"
            >
              Apply reconciliation
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add components/ReconcileModal.tsx
git commit -m "feat: add ReconcileModal with 3-step upload/review/confirm workflow"
```

---

### Task 5: Wire reconciliation into the levy page

**Files:**
- Modify: `app/app/[schemeId]/levy/page.tsx`

Changes needed:
1. Convert `mockLevyRoll` import to local React state so reconciliation can update statuses
2. Add `useState` for modal open/close
3. Add "Reconcile bank statement" button (agent-only) in the header area
4. Mount `<ReconcileModal>` when open
5. On confirm, apply updates to state and show a success toast

- [ ] **Step 1: Replace the top of the file**

Replace the existing imports and the opening of `LevyPaymentsPage` with:

```tsx
'use client'
import { useState } from 'react'
import { useMockAuth } from '@/lib/mock-auth'
import { useToast } from '@/lib/toast'
import { mockLevyRoll, mockLevyPeriod, mockCollectionTrend, mockUnit4BPayments } from '@/lib/mock/levy'
import type { LevyAccount } from '@/lib/mock/levy'
import ReconcileModal from '@/components/ReconcileModal'

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
  const [levyRoll, setLevyRoll] = useState<LevyAccount[]>(mockLevyRoll)
  const [reconcileOpen, setReconcileOpen] = useState(false)

  function handleReconcileConfirm(updates: Array<{ id: string; paid_cents: number; status: LevyAccount['status'] }>) {
    setLevyRoll(prev => prev.map(account => {
      const update = updates.find(u => u.id === account.id)
      if (!update) return account
      return { ...account, paid_cents: update.paid_cents, status: update.status, paid_date: new Date().toISOString() }
    }))
    setReconcileOpen(false)
    addToast(`Reconciliation applied — ${updates.length} accounts updated.`, 'success')
  }
```

- [ ] **Step 2: Update the agent/trustee view header section**

Find this existing block in the agent/trustee view:

```tsx
      <p className="text-[12px] text-muted mb-4">Scheme › Levy & Payments</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Levy & Payments</h1>
      <p className="text-[14px] text-muted mb-8">Levy collection, statements, and payment history.</p>
```

Replace it with:

```tsx
      <div className="flex items-start justify-between mb-8">
        <div>
          <p className="text-[12px] text-muted mb-1">Scheme › Levy & Payments</p>
          <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Levy & Payments</h1>
          <p className="text-[14px] text-muted">Levy collection, statements, and payment history.</p>
        </div>
        {canEdit && (
          <button
            onClick={() => setReconcileOpen(true)}
            className="flex-shrink-0 flex items-center gap-2 text-[13px] font-semibold bg-ink text-surface px-4 py-2 rounded-lg hover:bg-ink/80 transition-colors mt-1"
          >
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
              <path d="M1 4h12M1 8h8M1 12h5" strokeLinecap="round" />
              <path d="M10 9l2 2 2-2" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
            Reconcile statement
          </button>
        )}
      </div>
```

- [ ] **Step 3: Update all references from `mockLevyRoll` to `levyRoll` in the agent/trustee view**

In the agent/trustee view section (after `const canEdit = user?.role === 'agent'`), change:

```tsx
  const collected = mockLevyRoll.filter(a => a.status === 'paid').length
  const overdue = mockLevyRoll.filter(a => a.status === 'overdue').length
  const totalCollected = mockLevyRoll.reduce((sum, a) => sum + a.paid_cents, 0)
```

to:

```tsx
  const collected = levyRoll.filter(a => a.status === 'paid').length
  const overdue = levyRoll.filter(a => a.status === 'overdue').length
  const totalCollected = levyRoll.reduce((sum, a) => sum + a.paid_cents, 0)
```

And in the levy roll table, change `{mockLevyRoll.map(` to `{levyRoll.map(`.

- [ ] **Step 4: Add the modal and close the component**

Just before the final closing `</div>` and `}` of the component, add:

```tsx
      {reconcileOpen && (
        <ReconcileModal
          levyAccounts={levyRoll}
          periodLabel={mockLevyPeriod.label}
          onConfirm={handleReconcileConfirm}
          onClose={() => setReconcileOpen(false)}
        />
      )}
```

- [ ] **Step 5: Verify the full file compiles — run the dev server briefly**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app
npm run build 2>&1 | tail -20
```

Expected: no TypeScript errors. Fix any type issues before continuing.

- [ ] **Step 6: Commit**

```bash
git add app/app/\[schemeId\]/levy/page.tsx
git commit -m "feat: wire bank statement reconciliation into levy page"
```

---

## Part B — AI Scheme Copilot

### Task 6: Create the AI copilot API route

**Files:**
- Create: `app/api/copilot/route.ts`

This route accepts a POST request with `{ message, history }`, builds a system prompt from all mock scheme data, and streams Claude's response back as plain text.

- [ ] **Step 1: Create the directory and file**

```bash
mkdir -p /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/app/api/copilot
```

- [ ] **Step 2: Write the route**

```typescript
// app/api/copilot/route.ts
import { NextRequest } from 'next/server'
import Anthropic from '@anthropic-ai/sdk'
import { mockPortfolio, mockScheme, mockUnits } from '@/lib/mock/scheme'
import { mockLevyRoll, mockLevyPeriod, mockCollectionTrend } from '@/lib/mock/levy'
import { mockMaintenanceRequests } from '@/lib/mock/maintenance'
import { mockBudgetLines, mockReserveFund } from '@/lib/mock/financials'
import { mockAgmMeetings, mockResolutions } from '@/lib/mock/agm'
import { mockNotices } from '@/lib/mock/communications'
import { mockMembers } from '@/lib/mock/members'

const SYSTEM_PROMPT = `You are StrataHQ Copilot — an intelligent assistant for property managing agents in South Africa.

You help agents manage sectional title schemes (body corporates) by answering questions about levy collections, maintenance jobs, AGM status, financials, compliance, and communications. You also draft professional letters, notices, and reports when asked.

Rules:
- Always be specific. Cite actual names, amounts, percentages, and dates from the data below.
- Be concise. Use bullet points for lists. Answer in 3–6 sentences unless a longer response (like a drafted letter) is explicitly requested.
- For financial amounts, use South African Rand (R) formatting.
- When drafting letters or notices, produce formal, professional documents ready to send.
- Today's date is ${new Date().toLocaleDateString('en-ZA', { day: 'numeric', month: 'long', year: 'numeric' })}.

LIVE PORTFOLIO DATA:
${JSON.stringify({
  portfolio: mockPortfolio,
  currentScheme: mockScheme,
  units: mockUnits,
  levyPeriod: mockLevyPeriod,
  levyRoll: mockLevyRoll,
  collectionTrend: mockCollectionTrend,
  maintenanceRequests: mockMaintenanceRequests,
  budgetLines: mockBudgetLines,
  reserveFund: mockReserveFund,
  agmMeetings: mockAgmMeetings,
  resolutions: mockResolutions,
  notices: mockNotices,
  members: mockMembers,
}, null, 2)}`

export async function POST(request: NextRequest) {
  const apiKey = process.env.ANTHROPIC_API_KEY
  if (!apiKey) {
    return new Response(
      'ANTHROPIC_API_KEY is not set. Add it to your .env.local file to use the AI Copilot.',
      { status: 503, headers: { 'Content-Type': 'text/plain' } }
    )
  }

  const { message, history } = await request.json() as {
    message: string
    history: Array<{ role: 'user' | 'assistant'; content: string }>
  }

  const client = new Anthropic({ apiKey })

  const stream = await client.messages.create({
    model: 'claude-haiku-4-5-20251001',
    max_tokens: 1024,
    stream: true,
    system: SYSTEM_PROMPT,
    messages: [...history, { role: 'user', content: message }],
  })

  const encoder = new TextEncoder()
  const readable = new ReadableStream({
    async start(controller) {
      try {
        for await (const event of stream) {
          if (
            event.type === 'content_block_delta' &&
            event.delta.type === 'text_delta'
          ) {
            controller.enqueue(encoder.encode(event.delta.text))
          }
        }
      } finally {
        controller.close()
      }
    },
  })

  return new Response(readable, {
    headers: {
      'Content-Type': 'text/plain; charset=utf-8',
      'Cache-Control': 'no-cache',
    },
  })
}
```

- [ ] **Step 3: Check which mock files export what**

Verify the named exports from each mock file match what is imported in the route. Run:

```bash
grep -n "^export" /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/lib/mock/financials.ts
grep -n "^export" /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/lib/mock/agm.ts
grep -n "^export" /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/lib/mock/communications.ts
grep -n "^export" /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/lib/mock/members.ts
```

Adjust import names in the route if any export names differ from what is listed above.

- [ ] **Step 4: Create .env.local if it doesn't exist**

```bash
test -f /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/.env.local || \
  echo "ANTHROPIC_API_KEY=your-key-here" > /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/.env.local
```

The user must replace `your-key-here` with their actual API key.

- [ ] **Step 5: Commit**

```bash
git add app/api/copilot/route.ts
git commit -m "feat: add AI copilot streaming API route"
```

---

### Task 7: Create the Copilot floating chat component

**Files:**
- Create: `components/Copilot.tsx`

A floating button (bottom-right) that expands into a chat panel with streaming AI responses.

- [ ] **Step 1: Create the component**

```typescript
// components/Copilot.tsx
'use client'
import { useState, useRef, useEffect } from 'react'
import { useMockAuth } from '@/lib/mock-auth'

interface Message {
  role: 'user' | 'assistant'
  content: string
}

const SUGGESTED_QUESTIONS = [
  'Which schemes have the lowest collection rates?',
  'Are there any overdue maintenance SLAs?',
  'What is the total outstanding levy debt across all schemes?',
  'Draft an overdue levy reminder for Unit 3A',
]

export default function Copilot() {
  const { user } = useMockAuth()
  const [open, setOpen] = useState(false)
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [streaming, setStreaming] = useState(false)
  const [streamingContent, setStreamingContent] = useState('')
  const [error, setError] = useState<string | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  // Only available for agents and trustees
  if (!user || user.role === 'resident') return null

  // eslint-disable-next-line react-hooks/rules-of-hooks
  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 100)
    }
  }, [open])

  // eslint-disable-next-line react-hooks/rules-of-hooks
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streamingContent])

  async function sendMessage(text: string) {
    if (!text.trim() || streaming) return

    const userMessage: Message = { role: 'user', content: text.trim() }
    const history = messages
    setMessages(prev => [...prev, userMessage])
    setInput('')
    setStreaming(true)
    setStreamingContent('')
    setError(null)

    try {
      const response = await fetch('/api/copilot', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: text.trim(), history }),
      })

      if (!response.ok || !response.body) {
        const errText = await response.text()
        setError(errText || 'Something went wrong. Please try again.')
        setStreaming(false)
        return
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let accumulated = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        accumulated += decoder.decode(value, { stream: true })
        setStreamingContent(accumulated)
      }

      setMessages(prev => [...prev, { role: 'assistant', content: accumulated }])
      setStreamingContent('')
    } catch {
      setError('Network error — check your connection and try again.')
    } finally {
      setStreaming(false)
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      sendMessage(input)
    }
  }

  const showWelcome = messages.length === 0 && !streaming

  return (
    <>
      {/* Floating button */}
      <button
        onClick={() => setOpen(o => !o)}
        className={[
          'fixed bottom-6 right-6 z-50 w-12 h-12 rounded-full shadow-lg flex items-center justify-center transition-all duration-200',
          open ? 'bg-ink text-surface scale-90' : 'bg-ink text-surface hover:scale-105',
        ].join(' ')}
        aria-label="Open AI Copilot"
      >
        {open ? (
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
            <line x1="3" y1="3" x2="13" y2="13" /><line x1="13" y1="3" x2="3" y2="13" />
          </svg>
        ) : (
          <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5">
            <path d="M9 1C4.58 1 1 4.13 1 8c0 1.74.68 3.33 1.8 4.57L2 17l4.64-1.55C7.6 15.8 8.28 16 9 16c4.42 0 8-3.13 8-7s-3.58-7-8-7z" strokeLinejoin="round" />
          </svg>
        )}
      </button>

      {/* Chat panel */}
      {open && (
        <div className="fixed bottom-22 right-6 z-50 w-[380px] max-h-[520px] bg-surface border border-border rounded-xl shadow-2xl flex flex-col overflow-hidden"
          style={{ bottom: '80px' }}>

          {/* Header */}
          <div className="flex items-center gap-3 px-4 py-3 border-b border-border flex-shrink-0 bg-surface">
            <div className="w-7 h-7 rounded-full bg-ink flex items-center justify-center flex-shrink-0">
              <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="white" strokeWidth="1.5">
                <path d="M6 1C3.24 1 1 2.9 1 5.2c0 1.11.5 2.12 1.3 2.85L2 11l3-1c.3.1.63.16.97.16 2.76 0 5-1.9 5-4.24C11 3.52 8.76 1.5 6 1.5z" strokeLinejoin="round" />
              </svg>
            </div>
            <div>
              <p className="text-[13px] font-semibold text-ink">StrataHQ Copilot</p>
              <p className="text-[11px] text-muted">Ask anything about your schemes</p>
            </div>
            {messages.length > 0 && (
              <button
                onClick={() => { setMessages([]); setError(null) }}
                className="ml-auto text-[11px] text-muted hover:text-ink transition-colors"
              >
                Clear
              </button>
            )}
          </div>

          {/* Messages */}
          <div className="flex-1 overflow-y-auto px-4 py-3 space-y-3">

            {showWelcome && (
              <div>
                <p className="text-[13px] text-muted mb-3">Try asking:</p>
                <div className="space-y-2">
                  {SUGGESTED_QUESTIONS.map(q => (
                    <button
                      key={q}
                      onClick={() => sendMessage(q)}
                      className="w-full text-left text-[12px] text-ink bg-page border border-border rounded-lg px-3 py-2 hover:border-accent hover:bg-accent-dim transition-colors"
                    >
                      {q}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {messages.map((m, i) => (
              <div key={i} className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                <div className={[
                  'max-w-[85%] rounded-xl px-3 py-2 text-[13px] leading-relaxed whitespace-pre-wrap',
                  m.role === 'user'
                    ? 'bg-ink text-surface rounded-br-sm'
                    : 'bg-page border border-border text-ink rounded-bl-sm',
                ].join(' ')}>
                  {m.content}
                </div>
              </div>
            ))}

            {streaming && streamingContent && (
              <div className="flex justify-start">
                <div className="max-w-[85%] rounded-xl rounded-bl-sm px-3 py-2 text-[13px] leading-relaxed whitespace-pre-wrap bg-page border border-border text-ink">
                  {streamingContent}
                  <span className="inline-block w-1 h-3 bg-accent ml-0.5 animate-pulse" />
                </div>
              </div>
            )}

            {streaming && !streamingContent && (
              <div className="flex justify-start">
                <div className="bg-page border border-border rounded-xl rounded-bl-sm px-4 py-3">
                  <div className="flex gap-1">
                    <span className="w-1.5 h-1.5 bg-muted rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
                    <span className="w-1.5 h-1.5 bg-muted rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
                    <span className="w-1.5 h-1.5 bg-muted rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
                  </div>
                </div>
              </div>
            )}

            {error && (
              <div className="text-[12px] text-red bg-red-bg border border-red rounded-lg px-3 py-2">
                {error}
              </div>
            )}

            <div ref={messagesEndRef} />
          </div>

          {/* Input */}
          <div className="flex-shrink-0 border-t border-border px-3 py-2">
            <div className="flex items-end gap-2">
              <textarea
                ref={inputRef}
                value={input}
                onChange={e => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Ask anything…"
                rows={1}
                disabled={streaming}
                className="flex-1 resize-none bg-page border border-border rounded-lg px-3 py-2 text-[13px] text-ink placeholder-muted outline-none focus:border-accent transition-colors min-h-[36px] max-h-[120px] disabled:opacity-50"
                style={{ height: 'auto' }}
                onInput={e => {
                  const el = e.currentTarget
                  el.style.height = 'auto'
                  el.style.height = `${Math.min(el.scrollHeight, 120)}px`
                }}
              />
              <button
                onClick={() => sendMessage(input)}
                disabled={!input.trim() || streaming}
                className="flex-shrink-0 w-8 h-8 bg-ink text-surface rounded-lg flex items-center justify-center hover:bg-ink/80 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <path d="M1 7h12M8 2l5 5-5 5" strokeLinecap="round" strokeLinejoin="round" />
                </svg>
              </button>
            </div>
            <p className="text-[10px] text-muted mt-1.5 text-center">Enter to send · Shift+Enter for new line</p>
          </div>
        </div>
      )}
    </>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add components/Copilot.tsx
git commit -m "feat: add AI Copilot floating chat component with streaming"
```

---

### Task 8: Mount Copilot in scheme and agent layouts

**Files:**
- Modify: `app/app/[schemeId]/layout.tsx`
- Modify: `app/agent/layout.tsx`

- [ ] **Step 1: Add Copilot to the scheme layout**

Open `app/app/[schemeId]/layout.tsx`. Add the import and mount the component inside `<ToastProvider>` after `<AppShell>`:

```tsx
'use client'
import { useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useMockAuth } from '@/lib/mock-auth'
import AppShell from '@/components/AppShell'
import Sidebar, { type SidebarRole } from '@/components/Sidebar'
import { ToastProvider } from '@/lib/toast'
import Copilot from '@/components/Copilot'

export default function SchemeLayout({ children }: { children: React.ReactNode }) {
  const { user, loading } = useMockAuth()
  const router = useRouter()
  const params = useParams()
  const schemeId = params.schemeId as string

  useEffect(() => {
    if (loading) return
    if (!user) { router.replace('/auth/login'); return }
    if (user.schemeId !== schemeId) router.replace(`/app/${user.schemeId}`)
  }, [user, loading, router, schemeId])

  if (loading || !user) return null

  const sidebarRole: SidebarRole =
    user.role === 'agent' ? 'agent-scheme' :
    user.role === 'trustee' ? 'trustee' : 'resident'

  const headerLabel =
    user.role === 'resident'
      ? `Unit ${user.unitIdentifier ?? '?'} · ${user.schemeName}`
      : user.schemeName

  return (
    <ToastProvider>
      <AppShell
        headerLabel={headerLabel}
        sidebar={
          <Sidebar
            role={sidebarRole}
            headerLabel={headerLabel}
            schemeId={schemeId}
          />
        }
      >
        {children}
      </AppShell>
      <Copilot />
    </ToastProvider>
  )
}
```

- [ ] **Step 2: Read agent layout and add Copilot there too**

Read `app/agent/layout.tsx`, then add `import Copilot from '@/components/Copilot'` and mount `<Copilot />` after the `<AppShell>` close tag (inside whatever wrapper is there).

- [ ] **Step 3: Build to verify no TypeScript errors**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app
npm run build 2>&1 | tail -30
```

Expected: clean build with no type errors. If there are import errors (e.g., wrong export names from mock files), fix them in `app/api/copilot/route.ts`.

- [ ] **Step 4: Commit**

```bash
git add app/app/\[schemeId\]/layout.tsx app/agent/layout.tsx
git commit -m "feat: mount AI Copilot in scheme and agent layouts"
```

---

## Final Verification

- [ ] **Start dev server and test reconciliation**

```bash
npm run dev
```

1. Log in as agent → navigate to a scheme → Levy & Payments
2. Click "Reconcile statement"
3. Click "Load sample statement (demo)"
4. Verify Step 2 shows 7 transactions, most marked "Auto" (green), Unit 6C as "Review" (amber), BANK CHARGES filtered out
5. Click "Review X matches →"
6. Verify Step 3 shows changes to be applied — statuses and amounts correct
7. Click "Apply reconciliation"
8. Verify toast fires and levy roll table updates (3A still overdue, others updated)

- [ ] **Test AI Copilot**

1. Set `ANTHROPIC_API_KEY` in `.env.local` (real key)
2. Click the floating chat button (bottom-right)
3. Click a suggested question — verify streaming response with real scheme data
4. Log in as resident — verify floating button is NOT shown

- [ ] **Final commit**

```bash
git add .
git commit -m "feat: bank statement reconciliation and AI copilot — complete"
```
