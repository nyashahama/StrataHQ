// lib/mock/compliance.ts
// STSMA = Sectional Titles Schemes Management Act (South Africa)
// Compliance items for scheme-001 (Sunridge Heights)

export type ComplianceStatus = 'compliant' | 'at-risk' | 'non-compliant'
export type ComplianceCategory = 'financial' | 'governance' | 'administrative' | 'insurance'

export interface ComplianceItem {
  id: string
  category: ComplianceCategory
  title: string
  requirement: string   // what the STSMA requires
  status: ComplianceStatus
  detail: string        // current state
  action: string        // what to do if not compliant / at-risk
  due_date: string | null
}

export const mockComplianceItems: ComplianceItem[] = [
  // ── Financial ──────────────────────────────────────────────────────────
  {
    id: 'c-01',
    category: 'financial',
    title: 'Annual financial statements',
    requirement: 'Financial statements must be prepared and approved within 4 months of financial year end.',
    status: 'compliant',
    detail: 'Statements for FY2024/25 approved on 31 Aug 2025.',
    action: 'No action required.',
    due_date: null,
  },
  {
    id: 'c-02',
    category: 'financial',
    title: 'Reserve fund minimum contribution',
    requirement: 'Reserve fund must receive a minimum 10% of total levy income annually (STSMA Reg 2).',
    status: 'at-risk',
    detail: 'Reserve fund at R 67 420 — 67% of the recommended R 100 000 target. Current contribution rate is 6.2%.',
    action: 'Increase reserve fund levy contribution to at least 10% before the next budget cycle.',
    due_date: '2025-12-01',
  },
  {
    id: 'c-03',
    category: 'financial',
    title: 'CSOS levy payment',
    requirement: 'Annual Community Schemes Ombud Service (CSOS) levy must be paid.',
    status: 'compliant',
    detail: 'CSOS levy of R 1 850 paid on 15 Apr 2025. Next due Apr 2026.',
    action: 'No action required.',
    due_date: null,
  },
  {
    id: 'c-04',
    category: 'financial',
    title: 'Approved annual budget',
    requirement: 'Budget must be approved by trustees and presented at AGM before the start of the financial year.',
    status: 'compliant',
    detail: 'FY2025/26 budget of R 420 000 approved at AGM on 14 Oct 2025.',
    action: 'No action required.',
    due_date: null,
  },

  // ── Governance ─────────────────────────────────────────────────────────
  {
    id: 'c-05',
    category: 'governance',
    title: 'Annual General Meeting held',
    requirement: 'AGM must be held within 4 months of the financial year end each year (STSMA Reg 17).',
    status: 'compliant',
    detail: 'AGM held on 14 Oct 2025 with 62% quorum achieved.',
    action: 'No action required.',
    due_date: null,
  },
  {
    id: 'c-06',
    category: 'governance',
    title: 'Trustee meeting minutes',
    requirement: 'Minutes of all trustee meetings must be recorded and kept on file.',
    status: 'at-risk',
    detail: 'Q3 2025 (Jul–Sep) trustee meeting minutes have not been uploaded to the document vault.',
    action: 'Upload Q3 trustee meeting minutes to the document vault.',
    due_date: '2025-10-31',
  },
  {
    id: 'c-07',
    category: 'governance',
    title: 'Trustee election on record',
    requirement: 'Trustee committee elections must be held at the AGM and results recorded.',
    status: 'compliant',
    detail: 'Three trustees elected at Oct 2025 AGM. Election recorded in meeting minutes.',
    action: 'No action required.',
    due_date: null,
  },

  // ── Administrative ─────────────────────────────────────────────────────
  {
    id: 'c-08',
    category: 'administrative',
    title: 'Scheme rules registered with CSOS',
    requirement: 'Management and conduct rules must be filed with the Community Schemes Ombud Service.',
    status: 'non-compliant',
    detail: 'Scheme rules have not been registered with CSOS. This is a legal requirement under STSMA s10.',
    action: 'Submit scheme rules to CSOS via the CSOS online portal (www.csos.org.za). Filing fee: R 400.',
    due_date: '2025-11-30',
  },
  {
    id: 'c-09',
    category: 'administrative',
    title: '10-year maintenance plan',
    requirement: 'A written maintenance, repair, and replacement plan for common property must exist (STSMA Reg 3).',
    status: 'non-compliant',
    detail: 'No formal maintenance plan on record. The reserve fund target cannot be properly justified without one.',
    action: 'Appoint a qualified assessor to compile a 10-year maintenance plan. Budget approximately R 3 500–8 000.',
    due_date: '2026-02-28',
  },
  {
    id: 'c-10',
    category: 'administrative',
    title: 'Conduct rules in place',
    requirement: 'Scheme must have registered conduct rules governing owner and resident behaviour.',
    status: 'compliant',
    detail: 'Conduct rules adopted and distributed to all residents. On file in document vault.',
    action: 'No action required.',
    due_date: null,
  },

  // ── Insurance ──────────────────────────────────────────────────────────
  {
    id: 'c-11',
    category: 'insurance',
    title: 'Building insurance in force',
    requirement: 'Body corporate must insure all buildings to full replacement value (STSMA s3(1)(b)).',
    status: 'compliant',
    detail: 'Santam policy #BC-2025-9834. Buildings insured to R 18.5M replacement value. Renewed Mar 2025.',
    action: 'No action required.',
    due_date: null,
  },
  {
    id: 'c-12',
    category: 'insurance',
    title: 'Replacement valuation current',
    requirement: 'Insurance replacement valuation must be updated at least every 3 years.',
    status: 'at-risk',
    detail: 'Last valuation: August 2022 (3 years 2 months ago). May no longer reflect replacement cost.',
    action: 'Commission an updated replacement valuation from a registered quantity surveyor. Approx R 2 500.',
    due_date: '2025-12-31',
  },
]

// ── Score calculation ──────────────────────────────────────────────────────

const POINTS: Record<ComplianceStatus, number> = {
  compliant: 10,
  'at-risk': 5,
  'non-compliant': 0,
}

export function calcComplianceScore(items: ComplianceItem[]): number {
  const total = items.length * 10
  const earned = items.reduce((sum, item) => sum + POINTS[item.status], 0)
  return Math.round((earned / total) * 100)
}

export const COMPLIANCE_CATEGORIES: { key: ComplianceCategory; label: string }[] = [
  { key: 'financial',      label: 'Financial' },
  { key: 'governance',     label: 'Governance' },
  { key: 'administrative', label: 'Administrative' },
  { key: 'insurance',      label: 'Insurance' },
]
