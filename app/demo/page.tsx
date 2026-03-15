import Link from 'next/link'
import LogoIcon from '@/components/LogoIcon'

export default function DemoPage() {
  return (
    <div className="flex flex-col h-full">
      {/* Exit bar */}
      <div className="flex-shrink-0 h-9 bg-ink flex items-center justify-between px-4 gap-4 z-10">
        <div className="flex items-center gap-2">
          <LogoIcon className="w-3 h-3 fill-white opacity-60" />
          <span className="text-[11px] font-medium text-[rgba(247,246,243,0.5)] tracking-[0.04em] uppercase">
            Interactive Demo
          </span>
          <span className="text-[rgba(247,246,243,0.2)] text-[11px]">·</span>
          <span className="text-[11px] text-[rgba(247,246,243,0.4)]">
            All data is simulated — every action does something real
          </span>
        </div>
        <Link
          href="/"
          className="flex items-center gap-1.5 text-[11px] font-medium text-[rgba(247,246,243,0.5)]
            hover:text-[rgba(247,246,243,0.85)] transition-colors duration-150 no-underline"
        >
          ← Back to site
        </Link>
      </div>

      {/* Demo iframe — fills remaining height */}
      <iframe
        src="/demo-app.html"
        title="StrataHQ Interactive Demo"
        className="flex-1 w-full border-0"
        allow="same-origin"
      />
    </div>
  )
}
