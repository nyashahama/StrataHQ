'use client'

import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import { useMockAuth } from '@/lib/mock-auth'
import ThemeToggle from '@/components/ThemeToggle'

export type SidebarRole = 'agent-portfolio' | 'agent-scheme' | 'trustee' | 'resident'

export interface SidebarMembership {
  scheme_id: string
  scheme_name: string
}

interface SidebarProps {
  role: SidebarRole
  headerLabel: string
  schemeId?: string
  allMemberships?: SidebarMembership[]
}

// --- Inline SVG Icons ---

function GridIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="1" y="1" width="5" height="5" rx="0.5" />
      <rect x="8" y="1" width="5" height="5" rx="0.5" />
      <rect x="1" y="8" width="5" height="5" rx="0.5" />
      <rect x="8" y="8" width="5" height="5" rx="0.5" />
    </svg>
  )
}

function ListIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <line x1="1" y1="3.5" x2="13" y2="3.5" />
      <line x1="1" y1="7" x2="13" y2="7" />
      <line x1="1" y1="10.5" x2="13" y2="10.5" />
    </svg>
  )
}

function MailIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="1" y="3" width="12" height="8" rx="1" />
      <polyline points="1,3 7,8 13,3" />
    </svg>
  )
}

function CreditCardIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="1" y="3" width="12" height="8" rx="1" />
      <line x1="1" y1="6" x2="13" y2="6" />
    </svg>
  )
}

function WrenchIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M9 2a3 3 0 0 1 0 4.5L4.5 11a1.5 1.5 0 0 1-2-2L7 4.5A3 3 0 0 1 9 2z" />
    </svg>
  )
}

function VoteIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="1" y="1" width="12" height="12" rx="1" />
      <polyline points="4,7 6,9 10,5" />
    </svg>
  )
}

function MegaphoneIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M2 5h2l5-3v10L4 9H2a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1z" />
      <line x1="4" y1="9" x2="4" y2="12" />
    </svg>
  )
}

function FolderIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M1 4a1 1 0 0 1 1-1h3l1.5 2H12a1 1 0 0 1 1 1v5a1 1 0 0 1-1 1H2a1 1 0 0 1-1-1V4z" />
    </svg>
  )
}

function ChartIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <polyline points="1,11 4,7 7,9 10,4 13,6" />
    </svg>
  )
}

function UsersIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="5" cy="5" r="2.5" />
      <path d="M1 13a4 4 0 0 1 8 0" />
      <circle cx="10.5" cy="4.5" r="2" />
      <path d="M12.5 11.5a3 3 0 0 0-4-1" />
    </svg>
  )
}

function GearIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="7" cy="7" r="2" />
      <path d="M7 1v2M7 11v2M1 7h2M11 7h2M3.2 3.2l1.4 1.4M9.4 9.4l1.4 1.4M10.8 3.2l-1.4 1.4M4.6 9.4l-1.4 1.4" />
    </svg>
  )
}

function WhatsAppIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M7 1C3.69 1 1 3.69 1 7c0 1.18.32 2.28.87 3.23L1 13l2.87-.85A5.96 5.96 0 007 13c3.31 0 6-2.69 6-6S10.31 1 7 1z" strokeLinejoin="round" />
      <path d="M5 5.5c0-.28.22-.5.5-.5h.5c.28 0 .5.22.5.5v.5c0 .83-.67 1.5-1.5 1.5M9 8.5c-.28 0-.5-.22-.5-.5V7.5c0-.28.22-.5.5-.5h.5c.28 0 .5.22.5.5C10 8.33 9.33 9 8.5 9" strokeLinecap="round" />
    </svg>
  )
}

function ShieldIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M7 1L2 3v4c0 3 2.24 4.95 5 6 2.76-1.05 5-3 5-6V3L7 1z" strokeLinejoin="round" />
      <polyline points="4.5,7 6.5,9 9.5,5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

// --- Nav item type ---

interface NavItem {
  icon: React.ReactNode
  label: string
  href: string
  exactMatch?: boolean
}

// --- Nav config per role ---

