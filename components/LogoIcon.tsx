interface LogoIconProps {
  className?: string
}

export default function LogoIcon({ className }: LogoIconProps) {
  return (
    <svg viewBox="0 0 16 16" className={className} aria-hidden>
      <path d="M2 3h5v5H2V3zm7 0h5v5H9V3zM2 10h5v4H2v-4zm7 1h2v-2h2v2h2v2h-2v2H9v-2H7v-2h2v-2z" />
    </svg>
  )
}
