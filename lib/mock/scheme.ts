// lib/mock/scheme.ts

export interface Scheme {
  id: string
  name: string
  org_id: string
  unit_count: number
  address: string
  created_at: string
}

export interface Unit {
  id: string
  scheme_id: string
  identifier: string      // e.g. "1A", "4B"
  owner_name: string
  floor: number
  section_value: number   // participation quota as percentage e.g. 4.17
}

export const mockScheme: Scheme = {
  id: 'scheme-001',
  name: 'Sunridge Heights',
  org_id: 'org-001',
  unit_count: 24,
  address: '14 Sunridge Drive, Claremont, Cape Town, 7708',
  created_at: '2024-01-15T08:00:00Z',
}

// 8 representative units used across all module mock data
export const mockUnits: Unit[] = [
  { id: 'unit-1a', scheme_id: 'scheme-001', identifier: '1A', owner_name: 'Henderson, T.', floor: 1, section_value: 4.17 },
  { id: 'unit-2b', scheme_id: 'scheme-001', identifier: '2B', owner_name: 'Molefe, S.',     floor: 2, section_value: 4.17 },
  { id: 'unit-3a', scheme_id: 'scheme-001', identifier: '3A', owner_name: 'van der Berg, L.', floor: 3, section_value: 4.17 },
  { id: 'unit-4b', scheme_id: 'scheme-001', identifier: '4B', owner_name: 'Naidoo, R.',    floor: 4, section_value: 4.17 },
  { id: 'unit-5a', scheme_id: 'scheme-001', identifier: '5A', owner_name: 'Khumalo, B.',   floor: 5, section_value: 4.17 },
  { id: 'unit-6c', scheme_id: 'scheme-001', identifier: '6C', owner_name: 'Abrahams, J.', floor: 6, section_value: 4.17 },
  { id: 'unit-7b', scheme_id: 'scheme-001', identifier: '7B', owner_name: 'Petersen, M.', floor: 7, section_value: 4.17 },
  { id: 'unit-8a', scheme_id: 'scheme-001', identifier: '8A', owner_name: 'Dlamini, S.',  floor: 8, section_value: 4.17 },
]

// Agent portfolio — all 3 schemes managed by org-001
export interface PortfolioScheme {
  id: string
  name: string
  unit_count: number
  address: string
  levy_collection_pct: number   // e.g. 91
  open_maintenance_count: number
  health: 'good' | 'fair' | 'poor'
}

export const mockPortfolio: PortfolioScheme[] = [
  {
    id: 'scheme-001',
    name: 'Sunridge Heights',
    unit_count: 24,
    address: 'Claremont, Cape Town',
    levy_collection_pct: 91,
    open_maintenance_count: 7,
    health: 'good',
  },
  {
    id: 'scheme-002',
    name: 'Bayside Manor',
    unit_count: 12,
    address: 'Sea Point, Cape Town',
    levy_collection_pct: 75,
    open_maintenance_count: 4,
    health: 'fair',
  },
  {
    id: 'scheme-003',
    name: 'The Palms Estate',
    unit_count: 16,
    address: 'Kenilworth, Cape Town',
    levy_collection_pct: 58,
    open_maintenance_count: 12,
    health: 'poor',
  },
]