function getNavItems(role: SidebarRole, schemeId?: string): NavItem[] {
  if (role === 'agent-portfolio') {
    return [
      { icon: <GridIcon />, label: 'Portfolio overview', href: '/agent', exactMatch: true },
      { icon: <ListIcon />, label: 'All schemes', href: '/agent/schemes' },
      { icon: <MailIcon />, label: 'Invitations', href: '/agent/invitations' },
    ]
  }

  const base = `/app/${schemeId}`

  if (role === 'agent-scheme') {
    return [
      { icon: <GridIcon />, label: 'Overview', href: base, exactMatch: true },
      { icon: <CreditCardIcon />, label: 'Levy & Payments', href: `${base}/levy` },
      { icon: <WrenchIcon />, label: 'Maintenance', href: `${base}/maintenance` },
      { icon: <VoteIcon />, label: 'AGM & Voting', href: `${base}/agm` },
      { icon: <MegaphoneIcon />, label: 'Communications', href: `${base}/communications` },
      { icon: <FolderIcon />, label: 'Documents', href: `${base}/documents` },
      { icon: <ChartIcon />, label: 'Financials', href: `${base}/financials` },
      { icon: <UsersIcon />, label: 'Members', href: `${base}/members` },
      { icon: <WhatsAppIcon />, label: 'WhatsApp', href: `${base}/whatsapp` },
      { icon: <ShieldIcon />, label: 'Compliance', href: `${base}/compliance` },
    ]
  }

  if (role === 'trustee') {
    return [
      { icon: <GridIcon />, label: 'Overview', href: base, exactMatch: true },
      { icon: <CreditCardIcon />, label: 'Levy & Payments', href: `${base}/levy` },
      { icon: <WrenchIcon />, label: 'Maintenance', href: `${base}/maintenance` },
      { icon: <VoteIcon />, label: 'AGM & Voting', href: `${base}/agm` },
      { icon: <MegaphoneIcon />, label: 'Communications', href: `${base}/communications` },
      { icon: <FolderIcon />, label: 'Documents', href: `${base}/documents` },
      { icon: <ChartIcon />, label: 'Financials', href: `${base}/financials` },
      { icon: <UsersIcon />, label: 'Members', href: `${base}/members` },
      { icon: <WhatsAppIcon />, label: 'WhatsApp', href: `${base}/whatsapp` },
      { icon: <ShieldIcon />, label: 'Compliance', href: `${base}/compliance` },
    ]
  }

  // resident
  return [
    { icon: <GridIcon />, label: 'Overview', href: base, exactMatch: true },
    { icon: <CreditCardIcon />, label: 'My Levy', href: `${base}/levy` },
    { icon: <WrenchIcon />, label: 'Maintenance', href: `${base}/maintenance` },
    { icon: <VoteIcon />, label: 'AGM & Voting', href: `${base}/agm` },
    { icon: <MegaphoneIcon />, label: 'Notices', href: `${base}/communications` },
    { icon: <FolderIcon />, label: 'Documents', href: `${base}/documents` },
    { icon: <ChartIcon />, label: 'Financials', href: `${base}/financials` },
    { icon: <WhatsAppIcon />, label: 'WhatsApp', href: `${base}/whatsapp` },
  ]
}

function getBottomItem(role: SidebarRole, schemeId?: string): NavItem {
  if (role === 'agent-portfolio') {
    return { icon: <GearIcon />, label: 'Settings', href: '/agent/settings' }
  }
  if (role === 'resident') {
    return { icon: <GearIcon />, label: 'My Profile', href: `/app/${schemeId}/profile` }
  }
  return { icon: <GearIcon />, label: 'Settings', href: `/app/${schemeId}/settings` }
}

function getHeaderSmallLabel(role: SidebarRole): string {
  if (role === 'agent-portfolio') return 'Organisation'
  if (role === 'resident') return 'My unit'
  return 'Scheme'
}

// --- NavLink component ---

function NavLink({ item, pathname }: { item: NavItem; pathname: string }) {
  const isActive = item.exactMatch
    ? pathname === item.href
    : pathname.startsWith(item.href)

  return (
    <Link
      href={item.href}
      className={
        `flex items-center gap-2 px-3 py-[7px] text-[12px] border-l-2 transition-colors ` +
        (isActive
          ? 'bg-accent-dim text-accent font-medium border-accent'
          : 'text-muted hover:text-ink hover:bg-hover-subtle border-transparent')
      }
    >
      {item.icon}
      {item.label}
    </Link>
  )
}

// --- Main Sidebar component ---

export default function Sidebar({ role, headerLabel, schemeId, allMemberships }: SidebarProps) {
  const pathname = usePathname()
  const router = useRouter()
  const { logout } = useMockAuth()

  const navItems = getNavItems(role, schemeId)
  const bottomItem = getBottomItem(role, schemeId)
  const smallLabel = getHeaderSmallLabel(role)

  const showSchemeSwitcher =
    role === 'trustee' &&
    allMemberships &&
    allMemberships.length >= 2

  return (
    <aside className="w-[200px] flex-shrink-0 h-full flex flex-col bg-page border-r border-border">
      {/* Header */}
      <div className="bg-sidebar-header px-3 py-3 flex-shrink-0">
        <p
          className="uppercase tracking-wide text-white/40 mb-0.5"
          style={{ fontSize: '9px' }}
        >
          {smallLabel}
        </p>
        <p className="text-[12px] text-white font-semibold leading-tight">
          {headerLabel}
        </p>
        {showSchemeSwitcher && (
          <select
            className="mt-1.5 text-[11px] text-white/40 bg-transparent border-none outline-none w-full cursor-pointer"
            value={schemeId}
            onChange={(e) => router.push(`/app/${e.target.value}`)}
          >
            {allMemberships!.map((m) => (
              <option key={m.scheme_id} value={m.scheme_id} className="text-ink bg-surface">
                {m.scheme_name}
              </option>
            ))}
          </select>
        )}
      </div>

      {/* Nav items */}
      <nav className="flex-1 overflow-y-auto py-2">
        {navItems.map((item) => (
          <NavLink key={item.href} item={item} pathname={pathname} />
        ))}
      </nav>

      {/* Bottom section */}
      <div className="border-t border-border py-2 flex-shrink-0">
        <NavLink item={bottomItem} pathname={pathname} />
        <ThemeToggle />
        <button
          onClick={() => { logout(); router.push('/auth/login') }}
          className="flex items-center gap-2 px-3 py-[7px] text-[12px] text-muted hover:text-ink hover:bg-hover-subtle w-full transition-colors"
        >
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
            <path d="M5 2H3a1 1 0 0 0-1 1v8a1 1 0 0 0 1 1h2M9 10l3-3-3-3M12 7H5" />
          </svg>
          Log out
        </button>
      </div>
    </aside>
  )
}
