import Link from 'next/link'
import LogoIcon from './LogoIcon'

export default function Nav() {
  return (
    <nav className="sticky top-0 z-50 border-b border-border nav-surface backdrop-blur-[12px]">
      <div className="max-w-container mx-auto px-container flex items-center justify-between gap-6 h-14">
        {/* Logo */}
        <Link href="/" className="flex items-center gap-[9px] flex-shrink-0 no-underline">
          <div className="w-7 h-7 bg-sidebar-header rounded-sm grid place-items-center">
            <LogoIcon className="w-[14px] h-[14px] fill-white" />
          </div>
          <span className="font-sans text-[15px] font-semibold text-ink tracking-[-0.01em]">
            StrataHQ
          </span>
        </Link>

        {/* Nav links — hidden on mobile */}
        <ul className="hidden sm:flex items-center gap-1 list-none">
          {[
            { label: 'Features', href: '#features' },
            { label: 'Modules', href: '#modules' },
            { label: "Who it's for", href: '#roles' },
            { label: 'Pricing', href: '#pricing' },
          ].map(({ label, href }) => (
            <li key={label}>
              <Link
                href={href}
                className="px-[10px] py-[11px] rounded-sm text-[14px] text-muted no-underline font-normal
                  hover:text-ink hover:bg-hover-subtle transition-colors duration-150"
              >
                {label}
              </Link>
            </li>
          ))}
        </ul>

        {/* Actions */}
        <div className="flex items-center gap-2">
          <Link
            href="/auth/login"
            className="px-[14px] py-[10px] text-[14px] font-medium text-ink-2 bg-transparent
              border border-border-2 rounded hover:bg-hover-subtle transition-colors duration-150 no-underline"
          >
            Log in
          </Link>
          <Link
            href="/early-access"
            className="px-4 py-[10px] text-[14px] font-medium text-white bg-accent
              border border-accent rounded hover:opacity-90 transition-opacity duration-150 no-underline"
          >
            Request early access
          </Link>
        </div>
      </div>
    </nav>
  )
}
