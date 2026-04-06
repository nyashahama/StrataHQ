'use client'

import Link from 'next/link'
import { useState } from 'react'
import LogoIcon from './LogoIcon'

const navLinks = [
  { label: 'Features', href: '#features' },
  { label: 'Modules', href: '#modules' },
  { label: "Who it's for", href: '#roles' },
  { label: 'Pricing', href: '#pricing' },
]

export default function Nav() {
  const [open, setOpen] = useState(false)

  return (
    <>
      <nav className="fixed top-4 left-1/2 -translate-x-1/2 z-50 w-[calc(100%-32px)] max-w-[1080px]">
        <div className="landing-nav rounded-xl px-5 py-3 flex items-center justify-between gap-6">
          {/* Logo */}
          <Link href="/" className="flex items-center gap-[9px] flex-shrink-0 no-underline">
            <div className="w-7 h-7 bg-[var(--color-hero-accent)] rounded-md grid place-items-center">
              <LogoIcon className="w-[14px] h-[14px] fill-white" />
            </div>
            <span className="font-sans text-[15px] font-semibold text-white tracking-[-0.01em]">
              StrataHQ
            </span>
          </Link>

          {/* Nav links — hidden on mobile */}
          <ul className="hidden md:flex items-center gap-0.5 list-none">
            {navLinks.map(({ label, href }) => (
              <li key={label}>
                <Link
                  href={href}
                  className="px-3 py-2 rounded-lg text-[13px] text-white/50 no-underline font-normal
                    hover:text-white hover:bg-white/[0.06] transition-all duration-200"
                >
                  {label}
                </Link>
              </li>
            ))}
          </ul>

          {/* Actions */}
          <div className="hidden md:flex items-center gap-2">
            <Link
              href="/auth/login"
              className="px-4 py-2 text-[13px] font-medium text-white/70 bg-transparent
                border border-white/10 rounded-lg hover:text-white hover:border-white/20 hover:bg-white/[0.04] transition-all duration-200 no-underline"
            >
              Log in
            </Link>
            <Link
              href="/early-access"
              className="px-4 py-2 text-[13px] font-medium text-white bg-[var(--color-hero-accent)]
                rounded-lg hover:brightness-110 transition-all duration-200 no-underline"
            >
              Request early access
            </Link>
          </div>

          {/* Mobile hamburger */}
          <button
            onClick={() => setOpen(!open)}
            className="md:hidden flex flex-col justify-center items-center gap-[5px] w-8 h-8 bg-transparent border-none cursor-pointer"
            aria-label="Toggle menu"
          >
            <span className={`block w-5 h-[1.5px] bg-white/70 rounded-full transition-all duration-300 ${open ? 'rotate-45 translate-y-[6.5px]' : ''}`} />
            <span className={`block w-5 h-[1.5px] bg-white/70 rounded-full transition-all duration-300 ${open ? 'opacity-0 scale-x-0' : ''}`} />
            <span className={`block w-5 h-[1.5px] bg-white/70 rounded-full transition-all duration-300 ${open ? '-rotate-45 -translate-y-[6.5px]' : ''}`} />
          </button>
        </div>
      </nav>

      {/* Mobile menu overlay */}
      <div
        className={`fixed inset-0 z-40 mobile-menu-overlay flex flex-col items-center justify-center gap-6 transition-all duration-300 md:hidden ${
          open ? 'opacity-100 pointer-events-auto' : 'opacity-0 pointer-events-none'
        }`}
      >
        {navLinks.map(({ label, href }) => (
          <Link
            key={label}
            href={href}
            onClick={() => setOpen(false)}
            className="text-[20px] font-serif font-semibold text-white/80 no-underline hover:text-white transition-colors duration-200"
          >
            {label}
          </Link>
        ))}
        <div className="flex flex-col gap-3 mt-4 w-56">
          <Link
            href="/auth/login"
            onClick={() => setOpen(false)}
            className="w-full text-center px-5 py-3 text-[14px] font-medium text-white/70
              border border-white/15 rounded-lg hover:bg-white/[0.05] transition-all duration-200 no-underline"
          >
            Log in
          </Link>
          <Link
            href="/early-access"
            onClick={() => setOpen(false)}
            className="w-full text-center px-5 py-3 text-[14px] font-medium text-white bg-[var(--color-hero-accent)]
              rounded-lg hover:brightness-110 transition-all duration-200 no-underline"
          >
            Request early access
          </Link>
        </div>
      </div>
    </>
  )
}
