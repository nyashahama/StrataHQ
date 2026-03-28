import type { Config } from 'tailwindcss'

const config: Config = {
  darkMode: 'class',
  content: [
    './app/**/*.{ts,tsx}',
    './components/**/*.{ts,tsx}',
    './hooks/**/*.{ts,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        page:    'var(--color-page)',
        surface: 'var(--color-surface)',
        ink: {
          DEFAULT: 'var(--color-ink)',
          2: 'var(--color-ink-2)',
        },
        muted: {
          DEFAULT: 'var(--color-muted)',
          2: 'var(--color-muted-2)',
        },
        border: {
          DEFAULT: 'var(--color-border)',
          2: 'var(--color-border-2)',
        },
        accent: {
          DEFAULT: 'var(--color-accent)',
          bg: 'var(--color-accent-bg)',
          dim: 'var(--color-accent-dim)',
        },
        green: {
          DEFAULT: 'var(--color-green)',
          bg: 'var(--color-green-bg)',
        },
        red: {
          DEFAULT: 'var(--color-red)',
          bg: 'var(--color-red-bg)',
        },
        yellowbg:         'var(--color-yellowbg)',
        amber:            'var(--color-amber)',
        'sidebar-header': 'var(--color-sidebar-header)',
        'hover-subtle':   'var(--color-hover-subtle)',
      },
      fontFamily: {
        serif: ['var(--font-lora)', 'Georgia', 'serif'],
        sans:  ['var(--font-dm-sans)', 'system-ui', 'sans-serif'],
      },
      borderRadius: {
        sm:      '6px',
        DEFAULT: '8px',
        lg:      '12px',
      },
      boxShadow: {
        sm:      '0 1px 3px rgba(55,53,47,0.08), 0 1px 2px rgba(55,53,47,0.04)',
        DEFAULT: '0 4px 16px rgba(55,53,47,0.10), 0 1px 4px rgba(55,53,47,0.06)',
        lg:      '0 16px 48px rgba(55,53,47,0.14), 0 4px 12px rgba(55,53,47,0.08)',
      },
      maxWidth: {
        container: '1080px',
      },
    },
  },
  plugins: [],
}

export default config
