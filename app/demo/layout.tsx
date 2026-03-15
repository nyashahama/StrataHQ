import type { Metadata } from 'next'

export const metadata: Metadata = {
  title: 'StrataHQ — Interactive Demo',
  description:
    'Explore a fully interactive demo of the StrataHQ platform. Choose a role — Managing Agent, Trustee, or Resident — and try every feature with realistic live data.',
}

export default function DemoLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="fixed inset-0 overflow-hidden bg-[#F7F6F3]">
      {children}
    </div>
  )
}
