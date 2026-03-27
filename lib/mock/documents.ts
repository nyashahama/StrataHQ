// lib/mock/documents.ts

export interface SchemeDocument {
  id: string
  scheme_id: string
  name: string
  file_type: 'pdf' | 'docx' | 'xlsx' | 'jpg' | 'png'
  category: 'rules' | 'minutes' | 'insurance' | 'financial' | 'other'
  uploaded_at: string
  uploaded_by_name: string
  size_bytes: number
}

export const mockDocuments: SchemeDocument[] = [
  { id: 'doc-001', scheme_id: 'scheme-001', name: 'Conduct Rules — Sunridge Heights',  file_type: 'pdf',  category: 'rules',     uploaded_at: '2024-02-01T10:00:00Z', uploaded_by_name: 'Acme Property Management', size_bytes: 524288  },
  { id: 'doc-002', scheme_id: 'scheme-001', name: 'AGM Minutes — November 2025',       file_type: 'pdf',  category: 'minutes',   uploaded_at: '2025-11-21T14:00:00Z', uploaded_by_name: 'Acme Property Management', size_bytes: 286720  },
  { id: 'doc-003', scheme_id: 'scheme-001', name: 'AGM Minutes — November 2024',       file_type: 'pdf',  category: 'minutes',   uploaded_at: '2024-11-18T14:00:00Z', uploaded_by_name: 'Acme Property Management', size_bytes: 311296  },
  { id: 'doc-004', scheme_id: 'scheme-001', name: 'Insurance Certificate 2025–2026',   file_type: 'pdf',  category: 'insurance', uploaded_at: '2025-01-05T09:00:00Z', uploaded_by_name: 'Acme Property Management', size_bytes: 204800  },
  { id: 'doc-005', scheme_id: 'scheme-001', name: 'Approved Budget 2026',              file_type: 'xlsx', category: 'financial', uploaded_at: '2025-11-21T14:30:00Z', uploaded_by_name: 'Acme Property Management', size_bytes: 45056   },
  { id: 'doc-006', scheme_id: 'scheme-001', name: 'Management Agreement',              file_type: 'pdf',  category: 'other',     uploaded_at: '2024-01-20T09:00:00Z', uploaded_by_name: 'Acme Property Management', size_bytes: 638976  },
  { id: 'doc-007', scheme_id: 'scheme-001', name: '10-Year Maintenance Plan',          file_type: 'pdf',  category: 'financial', uploaded_at: '2024-06-10T11:00:00Z', uploaded_by_name: 'Acme Property Management', size_bytes: 1048576 },
]
