export type ComplianceStatus = "compliant" | "at-risk" | "non-compliant";
export type ComplianceCategory =
  | "financial"
  | "governance"
  | "administrative"
  | "insurance";

export interface ComplianceItem {
  due_date?: string | null;
  id: string;
  scheme_id: string;
  category: ComplianceCategory;
  title: string;
  requirement: string;
  status: ComplianceStatus;
  detail: string;
  action: string;
  assessed_at: string;
}

export interface ComplianceDashboard {
  items: ComplianceItem[];
  role: string;
  score: number;
  total: number;
  compliant_count: number;
  at_risk_count: number;
  non_compliant_count: number;
  last_assessed_at: string;
}

export const COMPLIANCE_CATEGORIES: {
  key: ComplianceCategory;
  label: string;
}[] = [
  { key: "financial", label: "Financial" },
  { key: "governance", label: "Governance" },
  { key: "administrative", label: "Administrative" },
  { key: "insurance", label: "Insurance" },
];
