import Link from 'next/link'
import LogoIcon from './LogoIcon'

interface FooterLink {
  label: string
  href: string
}

const footerLinks: Record<string, FooterLink[]> = {
  Product: [
    { label: 'Features', href: '/#features' },
    { label: 'Modules', href: '/#modules' },
    { label: 'Pricing', href: '/#pricing' },
    { label: 'Changelog', href: '/#roles' },
  ],
  Resources: [
    { label: 'Documentation', href: '/#features' },
    { label: 'STSMA guide', href: '/#problem' },
    { label: 'Blog', href: '/early-access' },
    { label: 'Help centre', href: '/auth/login' },
  ],
  Company: [
    { label: 'About', href: '/#roles' },
    { label: 'Contact', href: '/#problem' },
    { label: 'Privacy policy', href: '/auth/login' },
    { label: 'Terms', href: '/auth/login' },
  ],
}

export default function Footer() {
  return (
    <footer className="border-t border-border bg-page pt-[clamp(48px,7vw,64px)] pb-[clamp(28px,4vw,36px)]">
      <div className="max-w-[1080px] mx-auto px-container">
        {/* Top grid */}
        <div className="grid grid-cols-2 sm:grid-cols-[1.5fr_1fr_1fr_1fr] gap-[clamp(24px,4vw,48px)] mb-12">
          {/* Brand */}
          <div className="col-span-2 sm:col-span-1">
            <Link href="/" className="flex items-center gap-[9px] no-underline mb-3">
              <div className="w-7 h-7 bg-ink rounded-md grid place-items-center">
                <LogoIcon className="w-[14px] h-[14px] fill-white" />
              </div>
              <span className="font-sans text-[15px] font-semibold text-ink tracking-[-0.01em]">
                StrataHQ
              </span>
            </Link>
            <p className="text-[13px] text-muted leading-[1.65] max-w-[220px]">
              Body corporate management software built for South Africa&apos;s
              sectional title schemes.
            </p>
          </div>

          {/* Link columns */}
          {Object.entries(footerLinks).map(([heading, links]) => (
            <div key={heading}>
              <div className="text-[11px] font-semibold text-ink tracking-[0.06em] uppercase mb-4">
                {heading}
              </div>
              <ul className="flex flex-col">
                {links.map((link) => (
                  <li key={link.label}>
                    <Link
                      href={link.href}
                      className="inline-block text-[13px] text-muted no-underline hover:text-ink transition-colors duration-200 py-[7px]"
                    >
                      {link.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        {/* Bottom bar */}
        <div className="border-t border-border pt-6 flex flex-wrap justify-between gap-3 text-[12px] text-muted-2">
          <span>&copy; 2026 StrataHQ. Built in South Africa</span>
          <span>Compliant with STSMA Act 8 of 2011</span>
        </div>
      </div>
    </footer>
  )
}
