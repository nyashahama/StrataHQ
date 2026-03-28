// lib/mock/bank-statement.ts
// Sample bank statement CSV for demo/testing the reconciliation feature.
//
// Maps to the mock levy roll in lib/mock/levy.ts:
//   1A (Henderson)  → R2 450 full payment, HIGH confidence (UNIT1A in description)
//   2B (Molefe)     → R1 200 partial payment, HIGH confidence (UNIT 2B in description)
//   5A (Khumalo)    → R2 450 full payment, HIGH confidence (5A in description)
//   4B (Naidoo)     → R2 450 full payment, HIGH confidence (UNIT4B in description)
//   7B (Petersen)   → R2 450 full payment, HIGH confidence (7B in description)
//   8A (Dlamini)    → R2 450 full payment, HIGH confidence (UNIT 8A in description)
//   6C (Abrahams)   → R500 partial payment, MEDIUM confidence (surname in description)
//   3A (van der Berg) → no entry → UNMATCHED (stays overdue)
//   BANK CHARGES    → negative amount → filtered out by parser

import { parseBankStatementCSV } from '@/lib/reconcile'

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
