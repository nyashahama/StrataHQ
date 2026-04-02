// lib/reconcile.ts
import type { LevyAccountInfo } from '@/lib/levy'

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
  account: LevyAccountInfo | null
  confidence: MatchConfidence
  match_reason: string
  // what applying this match would do:
  new_paid_cents: number
  new_status: LevyAccountInfo['status']
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

  const dateIdx = cols.findIndex(c => c.includes('date'))
  const descIdx = cols.findIndex(c =>
    c.includes('desc') || c.includes('narr') || c.includes('reference') || c.includes('transaction')
  )
  const amtIdx = cols.findIndex(c => c.includes('amount') || c === 'amt' || c.includes('credit'))
  const balIdx = cols.findIndex(c => c.includes('balance') || c === 'bal')

  if (dateIdx === -1 || descIdx === -1 || amtIdx === -1) {
    throw new Error(
      'Could not detect required columns (Date, Description, Amount) in this CSV. Check the file format.'
    )
  }

  const transactions: ParsedTransaction[] = []

  for (let i = 1; i < lines.length; i++) {
    const row = splitCSVRow(lines[i])
    if (row.length <= Math.max(dateIdx, descIdx, amtIdx)) continue

    const amtRaw = row[amtIdx].replace(/[",\s]/g, '')
    const amount_cents = Math.round(parseFloat(amtRaw) * 100)

    if (isNaN(amount_cents) || amount_cents <= 0) continue // skip debits / non-numeric

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
    if (ch === '"') {
      inQuotes = !inQuotes
    } else if (ch === ',' && !inQuotes) {
      result.push(current)
      current = ''
    } else {
      current += ch
    }
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
 * 2. MEDIUM — exact/partial amount + owner surname found in description
 * 3. LOW    — exact amount only (ambiguous if multiple units have same levy)
 * 4. UNMATCHED — nothing found
 *
 * Each account can only be matched once (first/highest confidence wins).
 */
export function matchTransactions(
  transactions: ParsedTransaction[],
  accounts: LevyAccountInfo[]
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
        `UNIT ${uid}`,
        `UNIT${uid}`,
        ` ${uid} `,
        ` ${uid},`,
        `${uid} LEVY`,
        `${uid}LEVY`,
        `-${uid}-`,
        `/${uid}/`,
      ]
      if (patterns.some(p => descUpper.includes(p))) {
        usedAccountIds.add(account.id)
        return buildMatch(
          tx,
          account,
          'high',
          `Unit identifier "${account.unit_identifier}" found in description`
        )
      }
    }

    // 2. Try surname + amount match
    for (const account of accounts) {
      if (usedAccountIds.has(account.id)) continue
      const surname = extractSurname(account.owner_name).toUpperCase()
      if (surname.length < 3) continue // too short to be meaningful
      const amountMatch =
        tx.amount_cents === account.amount_cents || tx.amount_cents < account.amount_cents
      if (amountMatch && descUpper.includes(surname)) {
        usedAccountIds.add(account.id)
        return buildMatch(tx, account, 'medium', `Surname "${surname}" + amount match`)
      }
    }

    // 3. Try exact amount match (last resort)
    const exactAmountAccounts = accounts.filter(
      a => !usedAccountIds.has(a.id) && tx.amount_cents === a.amount_cents && a.status !== 'paid'
    )
    if (exactAmountAccounts.length === 1) {
      usedAccountIds.add(exactAmountAccounts[0].id)
      return buildMatch(
        tx,
        exactAmountAccounts[0],
        'low',
        'Exact amount match (no name/unit reference found)'
      )
    }

    return {
      transaction: tx,
      account: null,
      confidence: 'unmatched' as const,
      match_reason: 'No matching levy account found',
      new_paid_cents: 0,
      new_status: 'pending' as LevyAccountInfo['status'],
    }
  })
}

function extractSurname(ownerName: string): string {
  // Format is "Surname, F." or "van der Berg, L." — take everything before the comma
  return ownerName.split(',')[0].trim()
}

function buildMatch(
  tx: ParsedTransaction,
  account: LevyAccountInfo,
  confidence: MatchConfidence,
  match_reason: string
): ReconcileMatch {
  const new_paid_cents = account.paid_cents + tx.amount_cents
  const new_status: LevyAccountInfo['status'] =
    new_paid_cents >= account.amount_cents
      ? 'paid'
      : new_paid_cents > 0
        ? 'partial'
        : 'overdue'

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
    matched_high: matches.filter(m => m.confidence === 'high').length,
    matched_medium: matches.filter(m => m.confidence === 'medium').length,
    matched_low: matches.filter(m => m.confidence === 'low').length,
    unmatched: matches.filter(m => m.confidence === 'unmatched').length,
    total_amount_cents: matches.reduce((s, m) => s + m.transaction.amount_cents, 0),
    matched_amount_cents: matches
      .filter(m => m.account)
      .reduce((s, m) => s + m.transaction.amount_cents, 0),
  }
}
